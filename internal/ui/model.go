package ui

import (
	"context"
	"os"
	"sort"
	"time"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/ro80t/cacheriff/internal/driver"
)

// focusedPanel identifies which panel currently receives navigation keys.
type focusedPanel int

const (
	focusSidebar focusedPanel = iota
	focusMain
)

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

// packageScope selects which package list (global or local) the main
// panel currently shows below the always-visible caches section.
type packageScope int

const (
	scopeGlobal packageScope = iota
	scopeLocal
)

type driverItem struct {
	driver    driver.Driver
	available bool
}

// Model is the root Bubble Tea model for the application.
type Model struct {
	keys keyMap
	help help.Model

	drivers []driverItem
	cursor  int
	root    string // project directory LocalPackages is evaluated against

	focus     focusedPanel
	activeIdx int // index into drivers for the currently loaded driver, -1 if none

	state          loadState
	loadErr        error
	cache          []driver.Entry
	globalPackages []driver.Entry
	localPackages  []driver.Entry
	scope          packageScope // which of globalPackages/localPackages the main panel shows
	packageCursor  int          // index into the currently visible package list (see currentEntries)

	loadGen    int
	loadCancel context.CancelFunc

	// confirmRemove/pendingRemove drive the uninstall confirmation
	// modal; removing/removeErr/removeGen track the uninstall command
	// once confirmed. Only global packages support uninstall today.
	confirmRemove bool
	pendingRemove driver.Entry
	removing      bool
	removeErr     error
	removeGen     int
	removeCancel  context.CancelFunc

	spinner  spinner.Model
	viewport viewport.Model

	width, height int
	ready         bool
}

// NewModel builds the initial application state, detecting which
// package managers are available on this machine.
func NewModel() Model {
	var drivers []driverItem
	for _, d := range driver.All() {
		if !driver.SupportsCurrentOS(d) {
			continue
		}
		drivers = append(drivers, driverItem{driver: d, available: d.Available()})
	}
	sort.Slice(drivers, func(i, j int) bool {
		return drivers[i].driver.Name() < drivers[j].driver.Name()
	})

	sp := spinner.New()
	sp.Spinner = spinner.Line
	sp.Style = panelTitleStyle

	root, _ := os.Getwd()

	return Model{
		keys:      newKeyMap(),
		help:      help.New(),
		drivers:   drivers,
		root:      root,
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
		if m.state == loadInProgress || m.removing {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			m.viewport.SetContent(m.renderMainContent())
			return m, cmd
		}
		return m, nil

	case driverDataMsg:
		if msg.gen != m.loadGen {
			return m, nil // stale result from a since-abandoned selection
		}
		m.cache = msg.cache
		m.globalPackages = msg.global
		m.localPackages = msg.local
		m.packageCursor = 0
		if msg.err != nil {
			m.state = loadError
			m.loadErr = msg.err
		} else {
			m.state = loadDone
		}
		m.viewport.SetContent(m.renderMainContent())
		m.viewport.GotoTop()
		return m, nil

	case packageRemovedMsg:
		if msg.gen != m.removeGen {
			return m, nil // stale result from a since-abandoned selection
		}
		m.removing = false
		if msg.err != nil {
			m.removeErr = msg.err
		} else {
			m.removeErr = nil
			m.removeEntryFromList(msg.entry)
		}
		m.viewport.SetContent(m.renderMainContent())
		return m, nil
	}

	return m, nil
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.confirmRemove {
		return m.handleConfirmKey(msg)
	}

	switch {
	case key.Matches(msg, m.keys.Quit):
		if m.loadCancel != nil {
			m.loadCancel()
		}
		if m.removeCancel != nil {
			m.removeCancel()
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
		if m.cursor < len(m.drivers)-1 {
			m.cursor++
		}

	case key.Matches(msg, m.keys.Select):
		return m.selectDriver(m.cursor)
	}
	return m, nil
}

func (m Model) handleMainKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Left):
		return m.setScope(scopeGlobal), nil
	case key.Matches(msg, m.keys.Right):
		return m.setScope(scopeLocal), nil
	case key.Matches(msg, m.keys.Delete):
		return m.startRemove()
	case key.Matches(msg, m.keys.Up):
		return m.movePackageCursor(-1), nil
	case key.Matches(msg, m.keys.Down):
		return m.movePackageCursor(1), nil
	}

	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

