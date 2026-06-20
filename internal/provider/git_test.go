package provider

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

func TestGitReadsBranchAndDirtyState(t *testing.T) {
	root := t.TempDir()
	runGit(t, root, "init", "-b", "feature/test")
	runGit(t, root, "config", "user.email", "test@example.com")
	runGit(t, root, "config", "user.name", "Test")
	runGit(t, root, "commit", "--allow-empty", "-m", "init")
	project := Git(context.Background(), root)
	if project.Name != filepath.Base(root) || project.Branch != "feature/test" || project.Dirty {
		t.Fatalf("project=%+v", project)
	}
	if err := os.WriteFile(filepath.Join(root, "dirty.txt"), []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	if project = Git(context.Background(), root); !project.Dirty {
		t.Fatalf("project=%+v", project)
	}
}

func TestGitHonorsCancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	start := time.Now()
	got := Git(ctx, t.TempDir())
	if time.Since(start) > 100*time.Millisecond {
		t.Fatal("cancelled provider blocked")
	}
	if got.Branch != "" || got.Dirty {
		t.Fatalf("project=%+v", got)
	}
}

func TestGitRunnerGetsNoLockFlagAndSharedDeadline(t *testing.T) {
	var calls [][]string
	var deadlines []time.Time
	runner := func(ctx context.Context, _ string, args ...string) (string, error) {
		calls = append(calls, append([]string(nil), args...))
		deadline, ok := ctx.Deadline()
		if !ok {
			t.Fatal("missing deadline")
		}
		deadlines = append(deadlines, deadline)
		if len(calls) == 1 {
			time.Sleep(20 * time.Millisecond)
			return "main", nil
		}
		return "", nil
	}
	got := gitWithRunner(context.Background(), "/tmp/demo", runner)
	if got.Branch != "main" || len(calls) != 2 {
		t.Fatalf("project=%+v calls=%v", got, calls)
	}
	for _, args := range calls {
		if len(args) == 0 || args[0] != "--no-optional-locks" {
			t.Fatalf("args=%v", args)
		}
	}
	if !deadlines[1].Equal(deadlines[0]) {
		t.Fatalf("deadlines do not share one budget: %v", deadlines)
	}
}

func TestGitRunnerStopsAtTimeout(t *testing.T) {
	runner := func(ctx context.Context, _ string, _ ...string) (string, error) {
		<-ctx.Done()
		return "", errors.New("slow")
	}
	start := time.Now()
	_ = gitWithRunner(context.Background(), "/tmp/demo", runner)
	if elapsed := time.Since(start); elapsed > gitBudget()+100*time.Millisecond {
		t.Fatalf("elapsed=%v", elapsed)
	}
}

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v: %s", args, err, out)
	}
}
