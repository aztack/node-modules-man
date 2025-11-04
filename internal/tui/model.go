package tui

import (
	"context"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"node-module-man/internal/deleter"
	"node-module-man/internal/scanner"
	"node-module-man/pkg/utils"
)

type status int

const (
	statusScanning status = iota
	statusReady
	statusConfirm
	statusDeleting
	statusDone
)

type model struct {
	path      string
	opts      scanner.Options
	sp        spinner.Model
	startedAt time.Time

	st        status
	results   []scanner.ResultItem
	totalSize int64
	err       error

	// list view (custom rendering, not using bubbles/list)
	items        []item
	cursor       int
	scrollOffset int
	sortBy       string // "size" or "path"
	sortReverse  bool
	selectedSize int64
	filterText   string
	filtering    bool

	// confirm/delete state
	delCh        chan tea.Msg
	delTotal     int
	delCompleted int
	delLastPath  string
	delFreed     int64
	delFailures  []deleter.Failure

	// deletion control
	delCancel func()
	dryRun    bool

	// scanning stream
	scanCh     chan tea.Msg
	scanCancel func()
	scanning   bool

	// terminal size
	termW int
	termH int

	// help panel
	showHelp bool
}

func newModel(path string, opts scanner.Options, dryRun bool) model {
	sp := spinner.New()
	sp.Spinner = spinner.Dot
	m := model{
		path:        path,
		opts:        opts,
		sp:          sp,
		startedAt:   time.Now(),
		st:          statusScanning,
		dryRun:      dryRun,
		items:       []item{},
		cursor:      0,
		sortBy:      "size",
		sortReverse: true,
	}
	// start streaming scan
	ch := make(chan tea.Msg)
	m.scanCh = ch
	m.scanning = true
	go func() {
		ctx, cancel := context.WithCancel(context.Background())
		m.scanCancel = cancel
		out, errCh := scanner.ScanNodeModulesStream(ctx, path, opts)
		for r := range out {
			ch <- scanItemMsg{item: r}
		}
		// final error (may be nil)
		var err error
		if e, ok := <-errCh; ok {
			err = e
		}
		ch <- scanCompleteMsg{err: err}
		close(ch)
	}()
	return m
}

// public entry
func Run(path string, opts scanner.Options, dryRun bool) error {
	m := newModel(path, opts, dryRun)
	p := tea.NewProgram(m)
	_, err := p.Run()
	return err
}

// messages
type scanDoneMsg struct {
	results   []scanner.ResultItem
	totalSize int64
	err       error
}

func scanCmd(path string, opts scanner.Options) tea.Cmd {
	return func() tea.Msg {
		// Run sync scan in background
		res, total, err := scanner.ScanNodeModules(context.Background(), path, opts)
		return scanDoneMsg{results: res, totalSize: total, err: err}
	}
}

