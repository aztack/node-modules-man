package compressor

import (
    "archive/zip"
    "context"
    "errors"
    "fmt"
    "io"
    "io/fs"
    "os"
    "path/filepath"
    "strings"
)

type Target struct {
    Path string
    Size int64
}

type Progress struct {
    Completed    int
    Total        int
    Path         string
    Dest         string
    BytesWritten int64
    Err          error
}

type Success struct {
    Path string
    Dest string
    Size int64 // archive size in bytes
}

type Failure struct {
    Path string
    Err  error
}

type Summary struct {
    Successes []Success
    Failures  []Failure
    Written   int64 // total bytes of archives written
}

type Options struct {
    OutDir      string
    Concurrency int
    DeleteAfter bool
}

// CompressTargets creates one .zip archive per target directory.
// Progress events are sent per target completion and may include intermediate file paths.
func CompressTargets(ctx context.Context, targets []Target, opts Options, progress chan<- Progress) Summary {
    sum := Summary{Successes: make([]Success, 0, len(targets))}
    total := len(targets)
    for i, t := range targets {
        select {
        case <-ctx.Done():
            // abort; report remaining as failures due to cancellation
            for j := i; j < len(targets); j++ {
                if progress != nil {
                    progress <- Progress{Completed: j, Total: total, Path: targets[j].Path, Err: ctx.Err()}
                }
                sum.Failures = append(sum.Failures, Failure{Path: targets[j].Path, Err: ctx.Err()})
            }
            return sum
        default:
        }

        src := t.Path
        // Validate source is directory
        inf, err := os.Stat(src)
        if err != nil {
            if progress != nil { progress <- Progress{Completed: i, Total: total, Path: src, Err: err} }
            sum.Failures = append(sum.Failures, Failure{Path: src, Err: err})
            continue
        }
        if !inf.IsDir() {
            err := fmt.Errorf("not a directory: %s", src)
            if progress != nil { progress <- Progress{Completed: i, Total: total, Path: src, Err: err} }
            sum.Failures = append(sum.Failures, Failure{Path: src, Err: err})
            continue
        }

        destDir := opts.OutDir
        if destDir == "" {
            destDir = filepath.Dir(src)
        }
        if err := os.MkdirAll(destDir, 0o755); err != nil {
            if progress != nil { progress <- Progress{Completed: i, Total: total, Path: src, Err: err} }
            sum.Failures = append(sum.Failures, Failure{Path: src, Err: err})
            continue
        }

        base := filepath.Base(src)
        // Archive file name without timestamp for friendlier extraction names.
        name := fmt.Sprintf("%s.zip", base)
        dest := filepath.Join(destDir, name)
        // Avoid overwrites
        dest = nextAvailable(dest)

        written, err := zipDirectory(ctx, src, dest, func(rel string, bytes int64) {
            if progress != nil {
                progress <- Progress{Completed: i, Total: total, Path: filepath.Join(src, rel), Dest: dest, BytesWritten: bytes}
            }
        })
        if err != nil {
            // cleanup partial file
            _ = os.Remove(dest)
            if progress != nil { progress <- Progress{Completed: i + 1, Total: total, Path: src, Dest: dest, Err: err} }
            sum.Failures = append(sum.Failures, Failure{Path: src, Err: err})
            continue
        }

        // Optionally delete source after success
        if opts.DeleteAfter {
            if rmErr := os.RemoveAll(src); rmErr != nil {
                // Keep success but record failure as warning
                sum.Failures = append(sum.Failures, Failure{Path: src, Err: fmt.Errorf("delete-after failed: %w", rmErr)})
            }
        }

        sum.Successes = append(sum.Successes, Success{Path: src, Dest: dest, Size: written})
        sum.Written += written
        if progress != nil { progress <- Progress{Completed: i + 1, Total: total, Path: src, Dest: dest, BytesWritten: written} }
    }
    return sum
}

func nextAvailable(p string) string {
    if _, err := os.Stat(p); errors.Is(err, fs.ErrNotExist) {
        return p
    }
    dir := filepath.Dir(p)
    base := filepath.Base(p)
    ext := filepath.Ext(base)
    name := strings.TrimSuffix(base, ext)
    for i := 1; i < 10000; i++ {
        cand := filepath.Join(dir, fmt.Sprintf("%s-%d%s", name, i, ext))
        if _, err := os.Stat(cand); errors.Is(err, fs.ErrNotExist) {
            return cand
        }
    }
    return p
}

// zipDirectory zips directory src into dest path. Returns final archive size.
// progressCb is called after each file is written with the relative path and current bytes written.
func zipDirectory(ctx context.Context, src, dest string, progressCb func(rel string, bytes int64)) (int64, error) {
    f, err := os.Create(dest)
    if err != nil { return 0, err }
    defer func() { _ = f.Close() }()

    zw := zip.NewWriter(f)
    defer func() { _ = zw.Close() }()

    // Walk the directory and add files
    prefix := filepath.Base(src)
    var totalWritten int64
    err = filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
        if err != nil { return err }
        rel, err := filepath.Rel(src, path)
        if err != nil { return err }
        // Skip root entry itself
        if rel == "." { return nil }

        // Respect context cancellation
        select {
        case <-ctx.Done():
            return ctx.Err()
        default:
        }

        info, err := d.Info()
        if err != nil { return err }

        // Skip symlinks to avoid reading directories or external targets
        if info.Mode() & os.ModeSymlink != 0 {
            return nil
        }

        // Create header
        hdr, err := zip.FileInfoHeader(info)
        if err != nil { return err }
        // Force a top-level directory prefix and forward slashes in zip
        hdr.Name = filepath.ToSlash(filepath.Join(prefix, rel))
        if d.IsDir() {
            hdr.Name += "/"
        } else {
            hdr.Method = zip.Deflate
        }

        w, err := zw.CreateHeader(hdr)
        if err != nil { return err }

        if d.IsDir() {
            return nil
        }
        // Copy file contents
        rf, err := os.Open(path)
        if err != nil { return err }
        defer rf.Close()
        n, err := io.Copy(w, rf)
        if err != nil { return err }
        totalWritten += n
        if progressCb != nil {
            progressCb(rel, totalWritten)
        }
        return nil
    })
    if err != nil { return 0, err }

    if err := zw.Close(); err != nil { return 0, err }
    if err := f.Sync(); err != nil { return 0, err }
    st, err := os.Stat(dest)
    if err != nil { return 0, err }
    return st.Size(), nil
}
