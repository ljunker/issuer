package ui

import "github.com/charmbracelet/bubbles/key"

type keyMap struct {
	Tab     key.Binding
	Open    key.Binding
	Back    key.Binding
	New     key.Binding
	Comment key.Binding
	Labels  key.Binding
	Toggle  key.Binding
	State   key.Binding
	Reload  key.Binding
	Browser key.Binding
	Filter  key.Binding
	Help    key.Binding
	Quit    key.Binding
	Submit  key.Binding
	Cancel  key.Binding
	Select  key.Binding
	NextFld key.Binding
	Up      key.Binding
	Down    key.Binding
	Yes     key.Binding
	No      key.Binding
}

var keys = keyMap{
	Tab:     key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "Fokus")),
	Open:    key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "öffnen")),
	Back:    key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "zurück")),
	New:     key.NewBinding(key.WithKeys("n"), key.WithHelp("n", "neues Issue")),
	Comment: key.NewBinding(key.WithKeys("c"), key.WithHelp("c", "kommentieren")),
	Labels:  key.NewBinding(key.WithKeys("l"), key.WithHelp("l", "Labels")),
	Toggle:  key.NewBinding(key.WithKeys("x"), key.WithHelp("x", "schließen/öffnen")),
	State:   key.NewBinding(key.WithKeys("s"), key.WithHelp("s", "Status-Filter")),
	Reload:  key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "neu laden")),
	Browser: key.NewBinding(key.WithKeys("o"), key.WithHelp("o", "im Browser")),
	Filter:  key.NewBinding(key.WithKeys("/"), key.WithHelp("/", "filtern")),
	Help:    key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "Hilfe")),
	Quit:    key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "beenden")),
	Submit:  key.NewBinding(key.WithKeys("ctrl+s"), key.WithHelp("ctrl+s", "senden")),
	Cancel:  key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "abbrechen")),
	Select:  key.NewBinding(key.WithKeys(" "), key.WithHelp("space", "auswählen")),
	NextFld: key.NewBinding(key.WithKeys("tab", "shift+tab"), key.WithHelp("tab", "Feld wechseln")),
	Up:      key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("↑/k", "hoch")),
	Down:    key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("↓/j", "runter")),
	Yes:     key.NewBinding(key.WithKeys("y", "j"), key.WithHelp("y", "ja")),
	No:      key.NewBinding(key.WithKeys("n", "esc"), key.WithHelp("n", "nein")),
}

// helpKeys implements help.KeyMap for the current view.
type helpKeys struct {
	short []key.Binding
	full  [][]key.Binding
}

func (h helpKeys) ShortHelp() []key.Binding  { return h.short }
func (h helpKeys) FullHelp() [][]key.Binding { return h.full }