func (m model) Init() tea.Cmd {
	return tea.Batch(m.sp.Tick, m.waitScanMsg())
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "esc", "ctrl+c":
			if m.st == statusConfirm {
				m.st = statusReady
				return m, nil
			}
			if m.st == statusDeleting && m.delCancel != nil {
				m.delCancel()
				// keep waiting for done message
				return m, m.waitDeleteMsg()
			}
			if m.st == statusScanning && m.scanCancel != nil {
				// Gracefully cancel scanning before quitting
				m.scanCancel()
			}
			return m, tea.Quit
		case "?":
			// Toggle help panel
			m.showHelp = !m.showHelp
			return m, nil
		case "enter", "d":
			if m.st == statusReady {
				if m.selectedCount() == 0 {
					return m, nil
				}
				m.st = statusConfirm
				return m, nil
			}
		case "y":
			if m.st == statusConfirm {
				return m.startDeletion()
			}
		case "n":
			if m.st == statusConfirm {
				m.st = statusReady
				return m, nil
			}
		case "up", "k":
			if m.st == statusReady && m.cursor > 0 {
				m.cursor--
				m.adjustScroll()
				return m, nil
			}
		case "down", "j":
			if m.st == statusReady && m.cursor < len(m.items)-1 {
				m.cursor++
				m.adjustScroll()
				return m, nil
			}
		case " ":
			if m.st == statusReady {
				m.toggleSelected()
				return m, nil
			}
		case "s":
			if m.st == statusReady {
				m.toggleSortField()
				m.applySort()
				return m, nil
			}
		case "r":
			if m.st == statusReady {
				m.sortReverse = !m.sortReverse
				m.applySort()
				return m, nil
			}
		}
	case tea.WindowSizeMsg:
		m.termW, m.termH = msg.Width, msg.Height
		return m, nil

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.sp, cmd = m.sp.Update(msg)
		// keep polling scan/delete channels too
		if m.st == statusScanning {
			return m, tea.Batch(cmd, m.waitScanMsg())
		}
		if m.st == statusDeleting {
			return m, tea.Batch(cmd, m.waitDeleteMsg())
		}
		return m, cmd
	case scanItemMsg:
		m.appendResult(msg.item)
		return m, m.waitScanMsg()
	case scanCompleteMsg:
		m.err = msg.err
		m.scanning = false
		m.st = statusReady
		return m, nil
	case delProgressMsg:
		m.delCompleted = msg.completed
		m.delLastPath = msg.path
		if msg.err == nil {
			// Freed are already tracked in summary later; optimistic update here for UX
		}
		return m, m.waitDeleteMsg()
	case delDoneMsg:
		m.delFailures = msg.summary.Failures
		m.delFreed = msg.summary.Freed
		// remove successes from list and results
		succ := msg.summary.Successes
		m.removeDeleted(succ)
		m.selectedSize = 0
		m.totalSize -= m.delFreed
		m.st = statusDone
		return m, nil
	}
	if m.st == statusDeleting {
		// keep spinner ticking and keep waiting for delete messages
		return m, m.waitDeleteMsg()
	}
	return m, nil
}

func (m model) View() string {
	switch m.st {
	case statusScanning:
		base := m.headerText() + m.renderList()
		if m.showHelp {
			base += "\n" + m.helpText()
		}
		return base
	case statusReady:
		base := m.headerText() + m.renderList()
		if m.showHelp {
			base += "\n" + m.helpText()
		}
		return base
	case statusConfirm:
		cnt := m.selectedCount()
		size := utils.HumanizeBytes(m.selectedSize)
		return fmt.Sprintf("Confirm delete %d node_modules, freeing ~%s? (y/N)\nPress y to confirm, n/esc to cancel.\n", cnt, size)
	case statusDeleting:
		mode := ""
		if m.dryRun {
			mode = " [dry-run]"
		}
		return fmt.Sprintf("Deleting%s... %s\nProgress: %d/%d\nLast: %s\nPress q to cancel.\n", mode, m.sp.View(), m.delCompleted, m.delTotal, m.delLastPath)
	case statusDone:
		mode := ""
		if m.dryRun {
			mode = " (dry-run; no files removed)"
		}
		s := fmt.Sprintf("Delete complete%s. Freed %s. Failures: %d\n", mode, utils.HumanizeBytes(m.delFreed), len(m.delFailures))
		for _, f := range m.delFailures {
			s += fmt.Sprintf(" - %s: %v\n", f.Path, f.Err)
		}
		s += "Press q to quit or any key to return.\n"
		return s
	default:
		return ""
	}
}

// list helpers
type item struct {
	path string
	disp string
	size int64
	err  error
	sel  bool
}

// Custom list rendering - no bubbles/list component
func (m *model) renderList() string {
	if len(m.items) == 0 {
		return "No node_modules found.\n"
	}

	var b strings.Builder
	headerLines := strings.Count(m.headerText(), "\n") + 1
	visibleHeight := m.termH - headerLines - 1
	if visibleHeight < 3 {
		visibleHeight = 3
	}

	// Render visible items
	start := m.scrollOffset
	end := start + visibleHeight
	if end > len(m.items) {
		end = len(m.items)
	}

	for i := start; i < end; i++ {
		it := m.items[i]

		// Build line with styling
		var prefix string
		if i == m.cursor {
			prefix = cursorStyle.Render(">") + " "
		} else {
			prefix = "  "
		}

		var mark string
		if it.sel {
			mark = markSelectedStyle.Render("[x]")
		} else {
			mark = markStyle.Render("[ ]")
		}

		sizeStr := sizeColorStyle(it.size).Render(utils.HumanizeBytesCompact(it.size))

		var pathStr string
		if it.sel {
			pathStr = pathStyleSelected.Render(it.disp)
		} else {
			pathStr = it.disp
		}

		// Build final line
		line := prefix + mark + " " + sizeStr + " " + pathStr

		b.WriteString(line + "\n")
	}

	return b.String()
}

