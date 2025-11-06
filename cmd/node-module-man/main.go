package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"node-module-man/internal/deleter"
	"node-module-man/internal/compressor"
	"node-module-man/internal/scanner"
	ui "node-module-man/internal/tui"
	"node-module-man/pkg/utils"
)

// version is injected at build time via -ldflags "-X main.version=..."
var version = "dev"

type multiFlag []string

func (m *multiFlag) String() string     { return fmt.Sprint([]string(*m)) }
func (m *multiFlag) Set(v string) error { *m = append(*m, v); return nil }

func main() {
	var (
		root        string
		jsonOut     bool
		showVersion bool
		yesDelete   bool
		deleteJSON  string
		deleteStdin bool
		compressJSON string
		compressStdin bool
		outDir      string
		deleteAfter bool
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
	flag.BoolVar(&showVersion, "version", false, "Print version and exit")
	flag.BoolVar(&yesDelete, "yes", false, "Do not prompt for confirmation in CLI delete mode")
	flag.StringVar(&deleteJSON, "delete-json", "", "Delete targets from JSON file (array of paths or {path,size} objects)")
	flag.BoolVar(&deleteStdin, "delete-stdin", false, "Read delete targets JSON from stdin")
	flag.StringVar(&compressJSON, "compress-json", "", "Compress targets from JSON file (array of paths or {path,size} objects)")
	flag.BoolVar(&compressStdin, "compress-stdin", false, "Read compress targets JSON from stdin")
    flag.StringVar(&outDir, "out-dir", "", "Output directory for compressed archives (default: alongside source)")
    flag.BoolVar(&deleteAfter, "delete-after", true, "Delete original directory after successful compression (default true)")
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

	if showVersion {
		fmt.Println(version)
		return
	}

	// Deletion CLI mode via JSON input
	if deleteJSON != "" || deleteStdin {
		if !yesDelete {
			fmt.Fprintln(os.Stderr, "--yes is required for non-interactive deletion. Aborting.")
			os.Exit(2)
		}
		var r io.Reader
		if deleteStdin {
			r = os.Stdin
		} else {
			f, err := os.Open(deleteJSON)
			if err != nil {
				fmt.Fprintf(os.Stderr, "failed to open delete-json file: %v\n", err)
				os.Exit(2)
			}
			defer f.Close()
			r = f
		}
		targets, err := readDeleteTargets(r)
		if err != nil {
			fmt.Fprintf(os.Stderr, "invalid delete targets JSON: %v\n", err)
			os.Exit(2)
		}
		// Execute deletions
		ctx := context.Background()
		sum := deleter.DeleteTargets(ctx, targets, concurrency, nil, dryRun)
		if jsonOut {
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			if err := enc.Encode(sum); err != nil {
				fmt.Fprintf(os.Stderr, "failed to write json: %v\n", err)
				os.Exit(1)
			}
		} else {
			fmt.Printf("Deleted: %d  Failed: %d  Freed: %s\n", len(sum.Successes), len(sum.Failures), utils.HumanizeBytes(sum.Freed))
			if len(sum.Failures) > 0 {
				fmt.Println("Failures:")
				for _, f := range sum.Failures {
					fmt.Printf(" - %s: %v\n", f.Path, f.Err)
				}
			}
		}
		if len(sum.Failures) > 0 {
			os.Exit(1)
		}
		return
	}

	// Compression CLI mode via JSON input
	if compressJSON != "" || compressStdin {
		if !yesDelete {
			fmt.Fprintln(os.Stderr, "--yes is required for non-interactive compression. Aborting.")
			os.Exit(2)
		}
		var r io.Reader
		if compressStdin {
			r = os.Stdin
		} else {
			f, err := os.Open(compressJSON)
			if err != nil {
				fmt.Fprintf(os.Stderr, "failed to open compress-json file: %v\n", err)
				os.Exit(2)
			}
			defer f.Close()
			r = f
		}
		dt, err := readDeleteTargets(r)
		if err != nil {
			fmt.Fprintf(os.Stderr, "invalid compress targets JSON: %v\n", err)
			os.Exit(2)
		}
		// Map to compressor targets
		cts := make([]compressor.Target, 0, len(dt))
		for _, t := range dt {
			cts = append(cts, compressor.Target{Path: t.Path, Size: t.Size})
		}
		ctx := context.Background()
		sum := compressor.CompressTargets(ctx, cts, compressor.Options{OutDir: outDir, Concurrency: concurrency, DeleteAfter: deleteAfter}, nil)
		if jsonOut {
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			if err := enc.Encode(sum); err != nil {
				fmt.Fprintf(os.Stderr, "failed to write json: %v\n", err)
				os.Exit(1)
			}
		} else {
			fmt.Printf("Compressed: %d  Failed: %d  Written: %s\n", len(sum.Successes), len(sum.Failures), utils.HumanizeBytes(sum.Written))
			if len(sum.Failures) > 0 {
				fmt.Println("Failures:")
				for _, f := range sum.Failures {
					fmt.Printf(" - %s: %v\n", f.Path, f.Err)
				}
			}
		}
		if len(sum.Failures) > 0 {
			os.Exit(1)
		}
		return
	}

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

// readDeleteTargets is flexible with input schema:
// - ["/path/one", "/path/two"]
// - [{"path":"/p","size":123}, ...]
// - {"targets":[ ...either of above... ]}
func readDeleteTargets(r io.Reader) ([]deleter.Target, error) {
	dec := json.NewDecoder(r)
	// Use raw message to inspect shape
	var raw interface{}
	if err := dec.Decode(&raw); err != nil {
		return nil, err
	}
	var toTargets func(v interface{}) ([]deleter.Target, error)
	toTargets = func(v interface{}) ([]deleter.Target, error) {
		switch t := v.(type) {
		case []interface{}:
			res := make([]deleter.Target, 0, len(t))
			for _, e := range t {
				switch ee := e.(type) {
				case string:
					res = append(res, deleter.Target{Path: ee})
				case map[string]interface{}:
					p, _ := ee["path"].(string)
					var size int64
					switch vv := ee["size"].(type) {
					case float64:
						size = int64(vv)
					}
					if p != "" {
						res = append(res, deleter.Target{Path: p, Size: size})
					}
				default:
					// ignore unknown entries
				}
			}
			return res, nil
		case map[string]interface{}:
			if inner, ok := t["targets"]; ok {
				return toTargets(inner)
			}
		}
		return nil, fmt.Errorf("unsupported JSON format for delete targets")
	}
	return toTargets(raw)
}
