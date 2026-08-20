package wizard

import (
	"os"
	"strings"
	"time"

	"charm.land/bubbles/v2/filepicker"
	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/list"
	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
)

// TODO: Adaptive colors

type model struct {
	height int
	width  int

	spinner spinner.Model
	help    help.Model

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

func initialModelOld() model {
	m := model{
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
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
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
	return m, cmd
}

func (m model) View() tea.View {
	var view string

	fullscreenDocStyle := docStyle.Height(m.height - 1).Width(m.width - 3)

	if m.selectedFile == "" {
		// FIXME: Display list
		var s strings.Builder
		s.WriteString(dialogTitleStyle.Render("glassUSB Media Creation Wizard - Select Windows ISO"))
		s.WriteString("\n\n  ")
		if m.filepickerErr != "" {
			s.WriteString(m.filepicker.Styles.DisabledFile.Render(m.filepickerErr))
		} else if m.selectedFile == "" {
			s.WriteString("Pick a file:")
		}
		s.WriteString("\n\n")
		s.WriteString(m.filepicker.View())
		s.WriteString("\n")
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
