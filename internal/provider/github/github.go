// Package github implements provider.Provider for GitHub.
package github

import (
	"context"
	"fmt"
	"sort"
	"strings"

	gh "github.com/google/go-github/v90/github"

	"github.com/ljunker/issuer/internal/provider"
)

const perPage = 100

// maxIssuePages caps how many pages of issues are fetched per project.
const maxIssuePages = 5

type Provider struct {
	c *gh.Client
}

// New creates a GitHub provider authenticated with token. baseURL may be
// empty for github.com; otherwise it points to an API root (tests, GHES).
func New(token, baseURL string) (*Provider, error) {
	opts := []gh.ClientOptionsFunc{gh.WithAuthToken(token), gh.WithUserAgent("issuer")}
	if baseURL != "" {
		if !strings.HasSuffix(baseURL, "/") {
			baseURL += "/"
		}
		opts = append(opts, gh.WithURLs(&baseURL, &baseURL))
	}
	c, err := gh.NewClient(opts...)
	if err != nil {
		return nil, err
	}
	return &Provider{c: c}, nil
}

func (p *Provider) Name() string { return "GitHub" }

func split(projectID string) (string, string, error) {
	owner, repo, ok := strings.Cut(projectID, "/")
	if !ok || owner == "" || repo == "" {
		return "", "", fmt.Errorf("invalid project id %q", projectID)
	}
	return owner, repo, nil
}

func (p *Provider) ListProjects(ctx context.Context) ([]provider.Project, error) {
	opts := &gh.RepositoryListByAuthenticatedUserOptions{
		Affiliation: "owner,collaborator,organization_member",
		Sort:        "updated",
		Direction:   "desc",
		ListOptions: gh.ListOptions{PerPage: perPage},
	}
	var out []provider.Project
	for {
		repos, resp, err := p.c.Repositories.ListByAuthenticatedUser(ctx, opts)
		if err != nil {
			return nil, err
		}
		for _, r := range repos {
			if !r.GetHasIssues() || r.GetArchived() {
				continue
			}
			out = append(out, provider.Project{
				ID:          r.GetFullName(),
				Name:        r.GetName(),
				Owner:       r.GetOwner().GetLogin(),
				Description: r.GetDescription(),
				OpenIssues:  r.GetOpenIssuesCount(),
				Private:     r.GetPrivate(),
				URL:         r.GetHTMLURL(),
				UpdatedAt:   r.GetUpdatedAt().Time,
			})
		}
		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}
	return out, nil
}

func (p *Provider) ListIssues(ctx context.Context, projectID string, f provider.IssueFilter) ([]provider.Issue, error) {
	owner, repo, err := split(projectID)
	if err != nil {
		return nil, err
	}
	state := string(f.State)
	if state == "" {
		state = string(provider.StateOpen)
	}
	opts := &gh.IssueListByRepoOptions{
		State:       state,
		Sort:        "updated",
		Direction:   "desc",
		ListOptions: gh.ListOptions{PerPage: perPage},
	}
	var out []provider.Issue
	for page := 0; page < maxIssuePages; page++ {
		issues, resp, err := p.c.Issues.ListByRepo(ctx, owner, repo, opts)
		if err != nil {
			return nil, err
		}
		for _, i := range issues {
			if i.IsPullRequest() {
				continue
			}
			out = append(out, toIssue(i))
		}
		if resp.NextPage == 0 {
			break
		}
		opts.ListOptions.Page = resp.NextPage
	}
	return out, nil
}

func (p *Provider) GetIssue(ctx context.Context, projectID string, number int) (provider.Issue, error) {
	owner, repo, err := split(projectID)
	if err != nil {
		return provider.Issue{}, err
	}
	i, _, err := p.c.Issues.Get(ctx, owner, repo, number)
	if err != nil {
		return provider.Issue{}, err
	}
	return toIssue(i), nil
}

func (p *Provider) ListComments(ctx context.Context, projectID string, number int) ([]provider.Comment, error) {
	owner, repo, err := split(projectID)
	if err != nil {
		return nil, err
	}
	opts := &gh.IssueListCommentsOptions{ListOptions: gh.ListOptions{PerPage: perPage}}
	var out []provider.Comment
	for {
		cs, resp, err := p.c.Issues.ListComments(ctx, owner, repo, number, opts)
		if err != nil {
			return nil, err
		}
		for _, c := range cs {
			out = append(out, toComment(c))
		}
		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}
	return out, nil
}

