package task

import (
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/LAwLi3tCoding/signalrail/internal/status"
)

func TestCompletedTaskRejectsFurtherTransitions(t *testing.T) {
	root := initRepo(t)
	now := time.Now().UTC()
	if _, err := Update(root, Mutation{Kind: Set, Title: "Ship", TotalSteps: 1}, now); err != nil {
		t.Fatal(err)
	}
	if _, err := Update(root, Mutation{Kind: Done}, now); err != nil {
		t.Fatal(err)
	}
	if _, err := Update(root, Mutation{Kind: Step}, now); err == nil {
		t.Fatal("step reopened completed task")
	}
	if _, err := Update(root, Mutation{Kind: Block, BlockerNote: "late"}, now); err == nil {
		t.Fatal("block reopened completed task")
	}
}

func TestAtomicRenameFailurePreservesPreviousFile(t *testing.T) {
	root := initRepo(t)
	path := filepath.Join(root, ".signalrail", "state.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	original := []byte("{\"Title\":\"original\"}\n")
	if err := os.WriteFile(path, original, 0o600); err != nil {
		t.Fatal(err)
	}
	err := writeAtomicWithRename(path, taskFixture(), func(string, string) error { return errors.New("rename failed") })
	if err == nil {
		t.Fatal("expected rename failure")
	}
	got, readErr := os.ReadFile(path)
	if readErr != nil || string(got) != string(original) {
		t.Fatalf("original changed: %q err=%v", got, readErr)
	}
	matches, _ := filepath.Glob(filepath.Join(root, ".signalrail", "state-*.tmp"))
	if len(matches) != 0 {
		t.Fatalf("temporary files left behind: %v", matches)
	}
}

func taskFixture() status.Task { return status.Task{Title: "replacement", State: "active"} }

func TestTaskLifecyclePersistsAtomically(t *testing.T) {
	root := initRepo(t)
	now := time.Date(2026, 6, 19, 2, 0, 0, 0, time.UTC)

	got, err := Update(root, Mutation{Kind: Set, Title: "Build renderer", Phase: "coding", TotalSteps: 3}, now)
	if err != nil || got.State != "active" || got.Step != 0 {
		t.Fatalf("set: task=%+v err=%v", got, err)
	}
	got, err = Update(filepath.Join(root, "nested"), Mutation{Kind: Step}, now.Add(time.Minute))
	if err != nil || got.Step != 1 {
		t.Fatalf("step: task=%+v err=%v", got, err)
	}
	got, err = Update(root, Mutation{Kind: Block, BlockerNote: "waiting for review"}, now.Add(2*time.Minute))
	if err != nil || got.State != "blocked" || got.Step != 1 {
		t.Fatalf("block: task=%+v err=%v", got, err)
	}
	got, err = Update(root, Mutation{Kind: Done}, now.Add(3*time.Minute))
	if err != nil || got.State != "done" || got.Step != 3 {
		t.Fatalf("done: task=%+v err=%v", got, err)
	}
	loaded, err := Load(root)
	if err != nil || loaded.Title != "Build renderer" || loaded.UpdatedAt != now.Add(3*time.Minute) {
		t.Fatalf("load: task=%+v err=%v", loaded, err)
	}
}

func TestInvalidMutationPreservesExistingState(t *testing.T) {
	root := initRepo(t)
	now := time.Now().UTC().Truncate(time.Second)
	want, err := Update(root, Mutation{Kind: Set, Title: "Ship", TotalSteps: 2}, now)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Update(root, Mutation{Kind: Set, Title: "", TotalSteps: -1}, now); err == nil {
		t.Fatal("expected validation error")
	}
	got, err := Load(root)
	if err != nil || got != want {
		t.Fatalf("state changed: got=%+v want=%+v err=%v", got, want, err)
	}
	matches, _ := filepath.Glob(filepath.Join(root, ".signalrail", "*.tmp"))
	if len(matches) != 0 {
		t.Fatalf("temporary files left behind: %v", matches)
	}
}

func TestClearRemovesOnlyTaskState(t *testing.T) {
	root := initRepo(t)
	dir := filepath.Join(root, ".signalrail")
	if _, err := Update(root, Mutation{Kind: Set, Title: "Ship"}, time.Now()); err != nil {
		t.Fatal(err)
	}
	keep := filepath.Join(dir, "keep.txt")
	if err := os.WriteFile(keep, []byte("keep"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Update(root, Mutation{Kind: Clear}, time.Now()); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "state.json")); !os.IsNotExist(err) {
		t.Fatalf("state still exists: %v", err)
	}
	if _, err := os.Stat(keep); err != nil {
		t.Fatalf("unrelated file removed: %v", err)
	}
}

func TestConcurrentUpdatesSerialize(t *testing.T) {
	root := initRepo(t)
	if _, err := Update(root, Mutation{Kind: Set, Title: "Parallel", TotalSteps: 32}, time.Now()); err != nil {
		t.Fatal(err)
	}
	start := make(chan struct{})
	errs := make(chan error, 32)
	var ready sync.WaitGroup
	ready.Add(32)
	for i := 0; i < 32; i++ {
		go func() {
			ready.Done()
			<-start
			_, err := Update(root, Mutation{Kind: Step}, time.Now())
			errs <- err
		}()
	}
	ready.Wait()
	close(start)
	for i := 0; i < 32; i++ {
		if err := <-errs; err != nil {
			t.Fatal(err)
		}
	}
	got, err := Load(root)
	if err != nil {
		t.Fatal(err)
	}
	if got.Step != 32 {
		t.Fatalf("step=%d, want 32", got.Step)
	}
}

func initRepo(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, ".git"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(root, "nested"), 0o700); err != nil {
		t.Fatal(err)
	}
	return root
}
