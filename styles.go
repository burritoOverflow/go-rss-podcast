package main

import "github.com/charmbracelet/lipgloss"

// styles centralize the color palette and reusable Lipgloss styles so the
// header, list, details pane, and help overlay stay visually consistent.
var (
	colorPrimary   = lipgloss.Color("#7B61FF")
	colorSecondary = lipgloss.Color("#04B575")
	colorMuted     = lipgloss.Color("#888888")
	colorDanger    = lipgloss.Color("#FF5F5F")
	colorBg        = lipgloss.Color("#1A1A1A")
	colorSurface   = lipgloss.Color("#252525")
	colorText      = lipgloss.Color("#FFFFFF")
	colorAccent    = lipgloss.Color("#FF00FF")

	headerStyle = lipgloss.NewStyle().
			Foreground(colorText).
			Background(colorPrimary).
			Bold(true).
			Padding(0, 1)

	footerStyle = lipgloss.NewStyle().
			Foreground(colorMuted).
			Padding(0, 1)

	statsStyle = lipgloss.NewStyle().
			Foreground(colorMuted)

	searchActiveStyle = lipgloss.NewStyle().
				Foreground(colorPrimary).
				Bold(true)

	searchFilterStyle = lipgloss.NewStyle().
				Foreground(colorPrimary)

	sectionStyle = lipgloss.NewStyle().
			Foreground(colorSecondary).
			Bold(true)

	listHeaderStyle = lipgloss.NewStyle().
			Foreground(colorSecondary).
			Bold(true)

	selectedStyle = lipgloss.NewStyle().
			Foreground(colorText).
			Background(colorSurface).
			Bold(true)

	mutedTextStyle = lipgloss.NewStyle().
			Foreground(colorMuted)

	dangerTextStyle = lipgloss.NewStyle().
			Foreground(colorDanger)

	audioInfoStyle = lipgloss.NewStyle().
			Foreground(colorSecondary)

	fillWidthStyle = lipgloss.NewStyle()

	spinnerStyle = lipgloss.NewStyle().
			Foreground(colorAccent)

	detailsBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorPrimary).
			Padding(1).
			Background(colorBg)

	helpBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorSecondary).
			Padding(1, 2).
			Background(colorBg)
)
