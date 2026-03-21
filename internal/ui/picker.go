package ui

import (
	"errors"
	"fmt"

	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
)

// ErrPickerCancelled is returned when the user cancels the picker.
var ErrPickerCancelled = errors.New("picker cancelled")

// item implements list.DefaultItem for the session picker.
type item struct {
	name     string
	status   string
	cwd      string
	accessed string
}

func (i item) FilterValue() string { return i.name }
func (i item) Title() string       { return StatusIcon(i.status) + " " + i.name }
func (i item) Description() string { return i.cwd + "  " + i.accessed }

// pickerModel is the bubbletea model for the session picker.
type pickerModel struct {
	list     list.Model
	selected string
	quitting bool
}

func (m pickerModel) Init() tea.Cmd {
	return nil
}

func (m pickerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			if sel, ok := m.list.SelectedItem().(item); ok {
				m.selected = sel.name
			}
			m.quitting = true
			return m, tea.Quit
		case "esc", "ctrl+c":
			m.quitting = true
			return m, tea.Quit
		}
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m pickerModel) View() tea.View {
	if m.quitting {
		return tea.NewView("")
	}
	return tea.NewView(m.list.View())
}

// PickSession shows an interactive filterable list of sessions and returns
// the selected session name. Returns ErrPickerCancelled if the user cancels.
func PickSession(sessions []SessionRow) (string, error) {
	items := make([]list.Item, len(sessions))
	for i, s := range sessions {
		items[i] = item{
			name:     s.Name,
			status:   s.Status,
			cwd:      s.Cwd,
			accessed: s.Accessed,
		}
	}

	const listHeight = 20

	delegate := list.NewDefaultDelegate()
	l := list.New(items, delegate, 60, listHeight)
	l.Title = "Select a session"
	l.SetShowStatusBar(true)
	l.SetFilteringEnabled(true)

	m := pickerModel{list: l}

	p := tea.NewProgram(m)
	result, err := p.Run()
	if err != nil {
		return "", fmt.Errorf("picker failed: %w", err)
	}

	final := result.(pickerModel)
	if final.selected == "" {
		return "", ErrPickerCancelled
	}
	return final.selected, nil
}
