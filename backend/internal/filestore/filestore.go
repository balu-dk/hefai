// Package filestore stores uploaded files on the local filesystem under a
// configurable root (a Docker volume in production). Keys are relative
// paths like "<projectID>/<uuid>.pdf".
package filestore

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type Disk struct {
	root string
}

func NewDisk(root string) (*Disk, error) {
	if err := os.MkdirAll(root, 0o755); err != nil {
		return nil, fmt.Errorf("create file store root: %w", err)
	}
	return &Disk{root: root}, nil
}

// path resolves a key inside the root and rejects traversal outside it.
func (d *Disk) path(key string) (string, error) {
	p := filepath.Join(d.root, filepath.FromSlash(key))
	rootAbs, err := filepath.Abs(d.root)
	if err != nil {
		return "", err
	}
	pAbs, err := filepath.Abs(p)
	if err != nil {
		return "", err
	}
	if pAbs != rootAbs && !strings.HasPrefix(pAbs, rootAbs+string(os.PathSeparator)) {
		return "", fmt.Errorf("invalid storage key %q", key)
	}
	return pAbs, nil
}

// Save writes r to key and returns the byte count. Writes go through a temp
// file + rename so readers never see partial content.
func (d *Disk) Save(key string, r io.Reader) (int64, error) {
	p, err := d.path(key)
	if err != nil {
		return 0, err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return 0, err
	}
	tmp, err := os.CreateTemp(filepath.Dir(p), ".upload-*")
	if err != nil {
		return 0, err
	}
	defer os.Remove(tmp.Name())

	n, err := io.Copy(tmp, r)
	if closeErr := tmp.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		return 0, err
	}
	return n, os.Rename(tmp.Name(), p)
}

func (d *Disk) Open(key string) (io.ReadSeekCloser, error) {
	p, err := d.path(key)
	if err != nil {
		return nil, err
	}
	return os.Open(p)
}

func (d *Disk) Delete(key string) error {
	p, err := d.path(key)
	if err != nil {
		return err
	}
	err = os.Remove(p)
	if os.IsNotExist(err) {
		return nil // already gone — deletion is idempotent
	}
	return err
}
