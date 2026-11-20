// Package ui implements the Bubble Tea terminal interface.
package ui

import (
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"

	"github.com/ljunker/issuer/internal/config"
	"github.com/ljunker/issuer/internal/provider"
)

type focusArea int

const (
	focusProjects focusArea = iota
	focusRight
)

type rightMode int

const (
	modeIssues rightMode = iota
	modeDetail
	modeCompose
	modeLabels
)

const selectDebounce = 200 * time.Millisecond

type selectProjectMsg struct{ seq int }

type confirmPrompt struct {
	text string
	cmd  tea.Cmd
}

type Model struct {
	prov provider.Provider
	cfg  config.Config

	width, height int
	focus         focusArea
	mode          rightMode
	returnMode    rightMode // where compose/labels go back to

	projects list.Model
	issues   list.Model
	detail   detailView
	compose  composeView
	labels   labelPicker
	// target of compose (comment) / label picker
	targetIssue provider.Issue

	state         provider.State
	currentID     string
	highlightedID string
	selectSeq     int
	projectsReady bool

	issueCache    map[string][]provider.Issue // key: projectID|state
	labelCache    map[string][]provider.Label
	loadingIssues map[string]bool

	spinner  spinner.Model
	busy     int
	help     help.Model
	fullHelp bool
	confirm  *confirmPrompt
	err      error
	info     string
	infoID   int
}

