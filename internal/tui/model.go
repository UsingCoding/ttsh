package tui

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/usingcoding/ttsh/internal/app"
	"github.com/usingcoding/ttsh/internal/domain"
	"github.com/usingcoding/ttsh/internal/suggestions"
)

type mode uint8

const (
	modeSheet mode = iota
	modeNew
	modeEdit
	modeCalendar
	modeHelp
	modeConfirm
	modeError
)

type Model struct {
	service         *app.Service
	provider        suggestions.Provider
	logger          *slog.Logger
	session         *app.SheetSession
	date            time.Time
	selected        int
	width, height   int
	mode            mode
	inputs          []textinput.Model
	focus           int
	editIndex       int
	suggestions     []string
	suggestionIndex int
	calendar        time.Time
	marks           map[string]bool
	errorText       string
	formError       string
	pending         domain.EntryInput
	gPending        bool
}

func New(service *app.Service, provider suggestions.Provider, logger *slog.Logger) (Model, error) {
	if provider == nil {
		provider = suggestions.None{}
	}
	if logger == nil {
		logger = slog.Default()
	}
	now := time.Now()
	session, err := service.Open(context.Background(), now)
	if err != nil {
		return Model{}, err
	}
	model := Model{service: service, provider: provider, logger: logger, session: session, date: domain.DateOnly(now), selected: -1, mode: modeSheet}
	if len(session.List()) > 0 {
		model.selected = 0
	}
	return model, nil
}

func Run(service *app.Service, provider suggestions.Provider, logger *slog.Logger) error {
	model, err := New(service, provider, logger)
	if err != nil {
		return err
	}
	final, err := tea.NewProgram(model, tea.WithAltScreen()).Run()
	if err != nil {
		if current, ok := final.(Model); ok {
			_ = current.session.Close()
		} else {
			_ = model.session.Close()
		}
		return err
	}
	current, ok := final.(Model)
	if !ok {
		return fmt.Errorf("unexpected TUI model %T", final)
	}
	return current.session.Close()
}

func (m Model) Init() tea.Cmd {
	return tea.Tick(time.Minute, func(time.Time) tea.Msg { return tickMsg{} })
}

type tickMsg struct{}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch value := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = value.Width, value.Height
		return m, nil
	case tickMsg:
		return m, m.Init()
	case tea.KeyMsg:
		if m.mode != modeSheet {
			return m.updateModal(value)
		}
		return m.updateSheet(value)
	}
	return m, nil
}

func (m Model) updateSheet(key tea.KeyMsg) (tea.Model, tea.Cmd) {
	entries := m.session.List()
	switch key.String() {
	case "q", "ctrl+c":
		if err := m.session.Close(); err != nil {
			m.errorText = err.Error()
			m.mode = modeError
			return m, nil
		}
		return m, tea.Quit
	case "j", "down":
		m.gPending = false
		if len(entries) > 0 && m.selected < len(entries)-1 {
			m.selected++
		}
	case "k", "up":
		m.gPending = false
		if len(entries) > 0 && m.selected > 0 {
			m.selected--
		}
	case "g":
		if m.gPending && len(entries) > 0 {
			m.selected = 0
			m.gPending = false
		} else {
			m.gPending = true
		}
	case "G":
		m.gPending = false
		if len(entries) > 0 {
			m.selected = len(entries) - 1
		}
	case "n":
		m.gPending = false
		m.openForm(false, 0)
	case "e":
		m.gPending = false
		if m.selected >= 0 {
			m.openForm(true, m.selected+1)
		}
	case "s":
		m.gPending = false
		if m.selected >= 0 {
			if _, err := m.session.Stop(m.selected + 1); err != nil {
				m.showError(err)
			}
		}
	case "c":
		m.gPending = false
		m.openCalendar()
	case "?":
		m.gPending = false
		m.mode = modeHelp
	default:
		m.gPending = false
	}
	return m, nil
}

func (m *Model) openForm(edit bool, index int) {
	values := []string{"", domain.TimeOfDayAt(time.Now()).String(), "", ""}
	m.mode = modeNew
	m.editIndex = 0
	m.formError = ""
	if edit {
		entry, err := m.session.Get(index)
		if err != nil {
			m.showError(err)
			return
		}
		values = []string{entry.Name, entry.Start.String(), "", entry.Description}
		if entry.End != nil {
			values[2] = entry.End.String()
		}
		m.mode = modeEdit
		m.editIndex = index
	}
	m.inputs = make([]textinput.Model, 4)
	for i := range m.inputs {
		input := textinput.New()
		input.SetValue(values[i])
		input.CharLimit = 256
		m.inputs[i] = input
	}
	m.focus = 0
	m.inputs[0].Focus()
	m.suggestions = m.provider.Names(context.Background())
	m.suggestionIndex = 0
}

