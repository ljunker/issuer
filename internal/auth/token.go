// Package auth resolves credentials for the issue tracker backends.
package auth

import (
	"errors"
	"os"
	"os/exec"
	"strings"
)

// ErrNoToken is returned when no GitHub token could be found.
var ErrNoToken = errors.New("no GitHub token found: set GITHUB_TOKEN or run `gh auth login`")

// runGH is swapped out in tests.
var runGH = func() (string, error) {
	out, err := exec.Command("gh", "auth", "token").Output()
	return string(out), err
}

// GitHubToken returns a token from GITHUB_TOKEN, GH_TOKEN or `gh auth token`,
// in that order.
func GitHubToken() (string, error) {
	for _, env := range []string{"GITHUB_TOKEN", "GH_TOKEN"} {
		if t := strings.TrimSpace(os.Getenv(env)); t != "" {
			return t, nil
		}
	}
	out, err := runGH()
	if err != nil {
		return "", ErrNoToken
	}
	if t := strings.TrimSpace(out); t != "" {
		return t, nil
	}
	return "", ErrNoToken
}
