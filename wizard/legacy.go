package wizard

import (
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

	selectedFile string

	devices list.Model
	device  string
}

type item struct {
	title, desc string
}

func (i item) Title() string       { return i.title }
func (i item) Description() string { return i.desc }
func (i item) FilterValue() string { return i.title } // FIXME: filter better

func initialModelOld(path string) model {
	m := model{
		selectedFile: path,
		devices:      list.New([]list.Item{}, list.NewDefaultDelegate(), 0, 0), // FIXME
		spinner:      spinner.New(spinner.WithSpinner(spinner.Dot)),
		help:         help.New(),
	}
	m.devices.Title = "glassUSB Media Creation Wizard - Select target USB drive"
	// FIXME: Load devices
	return m
}

func (m model) Init() tea.Cmd {
	return m.spinner.Tick
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
		m.devices.SetSize(msg.Width-h, msg.Height-v)
	}

	var spinnerCmd, devicesCmd tea.Cmd
	m.spinner, spinnerCmd = m.spinner.Update(msg)
	m.devices, devicesCmd = m.devices.Update(msg)
	return m, tea.Batch(spinnerCmd, devicesCmd)
}

func (m model) View() tea.View {
	var view string

	fullscreenDocStyle := docStyle.Height(m.height - 1).Width(m.width - 3)

	/*view = fullscreenDialogStyle.Render(
	dialogTitleStyle.Render("glassUSB Media Creation Wizard")+"\n\n",
	m.spinner.View()+" Starting wizard...",
	)*/
	view = fullscreenDocStyle.Render(m.devices.View())

	v := tea.NewView(view)
	v.AltScreen = true
	return v
}
