package ui

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"

	"github.com/ljunker/issuer/internal/provider"
)

// ---- projects ----

type projectItem struct {
	p      provider.Project
	pinned bool
}

func (i projectItem) FilterValue() string { return i.p.ID }

type projectDelegate struct{}

func (projectDelegate) Height() int                         { return 2 }
func (projectDelegate) Spacing() int                        { return 0 }
func (projectDelegate) Update(tea.Msg, *list.Model) tea.Cmd { return nil }
func (projectDelegate) Render(w io.Writer, m list.Model, index int, it list.Item) {
	item, ok := it.(projectItem)
	if !ok {
		return
	}
	width := m.Width()
	selected := index == m.Index()

	name := item.p.Name
	if item.pinned {
		name = "★ " + name
	}
	count := ""
	if item.p.OpenIssues > 0 {
		count = fmt.Sprintf(" %d", item.p.OpenIssues)
	}
	nameW := width - 2 - lipgloss.Width(count)
	line1 := ansi.Truncate(name, max(nameW, 1), "…")
	line1 = line1 + strings.Repeat(" ", max(0, nameW-lipgloss.Width(line1))) + count

	owner := item.p.Owner
	if item.p.Private {
		owner += " · privat"
	}
	line2 := ansi.Truncate(owner, max(width-2, 1), "…")

	prefix := "  "
	if selected {
		prefix = titleStyle.Render("▌ ")
		line1 = boldStyle.Render(line1)
	}
	fmt.Fprint(w, prefix+line1+"\n"+"  "+mutedStyle.Render(line2))
}

// ---- issues ----

type issueItem struct{ i provider.Issue }

func (i issueItem) FilterValue() string {
	var b strings.Builder
	fmt.Fprintf(&b, "#%d %s %s", i.i.Number, i.i.Title, i.i.Author)
	for _, l := range i.i.Labels {
		b.WriteString(" " + l.Name)
	}
	return b.String()
}

type issueDelegate struct{}

func (issueDelegate) Height() int                         { return 2 }
func (issueDelegate) Spacing() int                        { return 1 }
func (issueDelegate) Update(tea.Msg, *list.Model) tea.Cmd { return nil }
func (issueDelegate) Render(w io.Writer, m list.Model, index int, it list.Item) {
	item, ok := it.(issueItem)
	if !ok {
		return
	}
	is := item.i
	width := m.Width() - 2
	selected := index == m.Index()

	icon := lipgloss.NewStyle().Foreground(colOpen).Render("●")
	if !is.Open() {
		icon = lipgloss.NewStyle().Foreground(colClosed).Render("✔")
	}
	num := mutedStyle.Render(fmt.Sprintf("#%d", is.Number))
	title := is.Title
	if selected {
		title = boldStyle.Render(title)
	}
	line1 := ansi.Truncate(icon+" "+title+" "+num, width, "…")

	var chips []string
	for _, l := range is.Labels {
		chips = append(chips, labelChip(l.Name, l.Color))
	}
	meta := fmt.Sprintf("%s · %s", is.Author, relTime(is.UpdatedAt))
	if is.Comments > 0 {
		meta += fmt.Sprintf(" · 💬 %d", is.Comments)
	}
	if len(is.Assignees) > 0 {
		meta += " · → " + strings.Join(is.Assignees, ", ")
	}
	line2 := mutedStyle.Render(meta)
	if len(chips) > 0 {
		line2 = strings.Join(chips, " ") + "  " + line2
	}
	line2 = ansi.Truncate(line2, width-2, "…")

	prefix := "  "
	if selected {
		prefix = titleStyle.Render("▌ ")
	}
	fmt.Fprint(w, prefix+line1+"\n"+prefix+"  "+line2)
}

// ---- helpers ----

func relTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return "gerade eben"
	case d < time.Hour:
		return fmt.Sprintf("vor %d Min.", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("vor %d Std.", int(d.Hours()))
	case d < 30*24*time.Hour:
		return fmt.Sprintf("vor %d T.", int(d.Hours()/24))
	case d < 365*24*time.Hour:
		return fmt.Sprintf("vor %d Mon.", int(d.Hours()/24/30))
	default:
		return t.Format("02.01.2006")
	}
}

func newList(d list.ItemDelegate, title string) list.Model {
	l := list.New(nil, d, 0, 0)
	l.Title = title
	l.SetShowHelp(false)
	l.SetShowStatusBar(false)
	l.DisableQuitKeybindings()
	l.Styles.Title = titleStyle.Padding(0, 0)
	l.Styles.TitleBar = lipgloss.NewStyle().Padding(0, 0, 1, 1)
	l.FilterInput.Prompt = "/ "
	l.SetStatusBarItemName("Eintrag", "Einträge")
	return l
}
