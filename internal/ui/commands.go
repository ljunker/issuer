package ui

import (
	"context"
	"os/exec"
	"runtime"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/ljunker/issuer/internal/provider"
)

const apiTimeout = 30 * time.Second

type (
	projectsMsg struct {
		projects []provider.Project
		err      error
	}
	issuesMsg struct {
		projectID string
		state     provider.State
		issues    []provider.Issue
		err       error
	}
	commentsMsg struct {
		projectID string
		number    int
		comments  []provider.Comment
		err       error
	}
	repoLabelsMsg struct {
		projectID string
		labels    []provider.Label
		err       error
	}
	commentAddedMsg struct {
		projectID string
		number    int
		comment   provider.Comment
		err       error
	}
	issueCreatedMsg struct {
		projectID string
		issue     provider.Issue
		err       error
	}
	issueUpdatedMsg struct {
		projectID string
		issue     provider.Issue
		info      string
		err       error
	}
	issueLabelsSetMsg struct {
		projectID string
		number    int
		labels    []provider.Label
		err       error
	}
	errMsg       struct{ err error }
	clearInfoMsg struct{ id int }
)

func withTimeout() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), apiTimeout)
}

func loadProjects(p provider.Provider) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := withTimeout()
		defer cancel()
		ps, err := p.ListProjects(ctx)
		return projectsMsg{projects: ps, err: err}
	}
}

func loadIssues(p provider.Provider, projectID string, state provider.State) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := withTimeout()
		defer cancel()
		is, err := p.ListIssues(ctx, projectID, provider.IssueFilter{State: state})
		return issuesMsg{projectID: projectID, state: state, issues: is, err: err}
	}
}

func loadComments(p provider.Provider, projectID string, number int) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := withTimeout()
		defer cancel()
		cs, err := p.ListComments(ctx, projectID, number)
		return commentsMsg{projectID: projectID, number: number, comments: cs, err: err}
	}
}

func loadRepoLabels(p provider.Provider, projectID string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := withTimeout()
		defer cancel()
		ls, err := p.ListLabels(ctx, projectID)
		return repoLabelsMsg{projectID: projectID, labels: ls, err: err}
	}
}

func addComment(p provider.Provider, projectID string, number int, body string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := withTimeout()
		defer cancel()
		c, err := p.AddComment(ctx, projectID, number, body)
		return commentAddedMsg{projectID: projectID, number: number, comment: c, err: err}
	}
}

func createIssue(p provider.Provider, projectID string, in provider.NewIssue) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := withTimeout()
		defer cancel()
		i, err := p.CreateIssue(ctx, projectID, in)
		return issueCreatedMsg{projectID: projectID, issue: i, err: err}
	}
}

func setState(p provider.Provider, projectID string, number int, open bool) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := withTimeout()
		defer cancel()
		i, err := p.SetState(ctx, projectID, number, open)
		info := "Issue geschlossen"
		if open {
			info = "Issue wieder geöffnet"
		}
		return issueUpdatedMsg{projectID: projectID, issue: i, info: info, err: err}
	}
}

func setIssueLabels(p provider.Provider, projectID string, number int, labels []string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := withTimeout()
		defer cancel()
		ls, err := p.SetIssueLabels(ctx, projectID, number, labels)
		return issueLabelsSetMsg{projectID: projectID, number: number, labels: ls, err: err}
	}
}

func openBrowser(url string) tea.Cmd {
	return func() tea.Msg {
		var cmd *exec.Cmd
		switch runtime.GOOS {
		case "darwin":
			cmd = exec.Command("open", url)
		case "windows":
			cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
		default:
			cmd = exec.Command("xdg-open", url)
		}
		if err := cmd.Start(); err != nil {
			return errMsg{err}
		}
		go cmd.Wait() //nolint:errcheck
		return nil
	}
}

func clearInfoAfter(id int) tea.Cmd {
	return tea.Tick(4*time.Second, func(time.Time) tea.Msg { return clearInfoMsg{id: id} })
}
