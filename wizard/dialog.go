package wizard

import (
	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

// ==================== Model ====================

type DialogModel struct {
	height int
	width  int
	keyMap dialogKeys

	help help.Model

	title      string
	message    string
	dialogType DialogType
	nextModel  tea.Model
}

type DialogType int

const (
	DialogTypeInfo DialogType = iota
	DialogTypeWarning
	DialogTypeError
)

type dialogKeys struct{ Continue, Quit key.Binding }

func NewDialogModel(title, message string, dialogType DialogType, nextModel tea.Model) DialogModel {
	m := DialogModel{
		height:     0,
		width:      0,
		keyMap:     initialDialogKeyMap,
		help:       help.New(),
		message:    message,
		dialogType: dialogType,
		nextModel:  nextModel,
	}
	if nextModel == nil {
		m.keyMap.Continue.SetEnabled(false)
	}
	return m
}

var initialDialogKeyMap = dialogKeys{
	Continue: key.NewBinding(
		key.WithHelp("enter", "continue"),
		key.WithKeys("enter"),
	),
	Quit: key.NewBinding(
		key.WithHelp("q", "quit"),
		key.WithKeys("q", "esc", "ctrl+c"),
	),
}

func (k dialogKeys) ShortHelp() []key.Binding { return []key.Binding{k.Continue, k.Quit} }

func (k dialogKeys) FullHelp() [][]key.Binding { return [][]key.Binding{k.ShortHelp()} }

func (m DialogModel) Init() tea.Cmd {
	return nil
}

// ==================== View ====================

var dialogStyle = docStyle.
	Align(lipgloss.Center, lipgloss.Center).
	Padding(0, 4)

var dialogTitleStyle = lipgloss.NewStyle().
	Background(lipgloss.BrightBlue).
	Foreground(lipgloss.Complementary(lipgloss.BrightBlue)).
	Padding(0, 1)

func dialogTitleWithColorStyle(color ansi.Color) lipgloss.Style {
	return dialogTitleStyle.Background(color).Foreground(lipgloss.Complementary(color))
}

func (m DialogModel) View() tea.View {
	fullscreenDialogStyle := dialogStyle.Height(m.height).Width(m.width)

	var view string
	switch m.dialogType {
	case DialogTypeError:
		view = fullscreenDialogStyle.BorderForeground(lipgloss.Red).Render(
			dialogTitleStyle.Render("glassUSB Media Creation Wizard")+"\n",
			dialogTitleWithColorStyle(lipgloss.Red).Render("Error!")+"\n\n",
			m.message+"\n\n",
			m.help.View(m.keyMap),
		)
	case DialogTypeWarning:
		view = fullscreenDialogStyle.BorderForeground(lipgloss.Yellow).Render(
			dialogTitleStyle.Render("glassUSB Media Creation Wizard")+"\n",
			dialogTitleWithColorStyle(lipgloss.Yellow).Render("Warning!")+"\n\n",
			m.message+"\n\n",
			m.help.View(m.keyMap),
		)
	case DialogTypeInfo:
		view = fullscreenDialogStyle.Render(
			dialogTitleStyle.Render("glassUSB Media Creation Wizard")+"\n\n",
			m.message+"\n\n",
			m.help.View(m.keyMap),
		)
	}

	v := tea.NewView(view)
	v.AltScreen = true
	return v
}

// ==================== Controller ====================

func (m DialogModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, initialDialogKeyMap.Continue):
			if m.nextModel == nil {
				return m, tea.Quit
			} else {
				return switchToModel(m.nextModel, m.height, m.width)
			}
		case key.Matches(msg, initialDialogKeyMap.Quit):
			return m, tea.Quit
		case msg.String() == "ctrl+c":
			return m, tea.Quit
		}
	case tea.WindowSizeMsg:
		m.height = msg.Height
		m.width = msg.Width
		m.help.SetWidth(msg.Width - dialogStyle.GetHorizontalFrameSize())
	}

	return m, nil
}
