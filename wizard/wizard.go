package wizard

import (
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// TODO: Adaptive colors

func InitialModel() tea.Model {
	return NewDialogModel(
		"glassUSB Media Creation Wizard",
		`This wizard will guide you through the process of creating a Windows installation USB drive.

Make sure you have a spare USB flash drive connected to your computer (>8 GB recommended for Windows 11), and a Windows installation ISO downloaded.

Press 'Continue' to select the Windows ISO you downloaded. Supported versions of Windows include Vista, 7 and newer.`,
		DialogTypeInfo,
		// FIXME: Replace with proper models that handle the wizard steps.
		NewSelectIsoModel(),
	)
}

// Shared controller utilities

func switchToModel(m tea.Model, height, width int) (tea.Model, tea.Cmd) {
	initCmd := m.Init()
	sizeCmd := tea.WindowSizeMsg{Width: width, Height: height}
	return m, tea.Batch(initCmd, func() tea.Msg { return sizeCmd })
}

type clearErrorMsg struct{}

func clearErrorAfter(t time.Duration) tea.Cmd {
	return tea.Tick(t, func(time.Time) tea.Msg { return clearErrorMsg{} })
}

// Shared view styles

var docStyle = lipgloss.NewStyle().
	Border(lipgloss.DoubleBorder()).
	BorderForeground(lipgloss.BrightBlue)
