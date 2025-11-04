package scanner

import (
    "os"
    "path/filepath"
    "testing"
)

func writeFileOfSize(t *testing.T, path string, size int64) {
    t.Helper()
    f, err := os.Create(path)
    if err != nil {
        t.Fatalf("create %s: %v", path, err)
    }
    defer f.Close()
    if err := f.Truncate(size); err != nil {
        t.Fatalf("truncate %s: %v", path, err)
    }
}

func TestScanNodeModules_FindsAndSizes(t *testing.T) {
    root := t.TempDir()

    // a/node_modules with two files 1KB and 2KB
    aNM := filepath.Join(root, "a", "node_modules")
    if err := os.MkdirAll(aNM, 0o755); err != nil {
        t.Fatalf("mkdir: %v", err)
    }
    writeFileOfSize(t, filepath.Join(aNM, "x.bin"), 1024)
    writeFileOfSize(t, filepath.Join(aNM, "y.bin"), 2048)

    // b/node_modules with one file 3KB
    bNM := filepath.Join(root, "b", "node_modules")
    if err := os.MkdirAll(bNM, 0o755); err != nil {
        t.Fatalf("mkdir: %v", err)
    }
    writeFileOfSize(t, filepath.Join(bNM, "z.bin"), 3072)

    // c with no node_modules
    if err := os.MkdirAll(filepath.Join(root, "c"), 0o755); err != nil {
        t.Fatalf("mkdir: %v", err)
    }

    results, total, err := ScanNodeModules(nil, root, Options{Concurrency: 2})
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if len(results) != 2 {
        t.Fatalf("expected 2 results, got %d", len(results))
    }
    wantTotal := int64(1024 + 2048 + 3072)
    if total != wantTotal {
        t.Fatalf("total size mismatch: got %d want %d", total, wantTotal)
    }
}

func TestScanNodeModules_MaxDepth(t *testing.T) {
    root := t.TempDir()
    // root/level1/level2/node_modules
    nm := filepath.Join(root, "level1", "level2", "node_modules")
    if err := os.MkdirAll(nm, 0o755); err != nil {
        t.Fatalf("mkdir: %v", err)
    }
    writeFileOfSize(t, filepath.Join(nm, "a.bin"), 10)

    // depth limits: with MaxDepth=1 should not find level2
    results, _, err := ScanNodeModules(nil, root, Options{MaxDepth: 1})
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if len(results) != 0 {
        t.Fatalf("expected 0 results with MaxDepth=1, got %d", len(results))
    }

    // with MaxDepth=2 should find
    results, _, err = ScanNodeModules(nil, root, Options{MaxDepth: 2})
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if len(results) != 1 {
        t.Fatalf("expected 1 result with MaxDepth=2, got %d", len(results))
    }
}

func TestScanNodeModules_Exclude(t *testing.T) {
    root := t.TempDir()
    a := filepath.Join(root, "a", "node_modules")
    b := filepath.Join(root, "b", "node_modules")
    if err := os.MkdirAll(a, 0o755); err != nil { t.Fatalf("mkdir: %v", err) }
    if err := os.MkdirAll(b, 0o755); err != nil { t.Fatalf("mkdir: %v", err) }
    writeFileOfSize(t, filepath.Join(a, "x"), 10)
    writeFileOfSize(t, filepath.Join(b, "y"), 10)

    // exclude path matching "*/a/*"
    results, total, err := ScanNodeModules(nil, root, Options{Excludes: []string{"*/a/*"}})
    if err != nil { t.Fatalf("unexpected error: %v", err) }
    if len(results) != 1 { t.Fatalf("expected 1 result, got %d", len(results)) }
    if total != 10 { t.Fatalf("unexpected total: %d", total) }

    // exclude by basename "node_modules" should skip all
    results, _, err = ScanNodeModules(nil, root, Options{Excludes: []string{"node_modules"}})
    if err != nil { t.Fatalf("unexpected error: %v", err) }
    if len(results) != 0 { t.Fatalf("expected 0 results, got %d", len(results)) }
}
