package main

import (
	"os"
	"strings"
	"time"

	"charm.land/bubbles/v2/filepicker"
	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/list"
	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

// TODO: Adaptive colors

type model struct {
	height int
	width  int

	spinner spinner.Model
	help    help.Model

	error   string
	warning string
	info    string

	filepicker    filepicker.Model
	selectedFile  string
	filepickerErr string

	devices list.Model
	device  string
}

type clearFilePickerErrorMsg struct{}

func clearFilePickerErrorAfter(t time.Duration) tea.Cmd {
	return tea.Tick(t, func(_ time.Time) tea.Msg {
		return clearFilePickerErrorMsg{}
	})
}

type item struct {
	title, desc string
}

func (i item) Title() string       { return i.title }
func (i item) Description() string { return i.desc }
func (i item) FilterValue() string { return i.title } // FIXME: filter better

type dialogContinueQuitKeyMap struct {
	Continue key.Binding
	Quit     key.Binding
}

func (k dialogContinueQuitKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Continue, k.Quit}
}

func (k dialogContinueQuitKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{{k.Continue, k.Quit}}
}

var dialogContinueQuitKeys = dialogContinueQuitKeyMap{
	Continue: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "continue"),
	),
	Quit: key.NewBinding(
		key.WithKeys("q", "esc", "ctrl+c"),
		key.WithHelp("q", "quit"),
	),
}

type dialogQuitKeyMap struct{ Quit key.Binding }

func (k dialogQuitKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Quit}
}

func (k dialogQuitKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{{k.Quit}}
}

var dialogQuitKeys = dialogContinueQuitKeyMap{
	Quit: key.NewBinding(key.WithKeys("q", "esc", "ctrl+c"), key.WithHelp("q", "quit")),
}

var docStyle = lipgloss.NewStyle().
	Border(lipgloss.DoubleBorder()).
	BorderForeground(lipgloss.BrightBlue).
	Margin(1, 1)

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

func initialModel() model {
	m := model{
		info: `This wizard will guide you through the process of creating a Windows installation USB drive.

Make sure you have a spare USB flash drive connected to your computer (>8 GB recommended for Windows 11), and a Windows installation ISO downloaded.

Press 'Continue' to select the Windows ISO you downloaded. Supported versions of Windows include Vista, 7 and newer.`,
		filepicker: filepicker.New(),
		devices:    list.New([]list.Item{}, list.NewDefaultDelegate(), 0, 0), // FIXME
		spinner:    spinner.New(spinner.WithSpinner(spinner.Dot)),
		help:       help.New(),
	}
	m.filepicker.AllowedTypes = []string{".iso", ".img"} // FIXME: Allow showing all files
	m.filepicker.CurrentDirectory, _ = os.Getwd()
	// FIXME: filepicker doesn't show on init the contents of current directory.
	m.devices.Title = "glassUSB Media Creation Wizard - Select target USB drive"
	// FIXME: Load devices
	return m
}

func (m model) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, m.filepicker.Init())
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		if m.error != "" || m.warning != "" || m.info != "" {
			switch {
			case key.Matches(msg, dialogContinueQuitKeys.Continue):
				if m.error != "" {
					m.error = ""
				} else if m.warning != "" {
					m.warning = ""
				} else if m.info != "" {
					m.info = ""
				}
			case key.Matches(msg, dialogContinueQuitKeys.Quit):
				return m, tea.Quit
			}
		} else {
			switch msg.String() {
			case "ctrl+c":
				return m, tea.Quit
			}
		}
	case tea.WindowSizeMsg:
		m.height = msg.Height
		m.width = msg.Width

		m.help.SetWidth(msg.Width)
		h, v := docStyle.GetFrameSize()
		m.filepicker.SetHeight(msg.Height - h - 5) // TODO: magic number for filepicker height
		m.devices.SetSize(msg.Width-h, msg.Height-v)
	case clearFilePickerErrorMsg:
		m.filepickerErr = ""
	}

	var cmd tea.Cmd
	m.spinner, cmd = m.spinner.Update(msg)
	if m.error == "" && m.warning == "" && m.info == "" {
		if m.selectedFile != "" {
			m.devices, cmd = m.devices.Update(msg)
		} else {
			m.filepicker, cmd = m.filepicker.Update(msg)
			if didSelect, path := m.filepicker.DidSelectFile(msg); didSelect {
				m.selectedFile = path
			}
			if didSelect, path := m.filepicker.DidSelectDisabledFile(msg); didSelect {
				m.filepickerErr = path + " is not valid."
				m.selectedFile = ""
				return m, tea.Batch(cmd, clearFilePickerErrorAfter(2*time.Second))
			}
		}
	}
	return m, cmd
}

func (m model) View() tea.View {
	var view string

	fullscreenDocStyle := docStyle.Height(m.height - 1).Width(m.width - 3)
	fullscreenDialogStyle := dialogStyle.Height(m.height - 1).Width(m.width - 3)

	if m.error != "" {
		view = fullscreenDialogStyle.BorderForeground(lipgloss.Red).Render(
			dialogTitleStyle.Render("glassUSB Media Creation Wizard")+"\n",
			dialogTitleWithColorStyle(lipgloss.Red).Render("Error!")+"\n\n",
			m.error+"\n\n",
			m.help.View(dialogQuitKeys),
		)
	} else if m.warning != "" {
		view = fullscreenDialogStyle.BorderForeground(lipgloss.Yellow).Render(
			dialogTitleStyle.Render("glassUSB Media Creation Wizard")+"\n",
			dialogTitleWithColorStyle(lipgloss.Yellow).Render("Warning!")+"\n\n",
			m.warning+"\n\n",
			m.help.View(dialogContinueQuitKeys),
		)
	} else if m.info != "" {
		view = fullscreenDialogStyle.Render(
			dialogTitleStyle.Render("glassUSB Media Creation Wizard")+"\n\n",
			m.info+"\n\n",
			m.help.View(dialogContinueQuitKeys),
		)
	} else if m.selectedFile == "" {
		// FIXME: Display list
		var s strings.Builder
		s.WriteString(dialogTitleStyle.Render("glassUSB Media Creation Wizard - Select Windows ISO") + "\n")
		s.WriteString("\n  ")
		if m.filepickerErr != "" {
			s.WriteString(m.filepicker.Styles.DisabledFile.Render(m.filepickerErr))
		} else if m.selectedFile == "" {
			s.WriteString("Pick a file:")
		}
		s.WriteString("\n\n" + m.filepicker.View() + "\n")
		//FIXME: s.WriteString(m.help.View())
		view = fullscreenDocStyle.Render(s.String()) // FIXME
	} else {
		/*view = fullscreenDialogStyle.Render(
		dialogTitleStyle.Render("glassUSB Media Creation Wizard")+"\n\n",
		m.spinner.View()+" Starting wizard...",
		)*/
		view = fullscreenDocStyle.Render(m.devices.View())
	}
	v := tea.NewView(view)
	v.AltScreen = true
	return v
}