func (m *model) adjustScroll() {
	headerLines := strings.Count(m.headerText(), "\n") + 1
	visibleHeight := m.termH - headerLines - 1
	if visibleHeight < 3 {
		visibleHeight = 3
	}

	// Scroll down if cursor is below visible area
	if m.cursor >= m.scrollOffset+visibleHeight {
		m.scrollOffset = m.cursor - visibleHeight + 1
	}

	// Scroll up if cursor is above visible area
	if m.cursor < m.scrollOffset {
		m.scrollOffset = m.cursor
	}
}

func (m *model) toggleSelected() {
	if m.cursor < 0 || m.cursor >= len(m.items) {
		return
	}
	m.items[m.cursor].sel = !m.items[m.cursor].sel
	if m.items[m.cursor].sel {
		m.selectedSize += m.items[m.cursor].size
	} else {
		m.selectedSize -= m.items[m.cursor].size
	}
}

func (m *model) toggleSortField() {
	if m.sortBy == "size" {
		m.sortBy = "path"
	} else {
		m.sortBy = "size"
	}
}

func (m *model) applySort() {
	sort.Slice(m.items, func(i, j int) bool {
		if m.sortBy == "path" {
			if m.sortReverse {
				return m.items[i].disp > m.items[j].disp
			}
			return m.items[i].disp < m.items[j].disp
		}
		if m.sortReverse {
			return m.items[i].size > m.items[j].size
		}
		return m.items[i].size < m.items[j].size
	})
}

// streaming scan wiring
type scanItemMsg struct{ item scanner.ResultItem }
type scanCompleteMsg struct{ err error }

func (m *model) waitScanMsg() tea.Cmd {
	if m.scanCh == nil {
		return nil
	}
	return func() tea.Msg {
		msg, ok := <-m.scanCh
		if !ok {
			return nil
		}
		return msg
	}
}

func (m *model) appendResult(r scanner.ResultItem) {
	m.results = append(m.results, r)
	if r.Err == nil {
		m.totalSize += r.Size
	}
	// Append to items array and sort
	m.items = append(m.items, item{
		path: r.Path,
		disp: m.displayPath(r.Path),
		size: r.Size,
		err:  r.Err,
	})
	m.applySort()
}

func (m *model) displayPath(p string) string {
	// Show path relative to scan root when possible, keep full if failure
	if rel, err := filepath.Rel(m.path, p); err == nil && rel != "." {
		return rel
	}
	return p
}

func (m *model) headerText() string {
	switch m.st {
	case statusScanning:
		elapsed := time.Since(m.startedAt).Round(time.Millisecond)
		return fmt.Sprintf("Scanning... %s  Found: %d  Total: %s  Elapsed: %s\nPress ? for help\n\n", m.sp.View(), len(m.results), utils.HumanizeBytes(m.totalSize), elapsed)
	case statusReady:
		return fmt.Sprintf("Found: %d  Total: %s  Selected: %s  | Keys: ? help, ↑↓ move, space select, s sort, r reverse, / filter, d/enter delete, q quit\n\n",
			len(m.results), utils.HumanizeBytes(m.totalSize), utils.HumanizeBytes(m.selectedSize))
	default:
		return ""
	}
}

func (m *model) helpText() string {
	// Minimal help panel
	lines := []string{
		"Help (press ? to close):",
		"  ↑/k, ↓/j  Move cursor",
		"  space     Toggle selection",
		"  s         Toggle sort field (size/path)",
		"  r         Reverse sort",
		"  d/enter   Delete selected",
		"  q/esc     Quit (cancels delete; cancels scan)",
	}
	return lipgloss.NewStyle().Padding(0, 1).Border(lipgloss.NormalBorder()).Render(strings.Join(lines, "\n"))
}

