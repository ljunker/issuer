package ui

import (
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type composeKind int

const (
	composeComment composeKind = iota
	composeIssue
)

// composeView is the form for a new comment or a new issue.
type composeView struct {
	kind    composeKind
	heading string
	title   textinput.Model
	body    textarea.Model
	focus   int // 0 = title, 1 = body (issue only)
	width   int
	height  int
}

func newComposeView(kind composeKind, heading string) composeView {
	ti := textinput.New()
	ti.Placeholder = "Titel"
	ti.Prompt = ""
	ti.CharLimit = 256

	ta := textarea.New()
	ta.Placeholder = "Beschreibung (Markdown) …"
	ta.ShowLineNumbers = false
	ta.CharLimit = 0
	ta.Prompt = "┃ "
	if kind == composeComment {
		ta.Placeholder = "Kommentar (Markdown) …"
	}

	c := composeView{kind: kind, heading: heading, title: ti, body: ta}
	if kind == composeIssue {
		c.title.Focus()
	} else {
		c.focus = 1
		c.body.Focus()
	}
	return c
}

func (c *composeView) setSize(w, h int) {
	c.width, c.height = w, h
	c.title.Width = w - 4
	c.body.SetWidth(w - 2)
	bodyH := h - 4 // heading + blank + help
	if c.kind == composeIssue {
		bodyH -= 3
	}
	c.body.SetHeight(max(bodyH, 3))
}

func (c composeView) valid() bool {
	if c.kind == composeIssue {
		return strings.TrimSpace(c.title.Value()) != ""
	}
	return strings.TrimSpace(c.body.Value()) != ""
}

func (c *composeView) update(msg tea.Msg) tea.Cmd {
	if km, ok := msg.(tea.KeyMsg); ok && c.kind == composeIssue && key.Matches(km, keys.NextFld) {
		if c.focus == 0 {
			c.focus = 1
			c.title.Blur()
			return c.body.Focus()
		}
		c.focus = 0
		c.body.Blur()
		return c.title.Focus()
	}
	var cmd tea.Cmd
	if c.focus == 0 {
		c.title, cmd = c.title.Update(msg)
	} else {
		c.body, cmd = c.body.Update(msg)
	}
	return cmd
}

func (c composeView) view() string {
	var b strings.Builder
	b.WriteString(" " + titleStyle.Render(c.heading) + "\n\n")
	if c.kind == composeIssue {
		label := mutedStyle.Render("Titel")
		if c.focus == 0 {
			label = titleStyle.Render("Titel")
		}
		b.WriteString(" " + label + "\n")
		b.WriteString(" " + lipgloss.NewStyle().Border(lipgloss.NormalBorder(), false, false, true, false).BorderForeground(colBorder).Render(c.title.View()) + "\n")
	}
	b.WriteString(c.body.View() + "\n")
	b.WriteString(mutedStyle.Render(" ctrl+s senden · esc abbrechen"))
	if c.kind == composeIssue {
		b.WriteString(mutedStyle.Render(" · tab Feld wechseln"))
	}
	return b.String()
}
