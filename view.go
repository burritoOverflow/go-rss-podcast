package main

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/mmcdole/gofeed"
)

// View renders the full screen layout: header, paginated episode list,
// details pane, status footer, and optional help overlay.
func (m model) View() string {
	if m.width == 0 || m.height == 0 {
		return "Loading..."
	}

	var b strings.Builder

	// Header
	title := truncate(m.feed.Title, m.width-30)
	b.WriteString(headerStyle.Render(fmt.Sprintf(" %s ", title)))
	b.WriteString("\n")

	// Subheader
	stats := fmt.Sprintf("%d/%d episodes • Page %d/%d • %d selected • %d downloading",
		len(m.filtered), len(m.items), m.page+1, m.totalPages, len(m.selected), m.downloads)
	b.WriteString(lipgloss.NewStyle().Foreground(colorMuted).Render(stats))
	b.WriteString("\n")

	// Search bar: shown while typing a query or while a filter is active.
	if m.searching {
		b.WriteString(lipgloss.NewStyle().Foreground(colorPrimary).Bold(true).Render(fmt.Sprintf("/%s█", m.query)))
		b.WriteString("\n")
	} else if m.query != "" {
		b.WriteString(lipgloss.NewStyle().Foreground(colorPrimary).Render(fmt.Sprintf("/%s (press / to edit, esc to clear)", m.query)))
		b.WriteString("\n")
	}
	b.WriteString("\n")

	// Main content: list + details
	// Give the episode list most of the width; details only needs ~1/3.
	detailsWidth := m.width / 3
	if detailsWidth < 36 {
		detailsWidth = 36
	}
	listWidth := m.width - detailsWidth - 4
	if listWidth < 50 {
		listWidth = 50
		detailsWidth = m.width - listWidth - 4
	}
	if detailsWidth < 30 {
		detailsWidth = 30
		listWidth = m.width - detailsWidth - 4
	}
	if listWidth < 30 {
		listWidth = m.width - 4
		detailsWidth = 0
	}

	listView := m.renderList(listWidth)
	detailsView := m.renderDetails(detailsWidth)

	row := lipgloss.JoinHorizontal(lipgloss.Top, listView, "  ", detailsView)
	b.WriteString(row)
	b.WriteString("\n\n")

	// Footer / status
	status := m.message
	if status == "" {
		status = "Press ? for help"
	}
	if m.downloads > 0 {
		status = fmt.Sprintf("%s %s", m.spinner.View(), status)
	}
	b.WriteString(footerStyle.Render(status))

	if m.help {
		help := helpBoxStyle.Render(
			"j/k or ↑/↓  move cursor\n" +
				"h/l or ←/→  change page\n" +
				"g/G         first/last page\n" +
				"space       select episode\n" +
				"a           select all on page\n" +
				"d           download selected / current\n" +
				"D           download all (filtered) episodes\n" +
				"/           fuzzy search by title\n" +
				"esc         clear search filter\n" +
				"?           toggle help\n" +
				"q           quit",
		)
		b.WriteString("\n\n")
		b.WriteString(help)
	}

	return b.String()
}

func (m model) renderList(width int) string {
	start, end := m.pageBounds()

	// Fixed-ish column widths.
	numW := len(strconv.Itoa(len(m.items)))
	if numW < 3 {
		numW = 3
	}

	// Date and Duration have predictable content ("2006-01-02", "H:MM:SS"),
	// so give them compact fixed widths and let Title absorb the rest of the
	// available space. Overhead: marker(1)+space(1)+numW+3×sep(2) = numW+8.
	dateW := 15
	durW := 8
	titleW := width - numW - dateW - durW - 8
	if titleW < 10 {
		titleW = 10
	}

	var rows []string
	// Column header (two leading spaces align with the selection marker column).
	header := lipgloss.NewStyle().
		Foreground(colorSecondary).
		Bold(true).
		Render(strings.Repeat(" ", 2) +
			padRight("#", numW) + "  " +
			padRight("Title", titleW) + "  " +
			padRight("Date", dateW) + "  " +
			padRight("Duration", durW))
	rows = append(rows, header)
	rows = append(rows, strings.Repeat("─", width))

	if len(m.filtered) == 0 {
		rows = append(rows, lipgloss.NewStyle().Foreground(colorMuted).Render("No matching episodes"))
		return lipgloss.NewStyle().Width(width).Render(strings.Join(rows, "\n"))
	}

	for pos := start; pos < end; pos++ {
		i := m.filtered[pos]
		item := m.items[i]
		local := pos - start
		numStr := strconv.Itoa(i + 1)

		duration := formatDuration(item.ITunesExt.Duration)

		date := "-"
		if item.PublishedParsed != nil {
			date = item.PublishedParsed.Format("2006-01-02")
		}

		marker := " "
		if m.selected[i] {
			marker = "✓"
		}

		title := truncate(item.Title, titleW)
		line := marker + " " +
			padRight(numStr, numW) + "  " +
			padRight(title, titleW) + "  " +
			padRight(date, dateW) + "  " +
			padRight(duration, durW)

		if local == m.cursor {
			line = selectedStyle.Width(width).Render(line)
		}
		rows = append(rows, line)
	}

	return lipgloss.NewStyle().Width(width).Render(strings.Join(rows, "\n"))
}

func (m model) renderDetails(width int) string {
	idx := m.globalIndex()
	if idx < 0 || idx >= len(m.items) {
		return detailsBoxStyle.Width(width).Render("No episode selected")
	}
	item := m.items[idx]

	duration := "-"
	if item.ITunesExt != nil && item.ITunesExt.Duration != "" {
		duration = item.ITunesExt.Duration
	}

	date := "-"
	if item.PublishedParsed != nil {
		date = item.PublishedParsed.Format("January 2, 2006")
	}

	desc := stripHTML(item.Description)
	desc = truncate(desc, 400)

	content := fmt.Sprintf(
		"%s\n\n%s\n\n%s\n%s\n%s",
		sectionStyle.Render(truncate(item.Title, width-4)),
		lipgloss.NewStyle().Foreground(colorMuted).Render(fmt.Sprintf("Published: %s • Duration: %s", date, duration)),
		sectionStyle.Render("Description"),
		desc,
		m.enclosureInfo(item),
	)

	return detailsBoxStyle.Width(width).Render(content)
}

func (m model) enclosureInfo(item *gofeed.Item) string {
	if len(item.Enclosures) == 0 {
		return lipgloss.NewStyle().Foreground(colorDanger).Render("No audio enclosure")
	}
	encl := item.Enclosures[0]
	size := "-"
	if encl.Length != "" {
		if n, err := strconv.ParseInt(encl.Length, 10, 64); err == nil {
			size = fmt.Sprintf("%.1f MB", float64(n)/1024/1024)
		}
	}
	return lipgloss.NewStyle().Foreground(colorSecondary).Render(fmt.Sprintf("Audio: %s (%s)", encl.Type, size))
}
