package scanner

import (
    "context"
    "errors"
    "fmt"
    "io/fs"
    "os"
    "path/filepath"
    "runtime"
    "strings"
    "sync"
)

// ResultItem represents a found node_modules directory and its computed size.
type ResultItem struct {
    Path string
    Size int64
    Err  error
}

// Options defines scanning behavior.
type Options struct {
    Concurrency   int  // workers for size calculation
    MaxDepth      int  // -1 unlimited; 0 means only root
    FollowSymlink bool // whether to follow symlinks
    Excludes      []string // glob patterns matched against full path and base name
}

// ScanNodeModules walks from root to find node_modules folders and compute their sizes.
// Returns the list of results, the total combined size, and a merged error (if any).
func ScanNodeModules(ctx context.Context, root string, opts Options) ([]ResultItem, int64, error) {
    if opts.Concurrency <= 0 {
        opts.Concurrency = runtime.NumCPU()
        if opts.Concurrency < 1 {
            opts.Concurrency = 1
        }
    }

    // Gather candidates first (paths to node_modules). We still bound traversal by MaxDepth.
    var candidates []string
    var walkErrs []error

    rootDepth := depthOf(root)
    walkFn := func(path string, d fs.DirEntry, err error) error {
        if err != nil {
            walkErrs = append(walkErrs, fmt.Errorf("walk error at %s: %w", path, err))
            return nil // continue
        }
        // Exclusions
        if d.IsDir() && excluded(path, opts.Excludes) {
            return filepath.SkipDir
        }
        // Depth control
        if opts.MaxDepth >= 0 {
            if depthOf(path)-rootDepth > opts.MaxDepth {
                if d.IsDir() {
                    return filepath.SkipDir
                }
                return nil
            }
        }

        name := d.Name()
        if name == "node_modules" && d.IsDir() {
            // record and skip descending into it (no need to find nested one under itself)
            candidates = append(candidates, path)
            return filepath.SkipDir
        }

        // Symlink handling
        if !opts.FollowSymlink {
            if d.Type()&os.ModeSymlink != 0 && d.IsDir() {
                return filepath.SkipDir
            }
        }
        return nil
    }

    _ = filepath.WalkDir(root, walkFn)

    // Compute sizes with a worker pool
type job struct{ path string }
    jobs := make(chan job)
    var wg sync.WaitGroup
    var mu sync.Mutex
    results := make([]ResultItem, 0, len(candidates))
    var total int64

    worker := func() {
        defer wg.Done()
        for j := range jobs {
            sz, err := dirSize(ctx, j.path, opts.FollowSymlink)
            mu.Lock()
            results = append(results, ResultItem{Path: j.path, Size: sz, Err: err})
            if err == nil {
                total += sz
            }
            mu.Unlock()
        }
    }

    n := opts.Concurrency
    if n < 1 {
        n = 1
    }
    wg.Add(n)
    for i := 0; i < n; i++ {
        go worker()
    }
    go func() {
        defer close(jobs)
        for _, p := range candidates {
            select {
            case <-ctx.Done():
                return
            case jobs <- job{path: p}:
            }
        }
    }()
    wg.Wait()

    // Merge errors
    return results, total, combineErrors(walkErrs)
}

