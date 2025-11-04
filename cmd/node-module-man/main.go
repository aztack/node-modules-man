package main

import (
    "context"
    "encoding/json"
    "flag"
    "fmt"
    "os"
    "path/filepath"
    "runtime"
    "time"

    "node-module-man/internal/scanner"
    ui "node-module-man/internal/tui"
    "node-module-man/pkg/utils"
)

type multiFlag []string
func (m *multiFlag) String() string { return fmt.Sprint([]string(*m)) }
func (m *multiFlag) Set(v string) error { *m = append(*m, v); return nil }

func main() {
    var (
        root        string
        jsonOut     bool
        concurrency int
        maxDepth    int
        useTUI      bool
        dryRun      bool
        excludes    multiFlag
        followLinks bool
    )

    flag.StringVar(&root, "path", ".", "Root path to scan")
    flag.StringVar(&root, "p", ".", "Alias of --path")
    flag.BoolVar(&jsonOut, "json", false, "Output JSON instead of table")
    flag.IntVar(&concurrency, "concurrency", runtime.NumCPU(), "Concurrency for size calculations")
    flag.IntVar(&concurrency, "c", runtime.NumCPU(), "Alias of --concurrency")
    flag.IntVar(&maxDepth, "max-depth", -1, "Max depth for directory walk (-1 for unlimited)")
    flag.IntVar(&maxDepth, "m", -1, "Alias of --max-depth")
    flag.BoolVar(&useTUI, "tui", true, "Run interactive TUI (default)")
    flag.BoolVar(&useTUI, "t", true, "Alias of --tui")
    flag.BoolVar(&dryRun, "dry-run", false, "Do not delete anything; simulate deletion in TUI")
    flag.BoolVar(&dryRun, "d", false, "Alias of --dry-run")
    flag.Var(&excludes, "exclude", "Glob pattern to exclude (can repeat). Matches full path or basename.")
    flag.Var(&excludes, "x", "Alias of --exclude")
    flag.BoolVar(&followLinks, "follow-symlinks", false, "Follow symlinked directories when computing sizes (pnpm-style)")
    flag.BoolVar(&followLinks, "L", false, "Alias of --follow-symlinks")
    flag.Parse()

    absRoot, err := filepath.Abs(root)
    if err != nil {
        fmt.Fprintf(os.Stderr, "failed to resolve path: %v\n", err)
        os.Exit(2)
    }

    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()

    opts := scanner.Options{
        Concurrency:   concurrency,
        MaxDepth:      maxDepth,
        FollowSymlink: followLinks,
        Excludes:      []string(excludes),
    }

    if useTUI {
        if err := ui.Run(absRoot, opts, dryRun); err != nil {
            fmt.Fprintf(os.Stderr, "tui error: %v\n", err)
            os.Exit(1)
        }
        return
    }

    start := time.Now()

    results, totalSize, scanErr := scanner.ScanNodeModules(ctx, absRoot, opts)
    if scanErr != nil {
        // We'll still print what we have but exit non-zero.
        fmt.Fprintf(os.Stderr, "scan completed with errors: %v\n", scanErr)
    }

    if jsonOut {
        enc := json.NewEncoder(os.Stdout)
        enc.SetIndent("", "  ")
        payload := struct {
            Root      string               `json:"root"`
            TotalSize int64                `json:"totalSize"`
            Results   []scanner.ResultItem `json:"results"`
            Duration  string               `json:"duration"`
        }{Root: absRoot, TotalSize: totalSize, Results: results, Duration: time.Since(start).String()}
        if err := enc.Encode(payload); err != nil {
            fmt.Fprintf(os.Stderr, "failed to write json: %v\n", err)
            os.Exit(1)
        }
    } else {
        fmt.Printf("node-module-man scan\nroot: %s\nfound: %d\n", absRoot, len(results))
        fmt.Println("----------------------------------------------")
        for _, r := range results {
            sizeStr := utils.HumanizeBytes(r.Size)
            if r.Err != nil {
                fmt.Printf("%s\t%s\t(ERROR: %v)\n", r.Path, sizeStr, r.Err)
            } else {
                fmt.Printf("%s\t%s\n", r.Path, sizeStr)
            }
        }
        fmt.Println("----------------------------------------------")
        fmt.Printf("Total size: %s\n", utils.HumanizeBytes(totalSize))
        fmt.Printf("Duration: %s\n", time.Since(start).Round(time.Millisecond))
    }

    if scanErr != nil {
        os.Exit(1)
    }
}
