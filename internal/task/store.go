package task

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/LAwLi3t-CN/signalrail/internal/status"
)

type Kind string

const (
	Set   Kind = "set"
	Step  Kind = "step"
	Block Kind = "block"
	Done  Kind = "done"
	Clear Kind = "clear"
)

type Mutation struct {
	Kind          Kind
	Title         string
	Phase         string
	Step          int
	TotalSteps    int
	BlockerNote   string
	SourceRuntime string
}

func Load(start string) (status.Task, error) {
	root, err := projectRoot(start)
	if err != nil {
		return status.Task{}, err
	}
	data, err := os.ReadFile(filepath.Join(root, ".signalrail", "state.json"))
	if errors.Is(err, os.ErrNotExist) {
		return status.Task{}, nil
	}
	if err != nil {
		return status.Task{}, fmt.Errorf("read task state: %w", err)
	}
	var out status.Task
	if err := json.Unmarshal(data, &out); err != nil {
		return status.Task{}, fmt.Errorf("decode task state: %w", err)
	}
	return out, nil
}

func Update(start string, mutation Mutation, now time.Time) (status.Task, error) {
	root, err := projectRoot(start)
	if err != nil {
		return status.Task{}, err
	}
	path := filepath.Join(root, ".signalrail", "state.json")
	release, err := acquireStateLock(path)
	if err != nil {
		return status.Task{}, err
	}
	defer release()
	if mutation.Kind == Clear {
		if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
			return status.Task{}, fmt.Errorf("clear task state: %w", err)
		}
		return status.Task{}, nil
	}
	current, err := Load(root)
	if err != nil {
		return status.Task{}, err
	}
	next, err := apply(current, mutation, now.UTC())
	if err != nil {
		return status.Task{}, err
	}
	if err := writeAtomic(path, next); err != nil {
		return status.Task{}, err
	}
	return next, nil
}

const (
	lockWait  = 2 * time.Second
	lockStale = 30 * time.Second
)

func acquireStateLock(statePath string) (func(), error) {
	dir := filepath.Dir(statePath)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, fmt.Errorf("create task state directory: %w", err)
	}
	path := filepath.Join(dir, "state.lock")
	deadline := time.Now().Add(lockWait)
	for {
		file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
		if err == nil {
			owned, statErr := file.Stat()
			if statErr != nil {
				file.Close()
				_ = os.Remove(path)
				return nil, fmt.Errorf("inspect task state lock: %w", statErr)
			}
			if closeErr := file.Close(); closeErr != nil {
				_ = os.Remove(path)
				return nil, fmt.Errorf("close task state lock: %w", closeErr)
			}
			return func() {
				if current, currentErr := os.Lstat(path); currentErr == nil && os.SameFile(owned, current) {
					_ = os.Remove(path)
				}
			}, nil
		}
		if !errors.Is(err, os.ErrExist) {
			return nil, fmt.Errorf("acquire task state lock: %w", err)
		}
		if info, statErr := os.Lstat(path); statErr == nil && time.Since(info.ModTime()) > lockStale {
			if removeErr := os.Remove(path); removeErr == nil || errors.Is(removeErr, os.ErrNotExist) {
				continue
			}
		}
		if time.Now().After(deadline) {
			return nil, errors.New("task state is busy; retry the command")
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func apply(task status.Task, mutation Mutation, now time.Time) (status.Task, error) {
	switch mutation.Kind {
	case Set:
		if mutation.Title == "" {
			return status.Task{}, errors.New("task title is required")
		}
		if mutation.TotalSteps < 0 || mutation.Step < 0 || (mutation.TotalSteps > 0 && mutation.Step > mutation.TotalSteps) {
			return status.Task{}, errors.New("invalid task progress")
		}
		task = status.Task{Title: mutation.Title, Phase: mutation.Phase, Step: mutation.Step, TotalSteps: mutation.TotalSteps, State: "active", SourceRuntime: mutation.SourceRuntime}
	case Step:
		if task.Title == "" {
			return status.Task{}, errors.New("no active task")
		}
		if task.State == "done" {
			return status.Task{}, errors.New("task is already done")
		}
		increment := mutation.Step
		if increment == 0 {
			increment = 1
		}
		if increment < 0 || (task.TotalSteps > 0 && task.Step+increment > task.TotalSteps) {
			return status.Task{}, errors.New("step exceeds task total")
		}
		task.Step += increment
		task.State = "active"
		task.BlockerNote = ""
	case Block:
		if task.Title == "" {
			return status.Task{}, errors.New("no active task")
		}
		if task.State == "done" {
			return status.Task{}, errors.New("task is already done")
		}
		task.State = "blocked"
		task.BlockerNote = mutation.BlockerNote
	case Done:
		if task.Title == "" {
			return status.Task{}, errors.New("no active task")
		}
		task.State = "done"
		task.BlockerNote = ""
		if task.TotalSteps > 0 {
			task.Step = task.TotalSteps
		}
	default:
		return status.Task{}, fmt.Errorf("unknown task mutation %q", mutation.Kind)
	}
	task.UpdatedAt = now
	return task, nil
}

func projectRoot(start string) (string, error) {
	if start == "" {
		var err error
		start, err = os.Getwd()
		if err != nil {
			return "", err
		}
	}
	current, err := filepath.Abs(start)
	if err != nil {
		return "", err
	}
	if info, statErr := os.Stat(current); statErr == nil && !info.IsDir() {
		current = filepath.Dir(current)
	}
	for {
		if _, err := os.Stat(filepath.Join(current, ".git")); err == nil {
			return current, nil
		}
		parent := filepath.Dir(current)
		if parent == current {
			return "", fmt.Errorf("no project root from %s", start)
		}
		current = parent
	}
}

func writeAtomic(path string, task status.Task) error {
	return writeAtomicWithRename(path, task, os.Rename)
}

func writeAtomicWithRename(path string, task status.Task, rename func(string, string) error) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("create task state directory: %w", err)
	}
	data, err := json.MarshalIndent(task, "", "  ")
	if err != nil {
		return fmt.Errorf("encode task state: %w", err)
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), "state-*.tmp")
	if err != nil {
		return fmt.Errorf("create task state temp file: %w", err)
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if err := tmp.Chmod(0o600); err != nil {
		tmp.Close()
		return err
	}
	if _, err := tmp.Write(append(data, '\n')); err != nil {
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
	if err := rename(tmpName, path); err != nil {
		return fmt.Errorf("replace task state: %w", err)
	}
	return nil
}
