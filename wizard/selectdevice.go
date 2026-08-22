package wizard

import (
	"time"

	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
	"github.com/retrixe/imprint/imaging"
)

// FIXME support reloading

// ==================== Model ====================

type SelectDeviceModel struct {
	height  int
	width   int
	isoPath string

	deviceList list.Model

	device  string
	devices []imaging.Device
	err     error
}

type item struct {
	title, desc string
}

func (i item) Title() string       { return i.title }
func (i item) Description() string { return i.desc }
func (i item) FilterValue() string { return i.title + " " + i.desc }

type DevicesLoadedMsg struct {
	devices []imaging.Device
	err     error
}

func NewSelectDeviceModel(isoPath string) *SelectDeviceModel {
	m := &SelectDeviceModel{
		isoPath:    isoPath,
		deviceList: list.New([]list.Item{}, list.NewDefaultDelegate(), 0, 0), // FIXME
		//spinner:    spinner.New(spinner.WithSpinner(spinner.Dot)),
	}
	// FIXME: m.deviceList.SetSpinner(spinner.Pulse)
	m.deviceList.Title = "glassUSB Media Creation Wizard - Select target USB drive"
	return m
}

func (m *SelectDeviceModel) Init() tea.Cmd {
	return tea.Batch(loadDevices, m.deviceList.StartSpinner())
}

// ==================== View ====================

func (m *SelectDeviceModel) View() tea.View {
	fullscreenDocStyle := docStyle.Height(m.height).Width(m.width)

	var view string
	// FIXME: Error state
	view = fullscreenDocStyle.Render(m.deviceList.View())

	v := tea.NewView(view)
	v.AltScreen = true
	return v
}

// ==================== Controller ====================

func (m *SelectDeviceModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
		m.deviceList.SetSize(msg.Width-x, msg.Height-y)
	case DevicesLoadedMsg:
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.devices = msg.devices

		items := make([]list.Item, len(m.devices))
		for i, d := range m.devices {
			if d.Model == "" {
				items[i] = item{title: d.Name, desc: d.Size}
			} else {
				items[i] = item{title: d.Name, desc: d.Model + ", " + d.Size}
			}
		}
		m.deviceList.StopSpinner()
		return m, m.deviceList.SetItems(items)
	}

	var deviceListCmd tea.Cmd
	m.deviceList, deviceListCmd = m.deviceList.Update(msg)

	return m, deviceListCmd
}

func loadDevices() tea.Msg {
	time.Sleep(2 * time.Second)
	devices, err := imaging.GetDevices(imaging.SystemPlatform)
	if err != nil {
		return DevicesLoadedMsg{err: err}
	}
	return DevicesLoadedMsg{devices: devices}
}