func (m Model) handleConfirmKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y", "enter":
		return m.confirmRemovePackage()
	case "n", "esc":
		m.confirmRemove = false
		m.pendingRemove = driver.Entry{}
		return m, nil
	case "ctrl+c":
		if m.loadCancel != nil {
			m.loadCancel()
		}
		if m.removeCancel != nil {
			m.removeCancel()
		}
		return m, tea.Quit
	}
	return m, nil
}

func (m Model) currentEntries() []driver.Entry {
	if m.scope == scopeLocal {
		return m.localPackages
	}
	return m.globalPackages
}

func (m Model) movePackageCursor(delta int) Model {
	entries := m.currentEntries()
	if len(entries) == 0 {
		return m
	}
	m.packageCursor = clamp(m.packageCursor+delta, 0, len(entries)-1)
	m.viewport.SetContent(m.renderMainContent())
	m.ensureCursorVisible()
	return m
}

func (m *Model) ensureCursorVisible() {
	// At the first entry, scroll all the way to the top rather than
	// just to that entry's own line, so the Paths section above it
	// (and anything else above the list) comes back into view instead
	// of staying permanently scrolled past.
	if m.packageCursor == 0 {
		m.viewport.SetYOffset(0)
		return
	}

	contentWidth := m.computeLayout().mainContentWidth
	line := m.packageCursorLineOffset(contentWidth)
	if line < m.viewport.YOffset {
		m.viewport.SetYOffset(line)
	} else if line >= m.viewport.YOffset+m.viewport.Height {
		m.viewport.SetYOffset(line - m.viewport.Height + 1)
	}
}

func (m Model) startRemove() (tea.Model, tea.Cmd) {
	if m.scope != scopeGlobal {
		return m, nil
	}
	entries := m.currentEntries()
	if m.packageCursor < 0 || m.packageCursor >= len(entries) {
		return m, nil
	}
	m.pendingRemove = entries[m.packageCursor]
	m.confirmRemove = true
	m.removeErr = nil
	return m, nil
}

func (m Model) confirmRemovePackage() (tea.Model, tea.Cmd) {
	item, ok := m.activeItem()
	m.confirmRemove = false
	if !ok {
		return m, nil
	}

	entry := m.pendingRemove
	m.pendingRemove = driver.Entry{}
	m.removing = true
	m.removeErr = nil
	m.removeGen++
	m.viewport.SetContent(m.renderMainContent())

	if m.removeCancel != nil {
		m.removeCancel()
	}
	ctx, cancel := context.WithTimeout(context.Background(), loadTimeout)
	m.removeCancel = cancel
	return m, tea.Batch(m.spinner.Tick, removePackageCmd(ctx, item.driver, entry, m.removeGen))
}

func (m *Model) removeEntryFromList(e driver.Entry) {
	list := &m.globalPackages
	if e.Kind == driver.KindLocalPackage {
		list = &m.localPackages
	}
	for i, cur := range *list {
		if cur == e {
			*list = append((*list)[:i], (*list)[i+1:]...)
			break
		}
	}
	if m.packageCursor >= len(*list) {
		m.packageCursor = maxInt(0, len(*list)-1)
	}
}

func (m Model) setScope(s packageScope) Model {
	if m.scope == s {
		return m
	}
	m.scope = s
	m.packageCursor = 0
	m.viewport.SetContent(m.renderMainContent())
	m.viewport.GotoTop()
	return m
}

func (m Model) selectDriver(idx int) (tea.Model, tea.Cmd) {
	item := m.drivers[idx]
	m.activeIdx = idx
	m.focus = focusMain
	m.scope = scopeGlobal
	m.packageCursor = 0
	m.confirmRemove = false
	m.removing = false
	m.removeErr = nil

	if !item.available {
		m.state = loadIdle
		m.cache, m.globalPackages, m.localPackages, m.loadErr = nil, nil, nil, nil
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
	m.cache, m.globalPackages, m.localPackages = nil, nil, nil
	m.viewport.SetContent(m.renderMainContent())

	return m, tea.Batch(m.spinner.Tick, loadDriverDataCmd(ctx, item.driver, m.root, m.loadGen))
}

func (m Model) activeItem() (driverItem, bool) {
	if m.activeIdx < 0 || m.activeIdx >= len(m.drivers) {
		return driverItem{}, false
	}
	return m.drivers[m.activeIdx], true
}
