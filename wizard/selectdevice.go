package wizard

import (
	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/list"
	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
	"github.com/retrixe/imprint/imaging"
)

// ==================== Model ====================

type SelectDeviceModel struct {
	height  int
	width   int
	isoPath string

	deviceList list.Model

	device  string
	devices []imaging.Device
}

type item struct {
	title, desc string
}

func (i item) Title() string       { return i.title }
func (i item) Description() string { return i.desc }
func (i item) FilterValue() string { return i.title + " " + i.desc }

type selectDeviceListDelegateKeys struct{ Select, Reload key.Binding }

var selectDeviceListDelegateKeyMap = selectDeviceListDelegateKeys{
	Select: key.NewBinding(
		key.WithHelp("enter", "select"),
		key.WithKeys("enter"),
	),
	Reload: key.NewBinding(
		key.WithHelp("r", "reload"),
		key.WithKeys("r"),
	),
}

func (k selectDeviceListDelegateKeys) ShortHelp() []key.Binding {
	return []key.Binding{k.Select, k.Reload}
}

func (k selectDeviceListDelegateKeys) FullHelp() [][]key.Binding {
	return [][]key.Binding{k.ShortHelp()}
}

type DevicesLoadedMsg struct {
	devices []imaging.Device
	err     error
}

func NewSelectDeviceModel(isoPath string) *SelectDeviceModel {
	delegate := newSelectDeviceListDelegate()
	m := &SelectDeviceModel{
		isoPath:    isoPath,
		deviceList: list.New([]list.Item{}, delegate, 0, 0),
	}
	m.deviceList.SetSpinner(spinner.Line) // Other options could be MiniDot, Pulse, Jump
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
	view = fullscreenDocStyle.Render(m.deviceList.View())

	v := tea.NewView(view)
	v.AltScreen = true
	return v
}

// ==================== Controller ====================

func (m *SelectDeviceModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch {
		case msg.String() == "ctrl+c":
			return m, tea.Quit
		case key.Matches(msg, selectDeviceListDelegateKeyMap.Reload):
			if m.devices != nil {
				m.devices = nil

				return m, tea.Batch(
					loadDevices,
					m.deviceList.StartSpinner(),
					m.deviceList.SetItems([]list.Item{}),
				)
			}
		case key.Matches(msg, selectDeviceListDelegateKeyMap.Select):
			// FIXME: Enter logic
		}
	case tea.WindowSizeMsg:
		m.height = msg.Height
		m.width = msg.Width

		x, y := docStyle.GetFrameSize()
		m.deviceList.SetSize(msg.Width-x, msg.Height-y)
	case DevicesLoadedMsg:
		if msg.err != nil {
			return switchToModel(NewDialogModel(
				"glassUSB Media Creation Wizard",
				"Error loading devices:\n"+msg.err.Error()+
					"\n\nPlease make sure you have a USB drive connected to your computer and try again.",
				DialogTypeError,
				NewSelectDeviceModel(m.isoPath),
			), m.height, m.width)
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

func newSelectDeviceListDelegate() list.DefaultDelegate {
	d := list.NewDefaultDelegate()
	d.FullHelpFunc = selectDeviceListDelegateKeyMap.FullHelp
	d.ShortHelpFunc = selectDeviceListDelegateKeyMap.ShortHelp
	return d
}

func loadDevices() tea.Msg {
	devices, err := imaging.GetDevices(imaging.SystemPlatform)
	if err != nil {
		return DevicesLoadedMsg{err: err}
	}
	return DevicesLoadedMsg{devices: devices}
}
