// Package provider defines the system-neutral interface that every issue
// tracker backend (GitHub, GitLab, Jira, ...) implements. The UI only talks
// to this interface.
package provider

import (
	"context"
	"time"
)

// State filters issues by their open/closed state.
type State string

const (
	StateOpen   State = "open"
	StateClosed State = "closed"
	StateAll    State = "all"
)

// Next cycles open → closed → all → open.
func (s State) Next() State {
	switch s {
	case StateOpen:
		return StateClosed
	case StateClosed:
		return StateAll
	default:
		return StateOpen
	}
}

type Project struct {
	ID          string // provider-specific, e.g. "owner/repo"
	Name        string
	Owner       string
	Description string
	OpenIssues  int
	Private     bool
	URL         string
	UpdatedAt   time.Time
}

type Label struct {
	Name        string
	Color       string // hex without '#', may be empty
	Description string
}

type Issue struct {
	Number    int
	Title     string
	Body      string
	State     string // "open" or "closed"
	Author    string
	Labels    []Label
	Assignees []string
	Comments  int
	CreatedAt time.Time
	UpdatedAt time.Time
	URL       string
}

func (i Issue) Open() bool { return i.State == string(StateOpen) }

type Comment struct {
	ID        int64
	Author    string
	Body      string
	CreatedAt time.Time
	URL       string
}

type IssueFilter struct {
	State State
}

type NewIssue struct {
	Title  string
	Body   string
	Labels []string
}

// Provider is implemented by every issue tracker backend.
type Provider interface {
	Name() string
	ListProjects(ctx context.Context) ([]Project, error)
	ListIssues(ctx context.Context, projectID string, f IssueFilter) ([]Issue, error)
	GetIssue(ctx context.Context, projectID string, number int) (Issue, error)
	ListComments(ctx context.Context, projectID string, number int) ([]Comment, error)
	AddComment(ctx context.Context, projectID string, number int, body string) (Comment, error)
	CreateIssue(ctx context.Context, projectID string, in NewIssue) (Issue, error)
	SetState(ctx context.Context, projectID string, number int, open bool) (Issue, error)
	ListLabels(ctx context.Context, projectID string) ([]Label, error)
	SetIssueLabels(ctx context.Context, projectID string, number int, labels []string) ([]Label, error)
}