func (m *Model) updateModal(key tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.mode {
	case modeHelp, modeError:
		if key.String() == "esc" || key.String() == "?" || key.String() == "enter" {
			m.mode = modeSheet
			m.errorText = ""
		}
		return *m, nil
	case modeConfirm:
		switch key.String() {
		case "y":
			m.confirmAdd()
		case "n", "esc":
			m.mode = modeNew
		}
		return *m, nil
	case modeCalendar:
		return m.updateCalendar(key)
	case modeNew, modeEdit:
		return m.updateForm(key)
	}
	return *m, nil
}

func (m *Model) updateForm(key tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch key.String() {
	case "esc":
		m.mode = modeSheet
		return *m, nil
	case "enter":
		m.submitForm()
		return *m, nil
	case "tab":
		if m.focus == 0 && len(m.filteredSuggestions()) > 0 {
			m.inputs[0].SetValue(m.filteredSuggestions()[m.suggestionIndex])
		}
		m.setFocus((m.focus + 1) % len(m.inputs))
		return *m, nil
	case "shift+tab":
		m.setFocus((m.focus + len(m.inputs) - 1) % len(m.inputs))
		return *m, nil
	case "up":
		if m.focus == 0 {
			m.moveSuggestion(-1)
			return *m, nil
		}
	case "down":
		if m.focus == 0 {
			m.moveSuggestion(1)
			return *m, nil
		}
	}
	var cmd tea.Cmd
	m.inputs[m.focus], cmd = m.inputs[m.focus].Update(key)
	m.suggestionIndex = 0
	return *m, cmd
}
func (m *Model) setFocus(index int) {
	m.inputs[m.focus].Blur()
	m.focus = index
	m.inputs[m.focus].Focus()
}
func (m *Model) moveSuggestion(delta int) {
	items := m.filteredSuggestions()
	if len(items) == 0 {
		return
	}
	m.suggestionIndex = (m.suggestionIndex + delta + len(items)) % len(items)
}
func (m *Model) filteredSuggestions() []string {
	prefix := strings.ToLower(m.inputs[0].Value())
	result := []string{}
	for _, name := range m.suggestions {
		if strings.HasPrefix(strings.ToLower(name), prefix) {
			result = append(result, name)
		}
	}
	if m.suggestionIndex >= len(result) {
		m.suggestionIndex = 0
	}
	return result
}
func (m *Model) submitForm() {
	start, err := domain.ParseTimeOfDay(m.inputs[1].Value())
	if err != nil {
		m.setFormError(err)
		return
	}
	input := domain.EntryInput{Name: m.inputs[0].Value(), Start: start, Description: m.inputs[3].Value()}
	if value := m.inputs[2].Value(); value != "" {
		end, err := domain.ParseTimeOfDay(value)
		if err != nil {
			m.setFormError(err)
			return
		}
		input.End = &end
	}
	if err := input.Validate(); err != nil {
		m.setFormError(err)
		return
	}
	if m.mode == modeEdit {
		if _, err := m.session.Update(m.editIndex, input); err != nil {
			m.setFormError(err)
			return
		}
		m.selected = m.editIndex - 1
		m.mode = modeSheet
		return
	}
	index, _, err := m.session.Add(input, app.RejectActive)
	if conflict, ok := err.(domain.ActiveEntryConflictError); ok {
		m.pending = input
		m.errorText = fmt.Sprintf("Another entry is active: <%s> started at %s. Stop it at %s and start new?", conflict.Entry.Name, conflict.Entry.Start, domain.TimeOfDayAt(time.Now()))
		m.mode = modeConfirm
		return
	}
	if err != nil {
		m.setFormError(err)
		return
	}
	m.selected = index - 1
	m.mode = modeSheet
}
func (m *Model) confirmAdd() {
	index, _, err := m.session.Add(m.pending, app.StopExistingActive)
	if err != nil {
		m.showError(err)
		return
	}
	m.selected = index - 1
	m.mode = modeSheet
}
func (m *Model) showError(err error)    { m.errorText = err.Error(); m.mode = modeError }
func (m *Model) setFormError(err error) { m.formError = err.Error() }

func (m *Model) openCalendar() {
	m.calendar = m.date
	if err := m.loadCalendarMarks(); err != nil {
		m.showError(err)
		return
	}
	m.mode = modeCalendar
}

func (m *Model) loadCalendarMarks() error {
	m.marks = map[string]bool{}
	dates, err := m.service.DatesWithEntriesForSession(context.Background(), m.calendar, m.session)
	if err != nil {
		return err
	}
	for _, date := range dates {
		m.marks[date.Format(time.DateOnly)] = true
	}
	return nil
}
func (m Model) updateCalendar(key tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch key.String() {
	case "esc":
		m.mode = modeSheet
	case "h", "left":
		m.calendar = m.calendar.AddDate(0, 0, -1)
	case "l", "right":
		m.calendar = m.calendar.AddDate(0, 0, 1)
	case "k", "up":
		m.calendar = m.calendar.AddDate(0, 0, -7)
	case "j", "down":
		m.calendar = m.calendar.AddDate(0, 0, 7)
	case "pgup":
		m.calendar = m.calendar.AddDate(0, -1, 0)
	case "pgdown":
		m.calendar = m.calendar.AddDate(0, 1, 0)
	case "t":
		m.calendar = domain.DateOnly(time.Now())
	case "enter":
		old := m.date
		if err := m.session.Close(); err != nil {
			m.showError(err)
			return m, nil
		}
		session, err := m.service.Open(context.Background(), m.calendar)
		if err != nil {
			recovery, recoveryErr := m.service.Open(context.Background(), old)
			if recoveryErr != nil {
				return m, tea.Quit
			}
			m.session = recovery
			m.date = old
			m.showError(err)
			return m, nil
		}
		m.session = session
		m.date = m.calendar
		m.selected = -1
		if len(session.List()) > 0 {
			m.selected = 0
		}
		m.mode = modeSheet
	}
	if m.mode == modeCalendar && key.String() != "enter" && key.String() != "esc" {
		if err := m.loadCalendarMarks(); err != nil {
			m.showError(err)
		}
	}
	return m, nil
}

