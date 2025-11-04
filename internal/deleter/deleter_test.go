package deleter

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDeleteTargets_DryRunDoesNotDelete(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "node_modules")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	tgs := []Target{{Path: dir, Size: 1234}}
	sum := DeleteTargets(nil, tgs, 1, nil, true)
	if len(sum.Failures) != 0 {
		t.Fatalf("unexpected failures: %v", sum.Failures)
	}
	if _, err := os.Stat(dir); err != nil {
		t.Fatalf("dir should still exist in dry-run: %v", err)
	}
	if sum.Freed != 1234 {
		t.Fatalf("freed mismatch: %d", sum.Freed)
	}
}
