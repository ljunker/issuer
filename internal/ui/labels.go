package ui

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"

	"github.com/ljunker/issuer/internal/provider"
)

// labelPicker lets the user toggle the labels of an issue.
type labelPicker struct {
	all       []provider.Label
	selected  map[string]bool
	initial   map[string]bool
	cursor    int
	offset    int
	filter    textinput.Model
	filtering bool
	loading   bool
	width     int
	height    int
}

func newLabelPicker(current []provider.Label) labelPicker {
	sel := map[string]bool{}
	init := map[string]bool{}
	for _, l := range current {
		sel[l.Name] = true
		init[l.Name] = true
	}
	fi := textinput.New()
	fi.Prompt = "/ "
	fi.Placeholder = "Label suchen"
	return labelPicker{selected: sel, initial: init, filter: fi, loading: true}
}

func (p *labelPicker) setLabels(ls []provider.Label) {
	p.all = ls
	p.loading = false
	p.clamp()
}

func (p *labelPicker) setSize(w, h int) { p.width, p.height = w, h; p.clamp() }

func (p labelPicker) visible() []provider.Label {
	q := strings.ToLower(strings.TrimSpace(p.filter.Value()))
	if q == "" {
		return p.all
	}
	var out []provider.Label
	for _, l := range p.all {
		if strings.Contains(strings.ToLower(l.Name), q) {
			out = append(out, l)
		}
	}
	return out
}

func (p labelPicker) listHeight() int { return max(p.height-5, 1) }

func (p *labelPicker) clamp() {
	n := len(p.visible())
	p.cursor = min(max(p.cursor, 0), max(n-1, 0))
	h := p.listHeight()
	if p.cursor < p.offset {
		p.offset = p.cursor
	}
	if p.cursor >= p.offset+h {
		p.offset = p.cursor - h + 1
	}
}

func (p labelPicker) changed() bool {
	if len(p.selected) != len(p.initial) {
		return true
	}
	for k := range p.selected {
		if !p.initial[k] {
			return true
		}
	}
	return false
}

func (p labelPicker) names() []string {
	out := make([]string, 0, len(p.selected))
	for k := range p.selected {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// update handles keys; it returns done=true when the user confirmed (enter
// outside the filter field).
func (p *labelPicker) update(msg tea.Msg) (cmd tea.Cmd, done bool) {
	km, ok := msg.(tea.KeyMsg)
	if !ok {
		return nil, false
	}
	if p.filtering {
		switch km.String() {
		case "enter", "esc":
			p.filtering = false
			p.filter.Blur()
			if km.String() == "esc" {
				p.filter.SetValue("")
			}
			p.clamp()
			return nil, false
		case "up", "down":
		default:
			p.filter, cmd = p.filter.Update(msg)
			p.cursor = 0
			p.clamp()
			return cmd, false
		}
	}
	switch {
	case key.Matches(km, keys.Up):
		p.cursor--
	case key.Matches(km, keys.Down):
		p.cursor++
	case key.Matches(km, keys.Select):
		if vis := p.visible(); len(vis) > 0 {
			name := vis[p.cursor].Name
			if p.selected[name] {
				delete(p.selected, name)
			} else {
				p.selected[name] = true
			}
		}
	case key.Matches(km, keys.Filter):
		p.filtering = true
		return p.filter.Focus(), false
	case km.String() == "enter":
		return nil, true
	}
	p.clamp()
	return nil, false
}

func (p labelPicker) view(heading string) string {
	var b strings.Builder
	b.WriteString(" " + titleStyle.Render(heading) + "\n")
	if p.filtering || p.filter.Value() != "" {
		b.WriteString(" " + p.filter.View() + "\n")
	} else {
		b.WriteString("\n")
	}
	if p.loading {
		b.WriteString("\n " + mutedStyle.Render("Labels werden geladen …"))
		return b.String()
	}
	vis := p.visible()
	if len(vis) == 0 {
		b.WriteString("\n " + mutedStyle.Render("Keine Labels gefunden."))
	}
	h := p.listHeight()
	for i := p.offset; i < len(vis) && i < p.offset+h; i++ {
		l := vis[i]
		box := "[ ]"
		if p.selected[l.Name] {
			box = titleStyle.Render("[x]")
		}
		cur := "  "
		if i == p.cursor {
			cur = titleStyle.Render("▌ ")
		}
		line := fmt.Sprintf("%s%s %s", cur, box, labelChip(l.Name, l.Color))
		if l.Description != "" {
			line += " " + mutedStyle.Render(l.Description)
		}
		b.WriteString("\n" + ansi.Truncate(line, p.width-1, "…"))
	}
	b.WriteString("\n\n" + mutedStyle.Render(" space umschalten · / suchen · enter übernehmen · esc abbrechen"))
	return b.String()
}