func (m Model) View() string {
	base := m.renderSheet()
	switch m.mode {
	case modeNew, modeEdit:
		return overlay(base, m.renderForm())
	case modeCalendar:
		return overlay(base, m.renderCalendar())
	case modeHelp:
		return overlay(base, "Help\n\nj/↓ next   k/↑ previous   gg first   G last\nn new   e edit   s stop   c calendar\n? help   q quit\n\nEsc close")
	case modeConfirm:
		return overlay(base, m.errorText+"\n\ny yes   n/Esc cancel")
	case modeError:
		return overlay(base, "Error\n\n"+m.errorText+"\n\nEsc close")
	}
	return base
}
func (m Model) renderSheet() string {
	entries := m.session.List()
	lines := []string{fmt.Sprintf("time — %s", m.date.Format("Mon 2006-01-02")), ""}
	if len(entries) == 0 {
		lines = append(lines, "No entries yet", "n new entry   c calendar")
	} else {
		for i, entry := range entries {
			marker := " "
			if i == m.selected {
				marker = ">"
			}
			end := "..."
			if entry.End != nil {
				end = entry.End.String()
			}
			line := fmt.Sprintf("%s <%s> @%s -> %s", marker, entry.Name, entry.Start, end)
			if entry.Description != "" {
				line += " # " + entry.Description
			}
			lines = append(lines, line)
		}
	}
	selection := "--"
	if m.selected >= 0 && m.selected < len(entries) {
		selection = fmt.Sprintf("#%d %s", m.selected+1, domain.FormatDuration(m.session.EntryDuration(entries[m.selected])))
	}
	lines = append(lines, "", fmt.Sprintf("%s    %d entries · %s", selection, len(entries), domain.FormatDuration(m.session.CompletedTotal())), "n new   e edit   s stop   c calendar   ? help   q quit")
	return strings.Join(lines, "\n")
}
func (m Model) renderForm() string {
	title := "New entry"
	if m.mode == modeEdit {
		title = "Edit entry"
	}
	labels := []string{"Name *", "Start", "End", "Description"}
	lines := []string{title, ""}
	for i, label := range labels {
		marker := " "
		if i == m.focus {
			marker = ">"
		}
		lines = append(lines, label, marker+m.inputs[i].View())
	}
	if m.focus == 0 {
		items := m.filteredSuggestions()
		if len(items) > 0 {
			lines = append(lines, "Suggestions")
			for i, item := range items {
				marker := " "
				if i == m.suggestionIndex {
					marker = ">"
				}
				lines = append(lines, marker+item)
			}
		}
	}
	if m.formError != "" {
		lines = append(lines, "", "Error: "+m.formError)
	}
	lines = append(lines, "", "Tab next · Shift-Tab previous · Enter save · Esc cancel")
	return strings.Join(lines, "\n")
}
func (m Model) renderCalendar() string {
	month := time.Date(m.calendar.Year(), m.calendar.Month(), 1, 0, 0, 0, 0, time.Local)
	lines := []string{month.Format("January 2006"), "Mon Tue Wed Thu Fri Sat Sun"}
	offset := (int(month.Weekday()) + 6) % 7
	row := strings.Repeat("    ", offset)
	for day := 1; day <= daysInMonth(month); day++ {
		d := time.Date(month.Year(), month.Month(), day, 0, 0, 0, 0, time.Local)
		value := fmt.Sprintf("%02d", day)
		if m.marks[d.Format(time.DateOnly)] {
			value += "*"
		}
		if domain.SameDate(d, m.calendar) {
			value = "[" + value + "]"
		}
		row += fmt.Sprintf("%-4s", value)
		if (offset+day)%7 == 0 {
			lines = append(lines, row)
			row = ""
		}
	}
	if row != "" {
		lines = append(lines, row)
	}
	lines = append(lines, "", "hjkl move · PgUp/PgDn month · t today", "Enter open · Esc cancel")
	return strings.Join(lines, "\n")
}
func daysInMonth(month time.Time) int { return month.AddDate(0, 1, -1).Day() }
func overlay(base, panel string) string {
	style := lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(1)
	return base + "\n\n" + style.Render(panel)
}
