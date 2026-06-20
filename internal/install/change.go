package install

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"
)

var ErrStalePlan = errors.New("config changed after planning")

type Change struct {
	Path          string
	Before, After []byte
}
type Warning struct{ Code, Message string }

func (change Change) Apply(backup bool) error { return change.apply(backup, os.Rename) }

func (change Change) apply(backup bool, rename func(string, string) error) error {
	info, err := inspectRegular(change.Path)
	if err != nil {
		return err
	}
	current, err := os.ReadFile(change.Path)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("read current config: %w", err)
	}
	if bytes.Equal(current, change.After) {
		return nil
	}
	if !bytes.Equal(current, change.Before) {
		return ErrStalePlan
	}
	dirPath := filepath.Dir(change.Path)
	if err := os.MkdirAll(dirPath, 0o700); err != nil {
		return fmt.Errorf("create config directory: %w", err)
	}
	if backup && len(current) > 0 {
		name := change.Path + ".bak." + time.Now().UTC().Format("20060102T150405.000000000Z")
		if err := os.WriteFile(name, current, 0o600); err != nil {
			return fmt.Errorf("write backup: %w", err)
		}
	}
	tmp, err := os.CreateTemp(dirPath, ".signalrail-*.tmp")
	if err != nil {
		return fmt.Errorf("create config temp file: %w", err)
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	mode := os.FileMode(0o600)
	if info != nil {
		mode = info.Mode().Perm()
	}
	if err := tmp.Chmod(mode); err != nil {
		tmp.Close()
		return err
	}
	if _, err := tmp.Write(change.After); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := rename(tmpName, change.Path); err != nil {
		return fmt.Errorf("replace config: %w", err)
	}
	if runtime.GOOS != "windows" {
		dir, err := os.Open(dirPath)
		if err != nil {
			return fmt.Errorf("open config directory: %w", err)
		}
		syncErr := dir.Sync()
		closeErr := dir.Close()
		if syncErr != nil {
			return fmt.Errorf("sync config directory: %w", syncErr)
		}
		if closeErr != nil {
			return closeErr
		}
	}
	return nil
}

func inspectRegular(path string) (os.FileInfo, error) {
	info, err := os.Lstat(path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("inspect config: %w", err)
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return nil, fmt.Errorf("config target must be a regular file")
	}
	return info, nil
}

func readOptional(path string) ([]byte, error) {
	if _, err := inspectRegular(path); err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	return data, err
}