func (p *Provider) AddComment(ctx context.Context, projectID string, number int, body string) (provider.Comment, error) {
	owner, repo, err := split(projectID)
	if err != nil {
		return provider.Comment{}, err
	}
	c, _, err := p.c.Issues.CreateComment(ctx, owner, repo, number, &gh.IssueComment{Body: &body})
	if err != nil {
		return provider.Comment{}, err
	}
	return toComment(c), nil
}

func (p *Provider) CreateIssue(ctx context.Context, projectID string, in provider.NewIssue) (provider.Issue, error) {
	owner, repo, err := split(projectID)
	if err != nil {
		return provider.Issue{}, err
	}
	req := gh.CreateIssueRequest{Title: in.Title, Labels: in.Labels}
	if in.Body != "" {
		req.Body = &in.Body
	}
	i, _, err := p.c.Issues.Create(ctx, owner, repo, req)
	if err != nil {
		return provider.Issue{}, err
	}
	return toIssue(i), nil
}

func (p *Provider) SetState(ctx context.Context, projectID string, number int, open bool) (provider.Issue, error) {
	owner, repo, err := split(projectID)
	if err != nil {
		return provider.Issue{}, err
	}
	state := "closed"
	if open {
		state = "open"
	}
	i, _, err := p.c.Issues.Update(ctx, owner, repo, number, gh.UpdateIssueRequest{State: &state})
	if err != nil {
		return provider.Issue{}, err
	}
	return toIssue(i), nil
}

func (p *Provider) ListLabels(ctx context.Context, projectID string) ([]provider.Label, error) {
	owner, repo, err := split(projectID)
	if err != nil {
		return nil, err
	}
	opts := &gh.ListOptions{PerPage: perPage}
	var out []provider.Label
	for {
		ls, resp, err := p.c.Issues.ListLabels(ctx, owner, repo, opts)
		if err != nil {
			return nil, err
		}
		out = append(out, toLabels(ls)...)
		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}
	sort.Slice(out, func(a, b int) bool { return strings.ToLower(out[a].Name) < strings.ToLower(out[b].Name) })
	return out, nil
}

func (p *Provider) SetIssueLabels(ctx context.Context, projectID string, number int, labels []string) ([]provider.Label, error) {
	owner, repo, err := split(projectID)
	if err != nil {
		return nil, err
	}
	if labels == nil {
		labels = []string{}
	}
	ls, _, err := p.c.Issues.ReplaceLabelsForIssue(ctx, owner, repo, number, labels)
	if err != nil {
		return nil, err
	}
	return toLabels(ls), nil
}

func toIssue(i *gh.Issue) provider.Issue {
	var assignees []string
	for _, a := range i.Assignees {
		assignees = append(assignees, a.GetLogin())
	}
	return provider.Issue{
		Number:    i.GetNumber(),
		Title:     i.GetTitle(),
		Body:      i.GetBody(),
		State:     i.GetState(),
		Author:    i.GetUser().GetLogin(),
		Labels:    toLabels(i.Labels),
		Assignees: assignees,
		Comments:  i.GetComments(),
		CreatedAt: i.GetCreatedAt().Time,
		UpdatedAt: i.GetUpdatedAt().Time,
		URL:       i.GetHTMLURL(),
	}
}

func toLabels(ls []*gh.Label) []provider.Label {
	out := make([]provider.Label, 0, len(ls))
	for _, l := range ls {
		out = append(out, provider.Label{Name: l.GetName(), Color: l.GetColor(), Description: l.GetDescription()})
	}
	return out
}

func toComment(c *gh.IssueComment) provider.Comment {
	return provider.Comment{
		ID:        c.GetID(),
		Author:    c.GetUser().GetLogin(),
		Body:      c.GetBody(),
		CreatedAt: c.GetCreatedAt().Time,
		URL:       c.GetHTMLURL(),
	}
}

// compile-time check
var _ provider.Provider = (*Provider)(nil)
