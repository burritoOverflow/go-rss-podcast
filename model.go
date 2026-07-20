package main

import (
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/mmcdole/gofeed"
)

// number of episodes shown per page in the list view
const pageSize = 10

// downloadMsg carries the result of an asynchronous episode download back to
// the Bubble Tea update loop so the UI can report status and decrement the
// in-flight download counter.
type downloadMsg struct {
	num      int
	filename string
	written  int64
	err      error
}

// model holds the complete TUI state: the parsed feed, the user's current
// position in the paginated list, which episodes are selected for batch
// download, terminal dimensions, and transient status messages.
type model struct {
	feed       *gofeed.Feed
	items      []*gofeed.Item
	outDir     string
	page       int
	totalPages int
	cursor     int
	// maps the global index of an episode to whether it is selected for
	// batch download.
	selected map[int]bool
	// maps the global index of an episode to whether
	// it is currently being downloaded.
	downloading map[int]bool
	width       int
	height      int
	message     string
	msgAt       time.Time
	spinner     spinner.Model
	downloads   int
	help        bool
	query       string
	searching   bool
	filtered    []int // indices into items matching query, in match order
}

func newModel(feed *gofeed.Feed, outDir string) model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = spinnerStyle

	filtered := allIndices(len(feed.Items))
	totalPages := (len(filtered) + pageSize - 1) / pageSize
	if totalPages == 0 {
		totalPages = 1
	}

	return model{
		feed:        feed,
		items:       feed.Items,
		outDir:      outDir,
		page:        0,
		totalPages:  totalPages,
		cursor:      0,
		selected:    make(map[int]bool),
		downloading: make(map[int]bool),
		spinner:     s,
		filtered:    filtered,
	}
}

func (m model) Init() tea.Cmd {
	return m.spinner.Tick
}

// Update handles keyboard input, window resizes, spinner ticks, and download
// completion messages. It is the single source of truth for all state changes.
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case tea.KeyMsg:
		if m.searching {
			return m.updateSearchInput(msg)
		}

		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit

		case "?":
			m.help = !m.help

		case "/":
			m.searching = true

		case "esc":
			if m.query != "" {
				m.query = ""
				m.refreshFilter()
			}

		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}

		case "down", "j":
			if m.cursor < m.pageCount()-1 {
				m.cursor++
			}

		case "left", "h", "p":
			if m.page > 0 {
				m.page--
				m.cursor = 0
			}

		case "right", "l", "n":
			if m.page < m.totalPages-1 {
				m.page++
				m.cursor = 0
			}

		case "home", "g":
			m.page = 0
			m.cursor = 0

		case "end", "G":
			m.page = m.totalPages - 1
			m.cursor = m.pageCount() - 1

		case " ":
			idx := m.globalIndex()
			if idx >= 0 && !m.downloading[idx] {
				if m.selected[idx] {
					delete(m.selected, idx)
				} else {
					m.selected[idx] = true
				}
			}

		case "a":
			start, end := m.pageBounds()
			allSelected := true
			for pos := start; pos < end; pos++ {
				idx := m.filtered[pos]
				if m.downloading[idx] {
					continue
				}
				if !m.selected[idx] {
					allSelected = false
					break
				}
			}
			for pos := start; pos < end; pos++ {
				idx := m.filtered[pos]
				if m.downloading[idx] {
					continue
				}
				if allSelected {
					delete(m.selected, idx)
				} else {
					m.selected[idx] = true
				}
			}

		case "d":
			return m.downloadSelection()

		case "D":
			return m.downloadAll()
		}

	case downloadMsg:
		m.downloads--
		delete(m.downloading, msg.num-1)
		if msg.err != nil {
			m.setMessage(fmt.Sprintf("Failed %s: %v", msg.filename, msg.err))
		} else {
			m.setMessage(fmt.Sprintf("Downloaded %s (%.1f MB)", msg.filename, float64(msg.written)/1024/1024))
		}

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m *model) setMessage(s string) {
	m.message = s
	m.msgAt = time.Now()
}

// updateSearchInput handles keystrokes while the search box has focus: text
// is appended to the query and the filtered list is recomputed live.
func (m model) updateSearchInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyCtrlC:
		return m, tea.Quit

	case tea.KeyEsc:
		m.searching = false
		m.query = ""
		m.refreshFilter()

	case tea.KeyEnter:
		m.searching = false

	case tea.KeyBackspace:
		if len(m.query) > 0 {
			runs := []rune(m.query)
			m.query = string(runs[:len(runs)-1])
			m.refreshFilter()
		}

	case tea.KeyRunes:
		m.query += string(msg.Runes)
		m.refreshFilter()
	}

	return m, nil
}

// refreshFilter recomputes the fuzzy-matched episode list for the current
// query and resets pagination to the first page of results.
func (m *model) refreshFilter() {
	m.filtered = fuzzyFilter(m.items, m.query)
	m.totalPages = (len(m.filtered) + pageSize - 1) / pageSize
	if m.totalPages == 0 {
		m.totalPages = 1
	}
	m.page = 0
	m.cursor = 0
}

func (m model) pageCount() int {
	start, end := m.pageBounds()
	return end - start
}

func (m model) pageBounds() (start, end int) {
	start = m.page * pageSize
	end = start + pageSize
	if end > len(m.filtered) {
		end = len(m.filtered)
	}
	return
}

// globalIndex maps the current page/cursor position to the underlying item
// index, accounting for any active search filter. Returns -1 if there is no
// episode at that position (e.g. an empty filtered result set).
func (m model) globalIndex() int {
	pos := m.page*pageSize + m.cursor
	if pos < 0 || pos >= len(m.filtered) {
		return -1
	}
	return m.filtered[pos]
}

func (m model) downloadSelection() (tea.Model, tea.Cmd) {
	if len(m.selected) == 0 {
		idx := m.globalIndex()
		if idx < 0 || m.downloading[idx] {
			return m, nil
		}
		m.downloads++
		m.downloading[idx] = true
		return m, downloadCmd(m.items[idx], idx+1, m.outDir)
	}

	cmds := make([]tea.Cmd, 0, len(m.selected))
	for idx := range m.selected {
		m.downloads++
		m.downloading[idx] = true
		cmds = append(cmds, downloadCmd(m.items[idx], idx+1, m.outDir))
	}
	m.selected = make(map[int]bool)
	return m, tea.Batch(cmds...)
}

func (m model) downloadAll() (tea.Model, tea.Cmd) {
	cmds := make([]tea.Cmd, 0, len(m.filtered))
	for _, idx := range m.filtered {
		if m.downloading[idx] {
			continue
		}
		m.downloads++
		m.downloading[idx] = true
		delete(m.selected, idx)
		cmds = append(cmds, downloadCmd(m.items[idx], idx+1, m.outDir))
	}
	return m, tea.Batch(cmds...)
}

func downloadCmd(item *gofeed.Item, num int, outDir string) tea.Cmd {
	return func() tea.Msg {
		filename, written, err := downloadEpisode(item, num, outDir)
		return downloadMsg{num: num, filename: filename, written: written, err: err}
	}
}