// New builds the root model. glamourStyle is "dark" or "light".
func New(p provider.Provider, cfg config.Config, glamourStyle string) Model {
	sp := spinner.New(spinner.WithSpinner(spinner.MiniDot), spinner.WithStyle(titleStyle))
	h := help.New()
	h.ShortSeparator = " · "
	return Model{
		prov:          p,
		cfg:           cfg,
		projects:      newList(projectDelegate{}, "Projekte"),
		issues:        newList(issueDelegate{}, "Issues"),
		detail:        newDetailView(glamourStyle),
		state:         cfg.DefaultState,
		issueCache:    map[string][]provider.Issue{},
		labelCache:    map[string][]provider.Label{},
		loadingIssues: map[string]bool{},
		spinner:       sp,
		help:          h,
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(m.startBusy(), loadProjects(m.prov))
}

// ---------------------------------------------------------------- helpers

func cacheKey(projectID string, s provider.State) string { return projectID + "|" + string(s) }

func (m *Model) startBusy() tea.Cmd {
	m.busy++
	if m.busy == 1 {
		return m.spinner.Tick
	}
	return nil
}

func (m *Model) doneBusy() {
	if m.busy > 0 {
		m.busy--
	}
}

func (m *Model) setInfo(s string) tea.Cmd {
	m.err = nil
	m.info = s
	m.infoID++
	return clearInfoAfter(m.infoID)
}

func (m *Model) setErr(err error) {
	m.info = ""
	m.err = err
}

func (m Model) currentProject() (provider.Project, bool) {
	for _, it := range m.projects.Items() {
		if p := it.(projectItem).p; p.ID == m.currentID {
			return p, true
		}
	}
	return provider.Project{}, false
}

// selectedIssue is the issue the actions (comment, labels, close) apply to.
func (m Model) selectedIssue() (provider.Issue, bool) {
	if m.mode == modeDetail {
		return m.detail.issue, true
	}
	if it, ok := m.issues.SelectedItem().(issueItem); ok {
		return it.i, true
	}
	return provider.Issue{}, false
}

func (m *Model) setIssueItems() tea.Cmd {
	cached := m.issueCache[cacheKey(m.currentID, m.state)]
	items := make([]list.Item, len(cached))
	for i, is := range cached {
		items[i] = issueItem{is}
	}
	cmd := m.issues.SetItems(items)
	// bubbles/list measures the pagination row before updating the page
	// count, so re-apply the size once the items are known.
	m.layout()
	return cmd
}

func (m *Model) selectProject(id string) tea.Cmd {
	if id == "" {
		return nil
	}
	changed := id != m.currentID
	m.currentID = id
	if changed {
		m.mode = modeIssues
		m.issues.ResetFilter()
		m.issues.Select(0)
	}
	cmd := m.setIssueItems()
	if _, ok := m.issueCache[cacheKey(id, m.state)]; ok {
		return cmd
	}
	return tea.Batch(cmd, m.fetchIssues(id))
}

func (m *Model) fetchIssues(id string) tea.Cmd {
	k := cacheKey(id, m.state)
	if m.loadingIssues[k] {
		return nil
	}
	m.loadingIssues[k] = true
	return tea.Batch(m.startBusy(), loadIssues(m.prov, id, m.state))
}

func matchesState(i provider.Issue, s provider.State) bool {
	return s == provider.StateAll || string(s) == i.State
}

// upsertIssue updates an issue in every cached list of its project, adding
// or removing it depending on each list's state filter.
func (m *Model) upsertIssue(projectID string, is provider.Issue) tea.Cmd {
	for _, s := range []provider.State{provider.StateOpen, provider.StateClosed, provider.StateAll} {
		k := cacheKey(projectID, s)
		list, ok := m.issueCache[k]
		if !ok {
			continue
		}
		idx := slices.IndexFunc(list, func(x provider.Issue) bool { return x.Number == is.Number })
		switch {
		case matchesState(is, s) && idx >= 0:
			list[idx] = is
		case matchesState(is, s):
			list = append([]provider.Issue{is}, list...)
		case idx >= 0:
			list = slices.Delete(list, idx, idx+1)
		}
		m.issueCache[k] = list
	}
	if m.detail.issue.Number == is.Number && projectID == m.currentID {
		m.detail.issue = is
		m.detail.refresh(false)
	}
	if projectID == m.currentID {
		return m.setIssueItems()
	}
	return nil
}

func (m *Model) sortedProjectItems(ps []provider.Project) []list.Item {
	pinned := map[string]int{}
	for i, id := range m.cfg.Pinned {
		pinned[strings.ToLower(id)] = i + 1
	}
	items := make([]list.Item, 0, len(ps))
	var pinnedItems []projectItem
	for _, p := range ps {
		if _, ok := pinned[strings.ToLower(p.ID)]; ok {
			pinnedItems = append(pinnedItems, projectItem{p: p, pinned: true})
			continue
		}
		items = append(items, projectItem{p: p})
	}
	slices.SortFunc(pinnedItems, func(a, b projectItem) int {
		return pinned[strings.ToLower(a.p.ID)] - pinned[strings.ToLower(b.p.ID)]
	})
	head := make([]list.Item, len(pinnedItems))
	for i, p := range pinnedItems {
		head[i] = p
	}
	return append(head, items...)
}

func (m *Model) highlighted() string {
	if it, ok := m.projects.SelectedItem().(projectItem); ok {
		return it.p.ID
	}
	return ""
}

// ---------------------------------------------------------------- layout

func (m Model) leftWidth() int {
	return min(max(m.width*28/100, 26), 48)
}

func (m Model) helpView() string {
	m.help.ShowAll = m.fullHelp
	m.help.Width = m.width - 2
	return " " + m.help.View(m.helpKeys())
}

func (m *Model) layout() {
	mainH := m.height - 1 - lipgloss.Height(m.helpView())
	lw := m.leftWidth()
	rw := m.width - lw
	m.projects.SetSize(lw-2, mainH-2)
	m.issues.SetSize(rw-2, mainH-2)
	m.detail.setSize(rw-2, mainH-4)
	// compose and labels exist only while their mode is active
	switch m.mode {
	case modeCompose:
		m.compose.setSize(rw-2, mainH-2)
	case modeLabels:
		m.labels.setSize(rw-2, mainH-2)
	}
}

// ---------------------------------------------------------------- update

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.layout()
		return m, nil

	case spinner.TickMsg:
		if m.busy == 0 {
			return m, nil
		}
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case clearInfoMsg:
		if msg.id == m.infoID {
			m.info = ""
		}
		return m, nil

	case errMsg:
		m.setErr(msg.err)
		return m, nil

	case selectProjectMsg:
		if msg.seq != m.selectSeq {
			return m, nil
		}
		return m, m.selectProject(m.highlightedID)

	case projectsMsg:
		m.doneBusy()
		if msg.err != nil {
			m.setErr(fmt.Errorf("Projekte laden: %w", msg.err))
			return m, nil
		}
		prev := m.highlighted()
		cmd := m.projects.SetItems(m.sortedProjectItems(msg.projects))
		m.projects.Title = fmt.Sprintf("Projekte · %s", m.prov.Name())
		if prev != "" {
			for i, it := range m.projects.Items() {
				if it.(projectItem).p.ID == prev {
					m.projects.Select(i)
				}
			}
		}
		m.projectsReady = true
		m.layout()
		m.highlightedID = m.highlighted()
		return m, tea.Batch(cmd, m.selectProject(m.highlightedID))

	case issuesMsg:
		m.doneBusy()
		k := cacheKey(msg.projectID, msg.state)
		delete(m.loadingIssues, k)
		if msg.err != nil {
			m.setErr(fmt.Errorf("Issues von %s: %w", msg.projectID, msg.err))
			return m, nil
		}
		m.issueCache[k] = msg.issues
		if msg.projectID == m.currentID && msg.state == m.state {
			return m, m.setIssueItems()
		}
		return m, nil

	case commentsMsg:
		m.doneBusy()
		if msg.err != nil {
			m.setErr(fmt.Errorf("Kommentare: %w", msg.err))
			m.detail.loading = false
			m.detail.refresh(false)
			return m, nil
		}
		if msg.projectID == m.currentID && msg.number == m.detail.issue.Number {
			m.detail.setComments(msg.comments)
		}
		return m, nil

	case repoLabelsMsg:
		m.doneBusy()
		if msg.err != nil {
			m.setErr(fmt.Errorf("Labels: %w", msg.err))
			if m.mode == modeLabels {
				m.mode = m.returnMode
			}
			return m, nil
		}
		m.labelCache[msg.projectID] = msg.labels
		if m.mode == modeLabels && msg.projectID == m.currentID {
			m.labels.setLabels(msg.labels)
		}
		return m, nil

	case commentAddedMsg:
		m.doneBusy()
		if msg.err != nil {
			m.setErr(fmt.Errorf("Kommentar senden: %w", msg.err))
			return m, nil
		}
		is := m.targetIssue
		inDetail := m.mode == modeDetail && msg.projectID == m.currentID && m.detail.issue.Number == msg.number
		if inDetail {
			is = m.detail.issue
			m.detail.comments = append(m.detail.comments, msg.comment)
		}
		is.Comments++
		cmd := m.upsertIssue(msg.projectID, is)
		if inDetail {
			m.detail.vp.GotoBottom()
		}
		return m, tea.Batch(cmd, m.setInfo("Kommentar gesendet"))

	case issueCreatedMsg:
		m.doneBusy()
		if msg.err != nil {
			m.setErr(fmt.Errorf("Issue anlegen: %w", msg.err))
			return m, nil
		}
		cmd := m.upsertIssue(msg.projectID, msg.issue)
		if msg.projectID == m.currentID {
			m.mode = modeDetail
			m.focus = focusRight
			m.detail.show(msg.issue)
			m.detail.setComments(nil)
		}
		return m, tea.Batch(cmd, m.setInfo(fmt.Sprintf("Issue #%d angelegt", msg.issue.Number)))

	case issueUpdatedMsg:
		m.doneBusy()
		if msg.err != nil {
			m.setErr(msg.err)
			return m, nil
		}
		return m, tea.Batch(m.upsertIssue(msg.projectID, msg.issue), m.setInfo(msg.info))

	case issueLabelsSetMsg:
		m.doneBusy()
		if msg.err != nil {
			m.setErr(fmt.Errorf("Labels setzen: %w", msg.err))
			return m, nil
		}
		is := m.targetIssue
		if m.detail.issue.Number == msg.number {
			is = m.detail.issue
		}
		is.Labels = msg.labels
		return m, tea.Batch(m.upsertIssue(msg.projectID, is), m.setInfo("Labels aktualisiert"))

	case tea.KeyMsg:
		return m.handleKey(msg)
	}

	// Forward everything else (e.g. cursor blink) to the active component.
	return m, m.forward(msg)
}

