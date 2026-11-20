package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"

	"github.com/ljunker/issuer/internal/provider"
)

// detailView shows one issue with its body and comments.
type detailView struct {
	issue    provider.Issue
	comments []provider.Comment
	loading  bool
	vp       viewport.Model

	glamourStyle string
	renderer     *glamour.TermRenderer
	rendererW    int
}

func newDetailView(style string) detailView {
	return detailView{vp: viewport.New(0, 0), glamourStyle: style}
}

func (d *detailView) setSize(w, h int) {
	d.vp.Width, d.vp.Height = w, h
	d.refresh(false)
}

func (d *detailView) show(i provider.Issue) {
	d.issue = i
	d.comments = nil
	d.loading = true
	d.refresh(true)
}

func (d *detailView) setComments(cs []provider.Comment) {
	d.comments = cs
	d.loading = false
	d.refresh(false)
}

func (d *detailView) markdown(s string) string {
	w := max(d.vp.Width-2, 20)
	if d.renderer == nil || d.rendererW != w {
		r, err := glamour.NewTermRenderer(
			glamour.WithStandardStyle(d.glamourStyle),
			glamour.WithWordWrap(w),
			glamour.WithEmoji(),
		)
		if err != nil {
			return s
		}
		d.renderer, d.rendererW = r, w
	}
	out, err := d.renderer.Render(s)
	if err != nil {
		return s
	}
	return strings.Trim(out, "\n")
}

func (d *detailView) refresh(top bool) {
	if d.vp.Width == 0 {
		return
	}
	i := d.issue
	w := d.vp.Width
	var b strings.Builder

	title := lipgloss.NewStyle().Bold(true).Width(w - 2).Render(fmt.Sprintf("%s %s", i.Title, mutedStyle.Render(fmt.Sprintf("#%d", i.Number))))
	b.WriteString(" " + title + "\n\n ")
	if i.Open() {
		b.WriteString(openBadge.Render("● offen"))
	} else {
		b.WriteString(closedBadge.Render("✔ geschlossen"))
	}
	b.WriteString(dimStyle.Render(fmt.Sprintf("  %s eröffnet %s · %d Kommentare", i.Author, relTime(i.CreatedAt), i.Comments)))
	b.WriteString("\n")

	if len(i.Labels) > 0 {
		var chips []string
		for _, l := range i.Labels {
			chips = append(chips, labelChip(l.Name, l.Color))
		}
		b.WriteString("\n " + mutedStyle.Render("Labels    ") + strings.Join(chips, " ") + "\n")
	}
	if len(i.Assignees) > 0 {
		b.WriteString(" " + mutedStyle.Render("Zugewiesen ") + strings.Join(i.Assignees, ", ") + "\n")
	}

	b.WriteString("\n" + mutedStyle.Render(strings.Repeat("─", w)) + "\n")
	body := strings.TrimSpace(i.Body)
	if body == "" {
		b.WriteString("\n " + mutedStyle.Italic(true).Render("Keine Beschreibung.") + "\n")
	} else {
		b.WriteString(d.markdown(body) + "\n")
	}

	switch {
	case d.loading:
		b.WriteString("\n " + mutedStyle.Render("Kommentare werden geladen …") + "\n")
	case len(d.comments) > 0:
		for _, c := range d.comments {
			head := fmt.Sprintf(" %s %s", boldStyle.Render(c.Author), mutedStyle.Render(relTime(c.CreatedAt)))
			b.WriteString("\n" + mutedStyle.Render(strings.Repeat("─", w)) + "\n" + head + "\n")
			b.WriteString(d.markdown(c.Body) + "\n")
		}
	}

	d.vp.SetContent(b.String())
	if top {
		d.vp.GotoTop()
	}
}

func (d *detailView) update(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd
	d.vp, cmd = d.vp.Update(msg)
	return cmd
}

func (d detailView) view() string { return d.vp.View() }