var (
	cursorStyle       = lipgloss.NewStyle().Foreground(lipgloss.Color("99"))            // purple
	markStyle         = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))           // gray
	markSelectedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("42")).Bold(true) // green
	sizeStyle         = lipgloss.NewStyle().Foreground(lipgloss.Color("45"))            // cyan
	pathStyleNormal   = lipgloss.NewStyle()
	pathStyleSelected = lipgloss.NewStyle().Foreground(lipgloss.Color("42"))             // green
	highlightStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("227")).Bold(true) // yellow
	headerStyle       = lipgloss.NewStyle().Bold(true)
)

// Choose color for size: dark red > light red > orange > yellow > green > light gray > dark gray
func sizeColorStyle(b int64) lipgloss.Style {
	// thresholds in bytes
	const (
		MB = 1024 * 1024
		GB = 1024 * MB
	)
	switch {
	case b >= 8*GB:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("160")) // dark red
	case b >= 4*GB:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("196")) // light red
	case b >= 2*GB:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("208")) // orange
	case b >= 1*GB:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("226")) // yellow
	case b >= 256*MB:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("46")) // green
	case b >= 64*MB:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("250")) // light gray
	default:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("240")) // dark gray
	}
}

// deletion wiring
type delProgressMsg struct {
	completed int
	total     int
	path      string
	err       error
}
type delDoneMsg struct{ summary deleter.Summary }

func (m *model) startDeletion() (tea.Model, tea.Cmd) {
	m.st = statusDeleting
	m.sp = spinner.New()
	m.sp.Spinner = spinner.Dot
	m.delCompleted = 0
	targets := m.selectedTargets()
	m.delTotal = len(targets)
	ch := make(chan tea.Msg)
	m.delCh = ch

	// launch worker goroutine
	go func() {
		// Bridge deleter progress into tea messages
		pch := make(chan deleter.Progress, 16)
		// run deletion in background
		var sum deleter.Summary
		done := make(chan struct{})
		go func() {
			ctx, cancel := context.WithCancel(context.Background())
			m.delCancel = cancel
			sum = deleter.DeleteTargets(ctx, targets, m.opts.Concurrency, pch, m.dryRun)
			close(done)
		}()
		for {
			select {
			case p, ok := <-pch:
				if !ok {
					p = deleter.Progress{Completed: m.delTotal, Total: m.delTotal}
				}
				ch <- delProgressMsg{completed: p.Completed, total: p.Total, path: p.Path, err: p.Err}
				if p.Completed >= p.Total && p.Total > 0 {
					// wait for summary
				}
			case <-done:
				ch <- delDoneMsg{summary: sum}
				close(ch)
				return
			}
		}
	}()

	return m, tea.Batch(m.sp.Tick, m.waitDeleteMsg())
}

func (m *model) waitDeleteMsg() tea.Cmd {
	if m.delCh == nil {
		return nil
	}
	return func() tea.Msg {
		msg, ok := <-m.delCh
		if !ok {
			return nil
		}
		return msg
	}
}

func (m *model) selectedCount() int {
	c := 0
	for _, it := range m.items {
		if it.sel {
			c++
		}
	}
	return c
}

func (m *model) selectedTargets() []deleter.Target {
	var out []deleter.Target
	for _, it := range m.items {
		if it.sel {
			out = append(out, deleter.Target{Path: it.path, Size: it.size})
		}
	}
	return out
}

func (m *model) removeDeleted(succ []deleter.Target) {
	if len(succ) == 0 {
		return
	}
	// build set
	rm := make(map[string]struct{}, len(succ))
	for _, t := range succ {
		rm[t.Path] = struct{}{}
	}
	// filter items
	kept := make([]item, 0, len(m.items))
	for _, it := range m.items {
		if _, ok := rm[it.path]; ok {
			continue
		}
		kept = append(kept, it)
	}
	m.items = kept
	// Adjust cursor if necessary
	if m.cursor >= len(m.items) && len(m.items) > 0 {
		m.cursor = len(m.items) - 1
	}
	// filter results slice as well
	newRes := m.results[:0]
	for _, r := range m.results {
		if _, ok := rm[r.Path]; ok {
			continue
		}
		newRes = append(newRes, r)
	}
	m.results = newRes
}
