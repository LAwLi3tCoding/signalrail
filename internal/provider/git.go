package provider

import (
	"context"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/LAwLi3t-CN/signalrail/internal/status"
	"os/exec"
)

const gitTimeout = 50 * time.Millisecond

func gitBudget() time.Duration {
	if runtime.GOOS == "windows" {
		return 250 * time.Millisecond
	}
	return gitTimeout
}

type gitRunner func(context.Context, string, ...string) (string, error)

func Git(parent context.Context, cwd string) status.Project {
	return gitWithRunner(parent, cwd, execGit)
}

func gitWithRunner(parent context.Context, cwd string, runner gitRunner) status.Project {
	project := status.Project{Name: filepath.Base(filepath.Clean(cwd))}
	ctx, cancel := context.WithTimeout(parent, gitBudget())
	defer cancel()
	branch, err := runGitProbe(ctx, runner, cwd, "branch", "--show-current")
	if err != nil {
		return project
	}
	project.Branch = branch
	dirty, err := runGitProbe(ctx, runner, cwd, "status", "--porcelain", "--untracked-files=normal")
	if err == nil {
		project.Dirty = dirty != ""
	}
	return project
}

func runGitProbe(ctx context.Context, runner gitRunner, cwd string, args ...string) (string, error) {
	all := append([]string{"--no-optional-locks", "-C", cwd}, args...)
	return runner(ctx, cwd, all...)
}

func execGit(ctx context.Context, _ string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	out, err := cmd.Output()
	return strings.TrimSpace(string(out)), err
}