func (m *Model) forward(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd
	switch m.mode {
	case modeCompose:
		cmd = m.compose.update(msg)
	case modeLabels:
		m.labels.filter, cmd = m.labels.filter.Update(msg)
	default:
		var c1, c2 tea.Cmd
		m.projects, c1 = m.projects.Update(msg)
		m.issues, c2 = m.issues.Update(msg)
		cmd = tea.Batch(c1, c2)
	}
	return cmd
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if msg.String() == "ctrl+c" {
		return m, tea.Quit
	}

	// pending yes/no question
	if m.confirm != nil {
		c := m.confirm
		switch {
		case key.Matches(msg, keys.Yes):
			m.confirm = nil
			return m, tea.Batch(m.startBusy(), c.cmd)
		case key.Matches(msg, keys.No):
			m.confirm = nil
		}
		return m, nil
	}

	switch m.mode {
	case modeCompose:
		return m.handleComposeKey(msg)
	case modeLabels:
		return m.handleLabelsKey(msg)
	}

	// a list is in filter-input mode: it gets every key
	if m.focus == focusProjects && m.projects.SettingFilter() {
		return m.updateProjects(msg)
	}
	if m.focus == focusRight && m.mode == modeIssues && m.issues.SettingFilter() {
		var cmd tea.Cmd
		m.issues, cmd = m.issues.Update(msg)
		return m, cmd
	}

	m.err = nil

	switch {
	case key.Matches(msg, keys.Quit):
		return m, tea.Quit
	case key.Matches(msg, keys.Help):
		m.fullHelp = !m.fullHelp
		m.layout()
		return m, nil
	case key.Matches(msg, keys.Tab):
		if m.focus == focusProjects {
			m.focus = focusRight
		} else {
			m.focus = focusProjects
		}
		return m, nil
	case key.Matches(msg, keys.Reload):
		return m, m.reload()
	case msg.String() == "R":
		return m, tea.Batch(m.startBusy(), loadProjects(m.prov))
	case key.Matches(msg, keys.State):
		m.state = m.state.Next()
		m.mode = modeIssues
		m.issues.ResetFilter()
		m.issues.Select(0)
		return m, m.selectProject(m.currentID)
	case key.Matches(msg, keys.New):
		return m, m.startNewIssue()
	case key.Matches(msg, keys.Browser):
		return m, m.openInBrowser()
	}

	if m.focus == focusProjects {
		if key.Matches(msg, keys.Open) {
			m.focus = focusRight
			if m.highlightedID != m.currentID {
				m.selectSeq++
				return m, m.selectProject(m.highlightedID)
			}
			return m, nil
		}
		return m.updateProjects(msg)
	}

	// right side
	switch {
	case key.Matches(msg, keys.Comment):
		return m, m.startComment()
	case key.Matches(msg, keys.Labels):
		return m, m.startLabels()
	case key.Matches(msg, keys.Toggle):
		m.askToggle()
		return m, nil
	}

	if m.mode == modeDetail {
		if key.Matches(msg, keys.Back) {
			m.mode = modeIssues
			return m, nil
		}
		return m, m.detail.update(msg)
	}

	// issue list
	switch {
	case key.Matches(msg, keys.Open):
		if is, ok := m.selectedIssue(); ok {
			m.mode = modeDetail
			m.detail.show(is)
			return m, tea.Batch(m.startBusy(), loadComments(m.prov, m.currentID, is.Number))
		}
		return m, nil
	case key.Matches(msg, keys.Back) && m.issues.FilterState() == list.Unfiltered:
		m.focus = focusProjects
		return m, nil
	}
	var cmd tea.Cmd
	m.issues, cmd = m.issues.Update(msg)
	return m, cmd
}

