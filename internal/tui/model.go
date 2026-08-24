package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/mikey/faultline/internal/event"
	"github.com/mikey/faultline/internal/source"
	"github.com/mikey/faultline/internal/store"
)

type viewMode int

const (
	viewList viewMode = iota
	viewDetail
	viewFilter
)

// OpenFunc opens a file at a line in an editor.
type OpenFunc func(file string, line int) error

// Deps wires TUI to the rest of the app.
type Deps struct {
	Store         *store.Store
	Open          OpenFunc
	InitialStatus []source.Status
	Events        <-chan event.Event
	StatusCh      <-chan source.Status
}

type Model struct {
	store    *store.Store
	open     OpenFunc
	events   <-chan event.Event
	statusCh <-chan source.Status

	statuses  []source.Status
	items     []event.Event
	cursor    int
	mode      viewMode
	filter    string
	filterIn  textinput.Model
	sortFreq  bool
	width     int
	height    int
	statusMsg string
	errMsg    string
}

func New(d Deps) Model {
	ti := textinput.New()
	ti.Placeholder = "filter type, message, file…"
	ti.CharLimit = 120
	ti.Width = 40

	m := Model{
		store:    d.Store,
		open:     d.Open,
		events:   d.Events,
		statusCh: d.StatusCh,
		statuses: append([]source.Status(nil), d.InitialStatus...),
		filterIn: ti,
		mode:     viewList,
	}
	m.refresh()
	return m
}

type tickMsg time.Time
type eventMsg event.Event
type statusMsg source.Status

func (m Model) Init() tea.Cmd {
	return tea.Batch(waitEvent(m.events), waitStatus(m.statusCh), tick())
}

func tick() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg { return tickMsg(t) })
}

func waitEvent(ch <-chan event.Event) tea.Cmd {
	return func() tea.Msg {
		if ch == nil {
			return nil
		}
		ev, ok := <-ch
		if !ok {
			return nil
		}
		return eventMsg(ev)
	}
}

func waitStatus(ch <-chan source.Status) tea.Cmd {
	return func() tea.Msg {
		if ch == nil {
			return nil
		}
		st, ok := <-ch
		if !ok {
			return nil
		}
		return statusMsg(st)
	}
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tickMsg:
		m.refresh()
		return m, tick()

	case eventMsg:
		m.refresh()
		m.statusMsg = fmt.Sprintf("New: %s", event.Event(msg).Title())
		return m, waitEvent(m.events)

	case statusMsg:
		m.upsertStatus(source.Status(msg))
		return m, waitStatus(m.statusCh)

	case tea.KeyMsg:
		if m.mode == viewFilter {
			return m.updateFilter(msg)
		}
		switch {
		case key.Matches(msg, keys.Quit):
			return m, tea.Quit
		case key.Matches(msg, keys.Up):
			if m.cursor > 0 {
				m.cursor--
			}
		case key.Matches(msg, keys.Down):
			if m.cursor < len(m.items)-1 {
				m.cursor++
			}
		case key.Matches(msg, keys.Enter):
			if m.mode == viewList && len(m.items) > 0 {
				m.mode = viewDetail
			} else if m.mode == viewDetail {
				m.mode = viewList
			}
		case key.Matches(msg, keys.Back):
			m.mode = viewList
		case key.Matches(msg, keys.Open):
			m.openSelected()
		case key.Matches(msg, keys.Clear):
			m.store.Clear()
			m.refresh()
			m.statusMsg = "Cleared"
		case key.Matches(msg, keys.Filter):
			m.mode = viewFilter
			m.filterIn.SetValue(m.filter)
			m.filterIn.Focus()
			return m, textinput.Blink
		case key.Matches(msg, keys.Sort):
			m.sortFreq = !m.sortFreq
			m.refresh()
		}
	}
	return m, nil
}

func (m Model) updateFilter(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEnter:
		m.filter = strings.TrimSpace(m.filterIn.Value())
		m.mode = viewList
		m.filterIn.Blur()
		m.refresh()
		return m, nil
	case tea.KeyEsc:
		m.mode = viewList
		m.filterIn.Blur()
		return m, nil
	}
	var cmd tea.Cmd
	m.filterIn, cmd = m.filterIn.Update(msg)
	return m, cmd
}

