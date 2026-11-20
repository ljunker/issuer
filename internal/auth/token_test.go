package auth

import (
	"errors"
	"testing"
)

func TestGitHubToken(t *testing.T) {
	orig := runGH
	t.Cleanup(func() { runGH = orig })

	tests := []struct {
		name    string
		ghToken string
		gh2     string
		ghOut   string
		ghErr   error
		want    string
		wantErr bool
	}{
		{name: "GITHUB_TOKEN wins", ghToken: "a", gh2: "b", ghOut: "c", want: "a"},
		{name: "GH_TOKEN second", gh2: "b", ghOut: "c", want: "b"},
		{name: "gh cli fallback", ghOut: "c\n", want: "c"},
		{name: "gh cli fails", ghErr: errors.New("x"), wantErr: true},
		{name: "gh cli empty", ghOut: "  \n", wantErr: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("GITHUB_TOKEN", tc.ghToken)
			t.Setenv("GH_TOKEN", tc.gh2)
			runGH = func() (string, error) { return tc.ghOut, tc.ghErr }
			got, err := GitHubToken()
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error, got %q", got)
				}
				return
			}
			if err != nil || got != tc.want {
				t.Fatalf("got %q, %v; want %q", got, err, tc.want)
			}
		})
	}
}