func (m Model) updateProjects(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.projects, cmd = m.projects.Update(msg)
	if id := m.highlighted(); id != m.highlightedID {
		m.highlightedID = id
		m.selectSeq++
		seq := m.selectSeq
		tick := tea.Tick(selectDebounce, func(time.Time) tea.Msg { return selectProjectMsg{seq: seq} })
		return m, tea.Batch(cmd, tick)
	}
	return m, cmd
}

func (m Model) handleComposeKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, keys.Cancel):
		m.mode = m.returnMode
		return m, nil
	case key.Matches(msg, keys.Submit):
		if !m.compose.valid() {
			if m.compose.kind == composeIssue {
				m.setErr(fmt.Errorf("Titel fehlt"))
			} else {
				m.setErr(fmt.Errorf("Kommentar ist leer"))
			}
			return m, nil
		}
		m.err = nil
		m.mode = m.returnMode
		if m.compose.kind == composeIssue {
			in := provider.NewIssue{
				Title: strings.TrimSpace(m.compose.title.Value()),
				Body:  strings.TrimSpace(m.compose.body.Value()),
			}
			return m, tea.Batch(m.startBusy(), createIssue(m.prov, m.currentID, in))
		}
		body := strings.TrimSpace(m.compose.body.Value())
		return m, tea.Batch(m.startBusy(), addComment(m.prov, m.currentID, m.targetIssue.Number, body))
	}
	return m, m.compose.update(msg)
}

