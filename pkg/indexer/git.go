package indexer

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

// gitClone clones a git repository to the target directory.
// Uses a 5-minute timeout for clone operations.
func gitClone(ctx context.Context, url string, target string, sshKeyPath string, sshCommand string) (err error) {
	const cloneTimeout = 5 * time.Minute

	var cancel context.CancelFunc
	ctx, cancel = context.WithTimeout(ctx, cloneTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "git", "clone", url, target)
	cmd.Env = buildGitEnv(sshKeyPath, sshCommand)

	var output []byte
	output, err = cmd.CombinedOutput()
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			err = fmt.Errorf("git clone timed out after %v: %w", cloneTimeout, err)
			return err
		}
		err = fmt.Errorf("git clone failed: %w: %s", err, string(output))
		return err
	}

	return err
}

// gitFetch fetches updates from remote and resets to origin/HEAD.
// Uses a 2-minute timeout for fetch operations.
func gitFetch(ctx context.Context, repoPath string, sshKeyPath string, sshCommand string) (err error) {
	const fetchTimeout = 2 * time.Minute

	var cancel context.CancelFunc
	ctx, cancel = context.WithTimeout(ctx, fetchTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "git", "-C", repoPath, "fetch", "--all")
	cmd.Env = buildGitEnv(sshKeyPath, sshCommand)

	var output []byte
	output, err = cmd.CombinedOutput()
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			err = fmt.Errorf("git fetch timed out after %v: %w", fetchTimeout, err)
			return err
		}
		err = fmt.Errorf("git fetch failed: %w: %s", err, string(output))
		return err
	}

	cmd = exec.CommandContext(ctx, "git", "-C", repoPath, "reset", "--hard", "origin/HEAD")
	cmd.Env = buildGitEnv(sshKeyPath, sshCommand)

	output, err = cmd.CombinedOutput()
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			err = fmt.Errorf("git reset timed out after %v: %w", fetchTimeout, err)
			return err
		}
		err = fmt.Errorf("git reset failed: %w: %s", err, string(output))
		return err
	}

	return err
}

// buildGitEnv constructs the environment for git commands with SSH configuration.
func buildGitEnv(sshKeyPath string, sshCommand string) (env []string) {
	env = os.Environ()

	// If custom SSH command is provided, use it
	if sshCommand != "" {
		env = append(env, fmt.Sprintf("GIT_SSH_COMMAND=%s", sshCommand))
		return env
	}

	// If SSH key path is provided, build SSH command
	if sshKeyPath != "" {
		sshCmd := fmt.Sprintf("ssh -i %s -o StrictHostKeyChecking=yes", sshKeyPath)
		env = append(env, fmt.Sprintf("GIT_SSH_COMMAND=%s", sshCmd))
		return env
	}

	return env
}

// buildRepoURL constructs a repository URL from template, org, repo name, and optional token.
func buildRepoURL(urlFormat string, org string, repo string, token string) (url string) {
	url = strings.ReplaceAll(urlFormat, "{org}", org)
	url = strings.ReplaceAll(url, "{repo}", repo)

	if token != "" {
		url = strings.Replace(url, "https://", fmt.Sprintf("https://%s@", token), 1)
	}

	return url
}