func (m *Model) openSelected() {
	ev, ok := m.selected()
	if !ok {
		m.errMsg = "nothing selected"
		return
	}
	if m.open == nil {
		m.errMsg = "editor not configured"
		return
	}
	if err := m.open(ev.File, ev.Line); err != nil {
		m.errMsg = err.Error()
		m.statusMsg = ""
		return
	}
	m.errMsg = ""
	m.statusMsg = fmt.Sprintf("Opened %s", ev.Location())
}

func (m *Model) selected() (event.Event, bool) {
	if len(m.items) == 0 || m.cursor < 0 || m.cursor >= len(m.items) {
		return event.Event{}, false
	}
	return m.items[m.cursor], true
}

func (m *Model) refresh() {
	if m.sortFreq {
		m.items = m.store.ByFrequency(m.filter)
	} else {
		m.items = m.store.List(m.filter)
	}
	if m.cursor >= len(m.items) {
		m.cursor = len(m.items) - 1
	}
	if m.cursor < 0 {
		m.cursor = 0
	}
}

func (m *Model) upsertStatus(st source.Status) {
	for i := range m.statuses {
		if m.statuses[i].Name == st.Name {
			m.statuses[i] = st
			return
		}
	}
	m.statuses = append(m.statuses, st)
}

var (
	titleStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("212"))
	okStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
	waitStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("214"))
	errStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	selStyle   = lipgloss.NewStyle().Background(lipgloss.Color("236")).Bold(true)
	dimStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	critStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true)
	errorStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("203"))
	warnStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("214"))
	helpStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
)

func (m Model) View() string {
	if m.width == 0 {
		m.width = 80
	}

	var b strings.Builder
	b.WriteString(titleStyle.Render("Faultline"))
	b.WriteString("\n")
	b.WriteString(m.viewSources())
	b.WriteString("\n")

	switch m.mode {
	case viewDetail:
		b.WriteString(m.viewDetail())
	case viewFilter:
		b.WriteString("Filter: ")
		b.WriteString(m.filterIn.View())
		b.WriteString("\n")
		b.WriteString(dimStyle.Render("Enter apply · Esc cancel"))
	default:
		b.WriteString(m.viewList())
	}

	b.WriteString("\n")
	if m.errMsg != "" {
		b.WriteString(errStyle.Render(m.errMsg))
		b.WriteString("\n")
	} else if m.statusMsg != "" {
		b.WriteString(dimStyle.Render(m.statusMsg))
		b.WriteString("\n")
	}
	b.WriteString(helpStyle.Render("↑/↓ move · Enter detail · o open · c clear · / filter · s sort · q quit"))
	return b.String()
}

func (m Model) viewSources() string {
	var lines []string
	lines = append(lines, titleStyle.Render("Watching"))
	for _, st := range m.statuses {
		mark := "•"
		style := dimStyle
		switch st.State {
		case source.StateOK:
			mark = "✓"
			style = okStyle
		case source.StateWaiting:
			mark = "…"
			style = waitStyle
		case source.StateError:
			mark = "✗"
			style = errStyle
		}
		line := fmt.Sprintf("%s %s  %s", mark, st.Name, st.Path)
		if st.Message != "" && st.State != source.StateOK {
			line += "  (" + st.Message + ")"
		}
		lines = append(lines, style.Render(line))
	}
	return strings.Join(lines, "\n")
}

func (m Model) viewList() string {
	var b strings.Builder
	header := fmt.Sprintf("New Errors (%d)", len(m.items))
	if m.filter != "" {
		header += fmt.Sprintf("  filter:%q", m.filter)
	}
	if m.sortFreq {
		header += "  sort:frequency"
	}
	b.WriteString(titleStyle.Render(header))
	b.WriteString("\n")
	if len(m.items) == 0 {
		b.WriteString(dimStyle.Render("No errors yet — waiting for log activity."))
		return b.String()
	}

	max := len(m.items)
	// Keep list reasonably short for terminal height.
	visible := max
	if m.height > 0 {
		visible = m.height - 12
		if visible < 3 {
			visible = 3
		}
	}
	start := 0
	if m.cursor >= visible {
		start = m.cursor - visible + 1
	}
	end := start + visible
	if end > max {
		end = max
	}

	for i := start; i < end; i++ {
		ev := m.items[i]
		icon := severityIcon(ev.Severity)
		loc := ev.Location()
		if loc == "" {
			loc = "(no location)"
		}
		count := ""
		if ev.Count > 1 {
			count = fmt.Sprintf(" ×%d", ev.Count)
		}
		line1 := fmt.Sprintf("%s %s%s", icon, ev.Title(), count)
		line2 := fmt.Sprintf("  %s", loc)
		line3 := fmt.Sprintf("  %s · %s", relativeTime(ev.LastSeen), ev.Source)
		block := line1 + "\n" + dimStyle.Render(line2) + "\n" + dimStyle.Render(line3)
		if i == m.cursor {
			block = selStyle.Render(line1) + "\n" + selStyle.Render(line2) + "\n" + selStyle.Render(line3)
		}
		b.WriteString(block)
		b.WriteString("\n\n")
	}
	return b.String()
}