func (m Model) handleLabelsKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if !m.labels.filtering && key.Matches(msg, keys.Cancel) {
		m.mode = m.returnMode
		return m, nil
	}
	cmd, done := m.labels.update(msg)
	if !done {
		return m, cmd
	}
	m.mode = m.returnMode
	if !m.labels.changed() {
		return m, nil
	}
	return m, tea.Batch(m.startBusy(), setIssueLabels(m.prov, m.currentID, m.targetIssue.Number, m.labels.names()))
}

// ---------------------------------------------------------------- actions

func (m *Model) reload() tea.Cmd {
	if m.currentID == "" {
		return nil
	}
	for _, s := range []provider.State{provider.StateOpen, provider.StateClosed, provider.StateAll} {
		if s != m.state {
			delete(m.issueCache, cacheKey(m.currentID, s))
		}
	}
	delete(m.labelCache, m.currentID)
	cmds := []tea.Cmd{m.fetchIssues(m.currentID)}
	if m.mode == modeDetail {
		m.detail.loading = true
		cmds = append(cmds, m.startBusy(), loadComments(m.prov, m.currentID, m.detail.issue.Number))
	}
	return tea.Batch(cmds...)
}

func (m *Model) startNewIssue() tea.Cmd {
	p, ok := m.currentProject()
	if !ok {
		return nil
	}
	m.returnMode = m.mode
	m.mode = modeCompose
	m.focus = focusRight
	m.compose = newComposeView(composeIssue, "Neues Issue in "+p.ID)
	m.layout()
	return textarea.Blink
}

func (m *Model) startComment() tea.Cmd {
	is, ok := m.selectedIssue()
	if !ok {
		return nil
	}
	m.targetIssue = is
	m.returnMode = m.mode
	m.mode = modeCompose
	m.compose = newComposeView(composeComment, fmt.Sprintf("Kommentar zu #%d %s", is.Number, is.Title))
	m.layout()
	return textarea.Blink
}

func (m *Model) startLabels() tea.Cmd {
	is, ok := m.selectedIssue()
	if !ok {
		return nil
	}
	m.targetIssue = is
	m.returnMode = m.mode
	m.mode = modeLabels
	m.labels = newLabelPicker(is.Labels)
	m.layout()
	if ls, ok := m.labelCache[m.currentID]; ok {
		m.labels.setLabels(ls)
		return nil
	}
	return tea.Batch(m.startBusy(), loadRepoLabels(m.prov, m.currentID))
}

func (m *Model) askToggle() {
	is, ok := m.selectedIssue()
	if !ok {
		return
	}
	verb := "schließen"
	if !is.Open() {
		verb = "wieder öffnen"
	}
	m.confirm = &confirmPrompt{
		text: fmt.Sprintf("#%d %s? (y/n)", is.Number, verb),
		cmd:  setState(m.prov, m.currentID, is.Number, !is.Open()),
	}
}

func (m *Model) openInBrowser() tea.Cmd {
	if m.focus == focusRight {
		if is, ok := m.selectedIssue(); ok && is.URL != "" {
			return openBrowser(is.URL)
		}
	}
	if it, ok := m.projects.SelectedItem().(projectItem); ok && it.p.URL != "" {
		return openBrowser(it.p.URL)
	}
	return nil
}

// ---------------------------------------------------------------- view

func (m Model) helpKeys() helpKeys {
	switch {
	case m.confirm != nil:
		return helpKeys{short: []key.Binding{keys.Yes, keys.No}}
	case m.mode == modeCompose:
		ks := []key.Binding{keys.Submit, keys.Cancel}
		if m.compose.kind == composeIssue {
			ks = append(ks, keys.NextFld)
		}
		return helpKeys{short: ks}
	case m.mode == modeLabels:
		return helpKeys{short: []key.Binding{keys.Select, keys.Filter, keys.Open, keys.Cancel}}
	}
	nav := []key.Binding{keys.Up, keys.Down, keys.Tab, keys.Open, keys.Back, keys.Filter}
	act := []key.Binding{keys.New, keys.Comment, keys.Labels, keys.Toggle}
	misc := []key.Binding{keys.State, keys.Reload,
		key.NewBinding(key.WithKeys("R"), key.WithHelp("R", "Projekte neu laden")),
		keys.Browser, keys.Help, keys.Quit}
	full := [][]key.Binding{nav, act, misc}
	if m.focus == focusProjects {
		return helpKeys{short: []key.Binding{keys.Open, keys.Filter, keys.Tab, keys.New, keys.State, keys.Help, keys.Quit}, full: full}
	}
	if m.mode == modeDetail {
		return helpKeys{short: []key.Binding{keys.Back, keys.Comment, keys.Labels, keys.Toggle, keys.Browser, keys.Help, keys.Quit}, full: full}
	}
	return helpKeys{short: []key.Binding{keys.Open, keys.New, keys.Comment, keys.Labels, keys.Toggle, keys.State, keys.Filter, keys.Help, keys.Quit}, full: full}
}

