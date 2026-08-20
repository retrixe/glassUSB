package wizard

import (
	"os"
	"strings"
	"time"

	"charm.land/bubbles/v2/filepicker"
	"charm.land/bubbles/v2/help"
	tea "charm.land/bubbletea/v2"
)

// FIXME: Open system file picker if available

// ==================== Model ====================

type SelectIsoModel struct {
	height     int
	width      int
	help       help.Model
	filepicker filepicker.Model

	error string
}

func NewSelectIsoModel() tea.Model {
	m := SelectIsoModel{
		filepicker: filepicker.New(),
		help:       help.New(),
	}
	m.filepicker.AutoHeight = false
	m.filepicker.AllowedTypes = []string{".iso", ".img"} // FIXME: Allow showing all files
	m.filepicker.CurrentDirectory, _ = os.Getwd()
	return m
}

func (m SelectIsoModel) Init() tea.Cmd {
	return m.filepicker.Init()
}

// ==================== View ====================

func (m SelectIsoModel) View() tea.View {
	fullscreenDocStyle := docStyle.Height(m.height).Width(m.width)

	var view strings.Builder
	view.WriteString(dialogTitleStyle.Render("glassUSB Media Creation Wizard - Select Windows ISO"))
	view.WriteString("\n\n  ")
	if m.error != "" {
		view.WriteString(m.filepicker.Styles.DisabledFile.Render(m.error))
	} else {
		view.WriteString("Pick a file:")
	}
	view.WriteString("\n\n")
	view.WriteString(m.filepicker.View())
	view.WriteString("\n")
	// FIXME: s.WriteString(m.help.View(m.filepicker.KeyMap))

	v := tea.NewView(fullscreenDocStyle.Render(view.String()))
	v.AltScreen = true
	return v
}

// ==================== Controller ====================

func (m SelectIsoModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		}
	case tea.WindowSizeMsg:
		m.height = msg.Height
		m.width = msg.Width

		m.help.SetWidth(msg.Width - docStyle.GetHorizontalFrameSize())
		const verticalMargin = 6 // magic number for filepicker height
		m.filepicker.SetHeight(msg.Height - docStyle.GetVerticalFrameSize() - verticalMargin)
	case clearErrorMsg:
		m.error = ""
	}

	// Update the filepicker
	var cmd tea.Cmd
	m.filepicker, cmd = m.filepicker.Update(msg)

	if didSelect, path := m.filepicker.DidSelectFile(msg); didSelect {
		return switchToModel(initialModelOld(path), m.height, m.width)
	} else if didSelect, path := m.filepicker.DidSelectDisabledFile(msg); didSelect {
		m.error = path + " is not valid."
		return m, tea.Batch(cmd, clearErrorAfter(2*time.Second))
	}

	return m, cmd
}