func (m Model) viewDetail() string {
	ev, ok := m.selected()
	if !ok {
		return dimStyle.Render("No error selected.")
	}
	var b strings.Builder
	b.WriteString(titleStyle.Render(ev.Title()))
	b.WriteString("\n")
	b.WriteString(fmt.Sprintf("Severity: %s\n", ev.Severity))
	b.WriteString(fmt.Sprintf("Source:   %s\n", ev.Source))
	if loc := ev.Location(); loc != "" {
		b.WriteString(fmt.Sprintf("Location: %s\n", loc))
	}
	b.WriteString(fmt.Sprintf("Count:    %d\n", ev.Count))
	b.WriteString(fmt.Sprintf("First:    %s\n", ev.FirstSeen.Format(time.RFC3339)))
	b.WriteString(fmt.Sprintf("Last:     %s\n", ev.LastSeen.Format(time.RFC3339)))
	b.WriteString("\n")
	b.WriteString(titleStyle.Render("Message"))
	b.WriteString("\n")
	b.WriteString(wrap(ev.Message, max(40, m.width-4)))
	b.WriteString("\n\n")
	if ev.Stack != "" {
		b.WriteString(titleStyle.Render("Stack"))
		b.WriteString("\n")
		b.WriteString(dimStyle.Render(truncateLines(ev.Stack, 20)))
		b.WriteString("\n\n")
	}
	b.WriteString(helpStyle.Render("Esc/Enter back · o open in editor"))
	return b.String()
}

func severityIcon(s event.Severity) string {
	switch s {
	case event.SeverityCritical:
		return critStyle.Render("●")
	case event.SeverityError:
		return errorStyle.Render("●")
	case event.SeverityWarning:
		return warnStyle.Render("●")
	default:
		return dimStyle.Render("●")
	}
}

func relativeTime(t time.Time) string {
	if t.IsZero() {
		return "unknown"
	}
	d := time.Since(t)
	switch {
	case d < time.Second:
		return "just now"
	case d < time.Minute:
		return fmt.Sprintf("%d seconds ago", int(d.Seconds()))
	case d < time.Hour:
		return fmt.Sprintf("%d minutes ago", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%d hours ago", int(d.Hours()))
	default:
		return t.Format("Jan 2 15:04")
	}
}

func wrap(s string, width int) string {
	if width <= 0 || len(s) <= width {
		return s
	}
	var b strings.Builder
	for len(s) > width {
		b.WriteString(s[:width])
		b.WriteByte('\n')
		s = s[width:]
	}
	b.WriteString(s)
	return b.String()
}

func truncateLines(s string, n int) string {
	lines := strings.Split(s, "\n")
	if len(lines) <= n {
		return s
	}
	return strings.Join(lines[:n], "\n") + "\n…"
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

type keyMap struct {
	Quit   key.Binding
	Up     key.Binding
	Down   key.Binding
	Enter  key.Binding
	Back   key.Binding
	Open   key.Binding
	Clear  key.Binding
	Filter key.Binding
	Sort   key.Binding
}

var keys = keyMap{
	Quit:   key.NewBinding(key.WithKeys("q", "ctrl+c")),
	Up:     key.NewBinding(key.WithKeys("up", "k")),
	Down:   key.NewBinding(key.WithKeys("down", "j")),
	Enter:  key.NewBinding(key.WithKeys("enter")),
	Back:   key.NewBinding(key.WithKeys("esc", "backspace")),
	Open:   key.NewBinding(key.WithKeys("o")),
	Clear:  key.NewBinding(key.WithKeys("c")),
	Filter: key.NewBinding(key.WithKeys("/")),
	Sort:   key.NewBinding(key.WithKeys("s")),
}