// ScanNodeModulesStream performs scanning and streams each computed ResultItem
// on the returned channel as soon as it's available. When done, the results
// channel is closed and a single error (if any) is sent on errCh then errCh is
// closed. The total combined size can be computed by the caller incrementally.
func ScanNodeModulesStream(ctx context.Context, root string, opts Options) (<-chan ResultItem, <-chan error) {
    out := make(chan ResultItem)
    errCh := make(chan error, 1)
    go func() {
        defer close(out)
        if opts.Concurrency <= 0 {
            opts.Concurrency = runtime.NumCPU()
            if opts.Concurrency < 1 {
                opts.Concurrency = 1
            }
        }
        var walkErrs []error
        rootDepth := depthOf(root)
        type job struct{ path string }
        jobs := make(chan job)
        var wg sync.WaitGroup

        worker := func() {
            defer wg.Done()
            for j := range jobs {
            sz, err := dirSize(ctx, j.path, opts.FollowSymlink)
                select {
                case <-ctx.Done():
                    return
                case out <- ResultItem{Path: j.path, Size: sz, Err: err}:
                }
            }
        }

        n := opts.Concurrency
        if n < 1 {
            n = 1
        }
        wg.Add(n)
        for i := 0; i < n; i++ {
            go worker()
        }

        walkFn := func(path string, d fs.DirEntry, err error) error {
            if err != nil {
                walkErrs = append(walkErrs, fmt.Errorf("walk error at %s: %w", path, err))
                return nil
            }
            if d.IsDir() && excluded(path, opts.Excludes) {
                return filepath.SkipDir
            }
            if opts.MaxDepth >= 0 {
                if depthOf(path)-rootDepth > opts.MaxDepth {
                    if d.IsDir() {
                        return filepath.SkipDir
                    }
                    return nil
                }
            }
            name := d.Name()
            if name == "node_modules" && d.IsDir() {
                select {
                case <-ctx.Done():
                    return ctx.Err()
                case jobs <- job{path: path}:
                }
                return filepath.SkipDir
            }
            if !opts.FollowSymlink {
                if d.Type()&os.ModeSymlink != 0 && d.IsDir() {
                    return filepath.SkipDir
                }
            }
            return nil
        }

        // feed jobs via walk
        go func() {
            _ = filepath.WalkDir(root, walkFn)
            close(jobs)
        }()
        wg.Wait()
        // emit merged walk errors
        errCh <- combineErrors(walkErrs)
        close(errCh)
    }()
    return out, errCh
}

// dirSize computes total size in bytes of a directory tree.
func dirSize(ctx context.Context, root string, followSymlink bool) (int64, error) {
    seen := make(map[string]struct{})
    return dirSizeRec(ctx, root, followSymlink, seen)
}

func dirSizeRec(ctx context.Context, root string, followSymlink bool, seen map[string]struct{}) (int64, error) {
    var total int64
    var firstErr error
    err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
        if err != nil {
            if firstErr == nil {
                firstErr = err
            }
            return nil // continue
        }
        select {
        case <-ctx.Done():
            return ctx.Err()
        default:
        }
        // handle symlinked directories
        if d.IsDir() && d.Type()&os.ModeSymlink != 0 {
            if followSymlink {
                // follow to real target if it's a dir
                if info, e := os.Stat(path); e == nil && info.IsDir() {
                    if real, e2 := filepath.EvalSymlinks(path); e2 == nil {
                        if _, ok := seen[real]; ok {
                            return filepath.SkipDir
                        }
                        seen[real] = struct{}{}
                        sz, e3 := dirSizeRec(ctx, real, followSymlink, seen)
                        if e3 != nil && firstErr == nil { firstErr = e3 }
                        total += sz
                        return filepath.SkipDir
                    }
                }
            }
            // not following
            return filepath.SkipDir
        }
        if d.IsDir() {
            return nil
        }
        // file (d.Info follows symlink for files)
        info, e := d.Info()
        if e != nil {
            if firstErr == nil {
                firstErr = e
            }
            return nil
        }
        total += info.Size()
        return nil
    })
    if err != nil && !errors.Is(err, context.Canceled) {
        if firstErr == nil {
            firstErr = err
        } else {
            firstErr = fmt.Errorf("%v; %v", firstErr, err)
        }
    }
    return total, firstErr
}

func depthOf(p string) int {
    // Normalize separators to OS-specific then count elements
    clean := filepath.Clean(p)
    if clean == string(os.PathSeparator) {
        return 0
    }
    // Count components by splitting — but filepath.SplitList is for PATH lists; use Walk instead
    depth := 0
    for {
        parent := filepath.Dir(clean)
        if parent == clean {
            break
        }
        depth++
        clean = parent
    }
    return depth
}

func combineErrors(errs []error) error {
    if len(errs) == 0 {
        return nil
    }
    if len(errs) == 1 {
        return errs[0]
    }
    var b strings.Builder
    b.WriteString("multiple errors:")
    for _, e := range errs {
        if e == nil {
            continue
        }
        b.WriteString("\n - ")
        b.WriteString(e.Error())
    }
    return errors.New(b.String())
}

func excluded(p string, patterns []string) bool {
    if len(patterns) == 0 {
        return false
    }
    base := filepath.Base(p)
    for _, pat := range patterns {
        if pat == "" { continue }
        // try full path
        if ok, _ := filepath.Match(pat, p); ok {
            return true
        }
        // try base name
        if ok, _ := filepath.Match(pat, base); ok {
            return true
        }
    }
    return false
}
