package main

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"testing"
)

// TestCLIReferenceUpToDate is the drift check behind the committed CLI
// reference: the rendered command tree must byte-match docs/cli/.
func TestCLIReferenceUpToDate(t *testing.T) {
	want, err := os.ReadFile(filepath.Join("..", "..", "docs", "cli", "reference.md"))
	if err != nil {
		t.Fatalf("read committed reference: %v (generate it with `go generate ./cmd/moth`)", err)
	}
	if got := renderCLIReference(newRootCmd()); !bytes.Equal(got, want) {
		t.Error("docs/cli/reference.md is stale; regenerate with `go generate ./cmd/moth`")
	}
}

// TestDocsGenDeterministic renders through the real command (which goes
// through cobra's Execute, adding the implicit help/completion commands)
// and directly, and expects identical bytes both times.
func TestDocsGenDeterministic(t *testing.T) {
	direct := renderCLIReference(newRootCmd())
	if !bytes.Equal(direct, renderCLIReference(newRootCmd())) {
		t.Fatal("renderCLIReference is not deterministic")
	}

	dir := t.TempDir()
	root := newRootCmd()
	root.SetOut(io.Discard)
	root.SetErr(io.Discard)
	root.SetArgs([]string{"docs", "gen", "--dir", dir})
	if err := root.Execute(); err != nil {
		t.Fatalf("docs gen: %v", err)
	}
	viaCommand, err := os.ReadFile(filepath.Join(dir, "reference.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(direct, viaCommand) {
		t.Error("docs gen output differs from direct rendering (help/completion leaked in?)")
	}
}
