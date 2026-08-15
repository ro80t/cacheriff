package ui

import (
	"context"
	"time"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"

	"cacheriff/internal/driver"
)

// focusedPanel identifies which panel currently receives navigation
// keys, mirroring lazygit's model of one focused panel at a time.
type focusedPanel int

const (
	focusSidebar focusedPanel = iota
	focusMain
)

// loadState tracks the lifecycle of fetching a driver's data.
type loadState int

const (
	loadIdle loadState = iota
	loadInProgress
	loadDone
	loadError
)

// loadTimeout bounds how long a driver's cache/package scan may run;
// some caches (e.g. npm's) contain very many small files.
const loadTimeout = 2 * time.Minute

// driverItem pairs a driver with whether it was detected as installed
// when the app started.
type driverItem struct {
	driver    driver.Driver
	available bool
}

// Model is the root Bubble Tea model for the application.
type Model struct {
	keys keyMap
	help help.Model

	items  []driverItem
	cursor int

	focus     focusedPanel
	activeIdx int // index into items for the currently loaded driver, -1 if none

	state    loadState
	loadErr  error
	cache    []driver.Entry
	packages []driver.Entry

	loadGen    int
	loadCancel context.CancelFunc

	spinner  spinner.Model
	viewport viewport.Model

	width, height int
	ready         bool
}

// NewModel builds the initial application state, detecting which
// package managers are available on this machine.
func NewModel() Model {
	var items []driverItem
	for _, d := range driver.All() {
		if !driver.SupportsCurrentOS(d) {
			continue
		}
		items = append(items, driverItem{driver: d, available: d.Available()})
	}

	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = panelTitleStyle

	return Model{
		keys:      newKeyMap(),
		help:      help.New(),
		items:     items,
		activeIdx: -1,
		spinner:   sp,
		viewport:  viewport.New(0, 0),
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.help.Width = msg.Width
		m.ready = true
		m.applyLayout()
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)

	case spinner.TickMsg:
		if m.state == loadInProgress {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}
		return m, nil

	case driverDataMsg:
		if msg.gen != m.loadGen {
			return m, nil // stale result from a since-abandoned selection
		}
		m.cache = msg.cache
		m.packages = msg.packages
		if msg.err != nil {
			m.state = loadError
			m.loadErr = msg.err
		} else {
			m.state = loadDone
		}
		m.viewport.SetContent(m.renderMainContent())
		m.viewport.GotoTop()
		return m, nil
	}

	return m, nil
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Quit):
		if m.loadCancel != nil {
			m.loadCancel()
		}
		return m, tea.Quit

	case key.Matches(msg, m.keys.Help):
		m.help.ShowAll = !m.help.ShowAll
		m.applyLayout()
		return m, nil

	case key.Matches(msg, m.keys.Tab):
		if m.focus == focusSidebar {
			m.focus = focusMain
		} else {
			m.focus = focusSidebar
		}
		return m, nil

	case key.Matches(msg, m.keys.Back):
		m.focus = focusSidebar
		return m, nil
	}

	if m.focus == focusSidebar {
		return m.handleSidebarKey(msg)
	}
	return m.handleMainKey(msg)
}

func (m Model) handleSidebarKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Up):
		if m.cursor > 0 {
			m.cursor--
		}

	case key.Matches(msg, m.keys.Down):
		if m.cursor < len(m.items)-1 {
			m.cursor++
		}

	case key.Matches(msg, m.keys.Select):
		return m.selectDriver(m.cursor)
	}
	return m, nil
}

func (m Model) handleMainKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

// selectDriver starts (or restarts) loading data for the item at idx.
func (m Model) selectDriver(idx int) (tea.Model, tea.Cmd) {
	item := m.items[idx]
	m.activeIdx = idx
	m.focus = focusMain

	if !item.available {
		m.state = loadIdle
		m.cache, m.packages, m.loadErr = nil, nil, nil
		m.viewport.SetContent(m.renderMainContent())
		return m, nil
	}

	if m.loadCancel != nil {
		m.loadCancel()
	}
	ctx, cancel := context.WithTimeout(context.Background(), loadTimeout)
	m.loadCancel = cancel
	m.loadGen++

	m.state = loadInProgress
	m.loadErr = nil
	m.cache, m.packages = nil, nil
	m.viewport.SetContent(m.renderMainContent())

	return m, tea.Batch(m.spinner.Tick, loadDriverDataCmd(ctx, item.driver, m.loadGen))
}

// activeItem returns the item currently shown in the main panel, if any.
func (m Model) activeItem() (driverItem, bool) {
	if m.activeIdx < 0 || m.activeIdx >= len(m.items) {
		return driverItem{}, false
	}
	return m.items[m.activeIdx], true
}