func (m Model) View() string {
	if m.width == 0 {
		return ""
	}
	helpV := m.helpView()
	mainH := m.height - 1 - lipgloss.Height(helpV)
	lw := m.leftWidth()
	rw := m.width - lw

	left := m.projectsView()
	right := m.rightView(rw - 2)

	ls, rs := paneStyle, paneStyle
	if m.focus == focusProjects {
		ls = paneFocusedStyle
	} else {
		rs = paneFocusedStyle
	}
	clip := lipgloss.NewStyle().MaxHeight(mainH - 2)
	main := lipgloss.JoinHorizontal(lipgloss.Top,
		ls.Width(lw-2).Height(mainH-2).Render(clip.Render(left)),
		rs.Width(rw-2).Height(mainH-2).Render(clip.Render(right)),
	)
	return lipgloss.JoinVertical(lipgloss.Left, main, m.statusView(), helpV)
}

func (m Model) projectsView() string {
	if !m.projectsReady {
		return " " + titleStyle.Render("Projekte") + "\n\n " + m.spinner.View() + mutedStyle.Render(" lade Projekte …")
	}
	if len(m.projects.Items()) == 0 {
		return " " + titleStyle.Render("Projekte") + "\n\n " + mutedStyle.Render("Keine Projekte gefunden.")
	}
	return m.projects.View()
}

func (m Model) rightView(w int) string {
	p, ok := m.currentProject()
	if !ok {
		return "\n " + mutedStyle.Render("Wähle links ein Projekt aus.")
	}
	switch m.mode {
	case modeCompose:
		return m.compose.view()
	case modeLabels:
		return m.labels.view(fmt.Sprintf("Labels für #%d %s", m.targetIssue.Number, m.targetIssue.Title))
	case modeDetail:
		crumb := " " + mutedStyle.Render(p.ID+" › ") + titleStyle.Render(fmt.Sprintf("#%d", m.detail.issue.Number))
		if pct := m.detail.vp.ScrollPercent(); m.detail.vp.TotalLineCount() > m.detail.vp.Height {
			crumb += mutedStyle.Render(fmt.Sprintf("  %3.0f%%", pct*100))
		}
		return crumb + "\n\n" + m.detail.view()
	}

	stateName := map[provider.State]string{
		provider.StateOpen: "offen", provider.StateClosed: "geschlossen", provider.StateAll: "alle",
	}[m.state]
	k := cacheKey(m.currentID, m.state)
	_, loaded := m.issueCache[k]
	title := fmt.Sprintf("%s · %s", p.ID, stateName)
	if loaded {
		title += fmt.Sprintf(" (%d)", len(m.issueCache[k]))
	}
	if m.issues.IsFiltered() {
		title += fmt.Sprintf(" · Filter „%s“ (esc entfernt)", m.issues.FilterValue())
	}
	if m.loadingIssues[k] {
		title += " " + m.spinner.View()
	}
	l := m.issues
	l.Title = title
	if !loaded || len(m.issueCache[k]) == 0 {
		msg := "Lade Issues …"
		if loaded {
			msg = "Keine Issues (" + stateName + "). n legt ein neues an, s wechselt den Filter."
		}
		return " " + titleStyle.Render(ansi.Truncate(title, w-2, "…")) + "\n\n " + mutedStyle.Render(msg)
	}
	return l.View()
}

func (m Model) statusView() string {
	var s string
	switch {
	case m.confirm != nil:
		s = lipgloss.NewStyle().Bold(true).Foreground(colAccent).Render("? " + m.confirm.text)
	case m.err != nil:
		s = errorStyle.Render("✗ " + m.err.Error())
	case m.info != "":
		s = infoStyle.Render("✓ " + m.info)
	case m.busy > 0:
		s = m.spinner.View() + mutedStyle.Render(" lade …")
	}
	return " " + ansi.Truncate(s, m.width-2, "…")
}
