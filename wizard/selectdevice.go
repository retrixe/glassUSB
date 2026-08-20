package wizard

import (
	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
)

// ==================== Model ====================

type SelectDeviceModel struct {
	height  int
	width   int
	isoPath string

	devices list.Model

	device string
}

type item struct {
	title, desc string
}

func (i item) Title() string       { return i.title }
func (i item) Description() string { return i.desc }
func (i item) FilterValue() string { return i.title } // FIXME: filter better

func NewSelectDeviceModel(isoPath string) SelectDeviceModel {
	m := SelectDeviceModel{
		isoPath: isoPath,
		devices: list.New([]list.Item{}, list.NewDefaultDelegate(), 0, 0), // FIXME
		//spinner: spinner.New(spinner.WithSpinner(spinner.Dot)),
	}
	m.devices.Title = "glassUSB Media Creation Wizard - Select target USB drive"
	// FIXME: Load devices
	return m
}

func (m SelectDeviceModel) Init() tea.Cmd {
	return nil
}

// ==================== View ====================

func (m SelectDeviceModel) View() tea.View {
	fullscreenDocStyle := docStyle.Height(m.height).Width(m.width)

	var view string
	view = fullscreenDocStyle.Render(m.devices.View())

	v := tea.NewView(view)
	v.AltScreen = true
	return v
}

// ==================== Controller ====================

func (m SelectDeviceModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		}
	case tea.WindowSizeMsg:
		m.height = msg.Height
		m.width = msg.Width

		x, y := docStyle.GetFrameSize()
		m.devices.SetSize(msg.Width-x, msg.Height-y)
	}

	var devicesCmd tea.Cmd
	m.devices, devicesCmd = m.devices.Update(msg)

	return m, devicesCmd
}
