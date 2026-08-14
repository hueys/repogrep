package github

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/google/go-github/v75/github"
)

// resolveToken discovers a GitHub token: first by reusing the `gh` CLI's
// existing keyring-backed auth (`gh auth token`), then falling back to the
// GITHUB_TOKEN environment variable.
func resolveToken(ctx context.Context) (string, error) {
	if _, err := exec.LookPath("gh"); err == nil {
		out, err := exec.CommandContext(ctx, "gh", "auth", "token").Output()
		if err == nil {
			if tok := strings.TrimSpace(string(out)); tok != "" {
				return tok, nil
			}
		}
	}
	if tok := strings.TrimSpace(os.Getenv("GITHUB_TOKEN")); tok != "" {
		return tok, nil
	}
	return "", fmt.Errorf("no GitHub token found: run `gh auth login` or set GITHUB_TOKEN")
}

func newClient(ctx context.Context) (*github.Client, error) {
	token, err := resolveToken(ctx)
	if err != nil {
		return nil, err
	}
	return github.NewClient(nil).WithAuthToken(token), nil
}
