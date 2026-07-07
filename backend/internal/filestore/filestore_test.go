package filestore

import (
	"io"
	"strings"
	"testing"
)

func TestDiskRoundtrip(t *testing.T) {
	store, err := NewDisk(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}

	n, err := store.Save("proj/abc.txt", strings.NewReader("hej fil"))
	if err != nil {
		t.Fatal(err)
	}
	if n != 7 {
		t.Errorf("saved %d bytes, want 7", n)
	}

	f, err := store.Open("proj/abc.txt")
	if err != nil {
		t.Fatal(err)
	}
	content, _ := io.ReadAll(f)
	f.Close()
	if string(content) != "hej fil" {
		t.Errorf("read %q", content)
	}

	if err := store.Delete("proj/abc.txt"); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Open("proj/abc.txt"); err == nil {
		t.Error("expected error opening deleted file")
	}
	if err := store.Delete("proj/abc.txt"); err != nil {
		t.Errorf("second delete should be idempotent, got %v", err)
	}
}

func TestDiskRejectsTraversal(t *testing.T) {
	store, err := NewDisk(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Save("../escape.txt", strings.NewReader("x")); err == nil {
		t.Error("path traversal accepted")
	}
	if _, err := store.Open("../../etc/passwd"); err == nil {
		t.Error("path traversal accepted on open")
	}
}
