package github

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ljunker/issuer/internal/provider"
)

func newTestProvider(t *testing.T, mux *http.ServeMux) *Provider {
	t.Helper()
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	p, err := New("test-token", srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func TestListIssuesFiltersPullRequests(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /repos/o/r/issues", func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("state"); got != "closed" {
			t.Errorf("state = %q, want closed", got)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-token" {
			t.Errorf("auth header = %q", got)
		}
		io.WriteString(w, `[
		  {"number":1,"title":"bug","state":"closed","user":{"login":"alice"},
		   "labels":[{"name":"bug","color":"d73a4a"}],"assignees":[{"login":"bob"}],"comments":3},
		  {"number":2,"title":"a PR","state":"closed","pull_request":{"url":"x"}}
		]`)
	})
	p := newTestProvider(t, mux)

	issues, err := p.ListIssues(context.Background(), "o/r", provider.IssueFilter{State: provider.StateClosed})
	if err != nil {
		t.Fatal(err)
	}
	if len(issues) != 1 {
		t.Fatalf("got %d issues, want 1 (PR must be filtered)", len(issues))
	}
	i := issues[0]
	if i.Number != 1 || i.Author != "alice" || i.Comments != 3 || i.Open() {
		t.Errorf("unexpected issue: %+v", i)
	}
	if len(i.Labels) != 1 || i.Labels[0] != (provider.Label{Name: "bug", Color: "d73a4a"}) {
		t.Errorf("labels = %+v", i.Labels)
	}
	if len(i.Assignees) != 1 || i.Assignees[0] != "bob" {
		t.Errorf("assignees = %+v", i.Assignees)
	}
}

func TestSetIssueLabelsSendsEmptyList(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("PUT /repos/o/r/issues/7/labels", func(w http.ResponseWriter, r *http.Request) {
		var body any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		// go-github wraps the slice as {"labels": [...]}
		b, _ := json.Marshal(body)
		if string(b) != `{"labels":[]}` && string(b) != `[]` {
			t.Errorf("body = %s, want empty label list", b)
		}
		io.WriteString(w, `[]`)
	})
	p := newTestProvider(t, mux)

	ls, err := p.SetIssueLabels(context.Background(), "o/r", 7, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(ls) != 0 {
		t.Errorf("labels = %+v", ls)
	}
}

func TestListProjectsSkipsArchivedAndIssueless(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /user/repos", func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, `[
		  {"full_name":"o/a","name":"a","owner":{"login":"o"},"has_issues":true,"open_issues_count":4},
		  {"full_name":"o/b","name":"b","owner":{"login":"o"},"has_issues":true,"archived":true},
		  {"full_name":"o/c","name":"c","owner":{"login":"o"},"has_issues":false}
		]`)
	})
	p := newTestProvider(t, mux)

	ps, err := p.ListProjects(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(ps) != 1 || ps[0].ID != "o/a" || ps[0].OpenIssues != 4 {
		t.Fatalf("projects = %+v", ps)
	}
}

func TestInvalidProjectID(t *testing.T) {
	p, _ := New("t", "")
	if _, err := p.ListIssues(context.Background(), "nope", provider.IssueFilter{}); err == nil {
		t.Fatal("expected error for invalid project id")
	}
}
