package tui

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
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

const (
	nameField = iota
	startField
	endField
	descriptionField
	formFieldCount
	descriptionHeight = 3
)

type Model struct {
	service         *app.Service
	provider        suggestions.Provider
	logger          *slog.Logger
	session         *app.SheetSession
	date            time.Time
	selected        int
	width, height   int
	theme           Theme
	styles          styles
	mode            mode
	inputs          []textinput.Model
	description     textarea.Model
	focus           int
	editIndex       int
	suggestions     []suggestions.Suggestion
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
	theme := defaultMutedSlateTheme()
	model := Model{
		service: service, provider: provider, logger: logger, session: session,
		date: domain.DateOnly(now), selected: -1, mode: modeSheet,
		theme: theme, styles: newStyles(theme),
	}
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
		m.configureInputs()
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
		return m, m.openForm(false, 0)
	case "e":
		m.gPending = false
		if m.selected >= 0 {
			return m, m.openForm(true, m.selected+1)
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

func (m *Model) openForm(edit bool, index int) tea.Cmd {
	values := []string{"", domain.TimeOfDayAt(time.Now()).String(), "", ""}
	m.mode = modeNew
	m.editIndex = 0
	m.formError = ""
	if edit {
		entry, err := m.session.Get(index)
		if err != nil {
			m.showError(err)
			return nil
		}
		values = []string{entry.Name, entry.Start.String(), "", entry.Description}
		if entry.End != nil {
			values[2] = entry.End.String()
		}
		m.mode = modeEdit
		m.editIndex = index
	}
	m.inputs = make([]textinput.Model, descriptionField)
	for i := range m.inputs {
		input := textinput.New()
		input.SetValue(values[i])
		input.CharLimit = 256
		m.inputs[i] = input
	}
	m.description = textarea.New()
	m.description.SetValue(values[descriptionField])
	m.description.CharLimit = 256
	m.configureInputs()
	m.description.Blur()
	m.focus = nameField
	focusCmd := m.setFocus(nameField)
	m.suggestions = m.provider.Suggestions(context.Background())
	m.suggestionIndex = 0
	return focusCmd
}

func (m *Model) configureInputs() {
	width := 52
	if viewport, ok := newViewport(m.width, m.height); ok {
		width = min(52, max(viewport.width-10, 8))
	}
	if len(m.inputs) == 0 {
		return
	}
	for i := range m.inputs {
		m.inputs[i].Prompt = ""
		m.inputs[i].Width = width
		m.inputs[i].TextStyle = m.styles.formFocused
		m.inputs[i].PlaceholderStyle = m.styles.formInactive
		m.inputs[i].CompletionStyle = m.styles.suggestion
		m.inputs[i].Cursor.Style = m.styles.cursor
	}
	focusedDescription := textarea.Style{
		Base:        m.styles.formFocused,
		CursorLine:  m.styles.formFocused,
		EndOfBuffer: m.styles.formFocused,
		Placeholder: m.styles.formFocused,
		Prompt:      m.styles.formFocused,
		Text:        m.styles.formFocused,
	}
	blurredDescription := textarea.Style{
		Base:        m.styles.formInactive,
		CursorLine:  m.styles.formInactive,
		EndOfBuffer: m.styles.formInactive,
		Placeholder: m.styles.formInactive,
		Prompt:      m.styles.formInactive,
		Text:        m.styles.formInactive,
	}
	m.description.Prompt = ""
	m.description.ShowLineNumbers = false
	m.description.FocusedStyle = focusedDescription
	m.description.BlurredStyle = blurredDescription
	m.description.Cursor.Style = m.styles.cursor
	m.description.SetWidth(width)
	m.description.SetHeight(descriptionHeight)
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
		if m.focus == nameField {
			items := m.filteredSuggestions()
			if len(items) > 0 {
				m.inputs[nameField].SetValue(items[m.suggestionIndex].Name)
			}
		}
		return *m, m.setFocus((m.focus + 1) % formFieldCount)
	case "shift+tab":
		return *m, m.setFocus((m.focus + formFieldCount - 1) % formFieldCount)
	case "up":
		if m.focus == nameField {
			m.moveSuggestion(-1)
			return *m, nil
		}
	case "down":
		if m.focus == nameField {
			m.moveSuggestion(1)
			return *m, nil
		}
	}
	var cmd tea.Cmd
	if m.focus == descriptionField {
		m.description, cmd = m.description.Update(key)
	} else {
		m.inputs[m.focus], cmd = m.inputs[m.focus].Update(key)
	}
	m.suggestionIndex = 0
	return *m, cmd
}
func (m *Model) setFocus(index int) tea.Cmd {
	if m.focus == descriptionField {
		m.description.Blur()
	} else {
		m.inputs[m.focus].Blur()
	}
	m.focus = index
	if m.focus == descriptionField {
		return m.description.Focus()
	}
	return m.inputs[m.focus].Focus()
}
func (m *Model) moveSuggestion(delta int) {
	items := m.filteredSuggestions()
	if len(items) == 0 {
		return
	}
	m.suggestionIndex = (m.suggestionIndex + delta + len(items)) % len(items)
}
func (m *Model) filteredSuggestions() []suggestions.Suggestion {
	prefix := strings.ToLower(m.inputs[0].Value())
	result := make([]suggestions.Suggestion, 0, len(m.suggestions))
	for _, suggestion := range m.suggestions {
		if strings.HasPrefix(strings.ToLower(suggestion.Name), prefix) {
			result = append(result, suggestion)
		}
	}
	if m.suggestionIndex >= len(result) {
		m.suggestionIndex = 0
	}
	return result
}
func (m *Model) submitForm() {
	start, err := domain.ParseTimeOfDay(m.inputs[startField].Value())
	if err != nil {
		m.setFormError(err)
		return
	}
	input := domain.EntryInput{Name: m.inputs[nameField].Value(), Start: start, Description: m.description.Value()}
	if value := m.inputs[endField].Value(); value != "" {
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
	viewport, full := newViewport(m.width, m.height)
	if !full {
		return m.renderIntrinsic()
	}
	base := m.sheetCanvas(viewport)
	if panel, ok := m.renderPopup(); ok {
		base.popup(viewport, panel, m.styles)
	}
	return base.render()
}

func (m Model) sheetCanvas(viewport viewport) *canvas {
	sheet := newCanvas(viewport.width, viewport.height, m.styles.canvas)
	sheet.fill(0, 0, viewport.width, headerRows, m.styles.header)
	sheet.write(1, 0, "ttsh", m.styles.headerTitle)
	date := m.date.Format("Mon 2006-01-02")
	if dateX := viewport.width - ansi.StringWidth(date) - 1; dateX > 6 {
		sheet.write(dateX, 0, date, m.styles.date)
	}

	entries := m.session.List()
	if len(entries) == 0 {
		m.renderEmptyState(sheet, viewport)
	} else {
		m.renderEntries(sheet, viewport, entries)
	}
	m.renderFooter(sheet, viewport, entries)
	return sheet
}

func (m Model) renderEmptyState(sheet *canvas, viewport viewport) {
	row := viewport.bodyTop + min(max(viewport.bodyHeight()/3, 0), max(viewport.bodyHeight()-1, 0))
	if row >= viewport.bodyBottom {
		return
	}
	sheet.write(viewport.contentX, row, clipped("No entries today", viewport.contentWidth), m.styles.empty)
	if row+1 < viewport.bodyBottom {
		sheet.write(viewport.contentX, row+1, clipped("n create entry", viewport.contentWidth), m.styles.footerKey)
	}
}

func (m Model) renderEntries(sheet *canvas, viewport viewport, entries []domain.Entry) {
	window := visibleEntries(len(entries), m.selected, viewport.bodyHeight())
	row := viewport.bodyTop
	if window.above && row < viewport.bodyBottom {
		sheet.write(viewport.contentX, row, clipped("↑ earlier entries", viewport.contentWidth), m.styles.empty)
		row++
	}
	for index := window.start; index < window.end && row < viewport.bodyBottom; index++ {
		m.renderEntry(sheet, viewport, row, index, entries[index])
		row++
	}
	if window.below && row < viewport.bodyBottom {
		sheet.write(viewport.contentX, row, clipped("↓ later entries", viewport.contentWidth), m.styles.empty)
	}
}

func (m Model) renderEntry(sheet *canvas, viewport viewport, row, index int, entry domain.Entry) {
	selected := index == m.selected
	rowStyle := m.styles.entry
	nameStyle, startStyle, endStyle, descriptionStyle := m.styles.entryName, m.styles.entryStart, m.styles.entryEnd, m.styles.entryDescription
	runningStyle := m.styles.running
	markerStyle := m.styles.entry
	if selected {
		rowStyle = m.styles.entrySelected
		nameStyle, startStyle, endStyle, descriptionStyle = m.styles.entrySelectedName, m.styles.entrySelectedStart, m.styles.entrySelectedEnd, m.styles.entrySelectedDesc
		runningStyle = m.styles.runningSelected
		markerStyle = m.styles.cursor
	}
	sheet.fill(viewport.contentX, row, viewport.contentWidth, 1, rowStyle)
	x := viewport.contentX
	marker := " "
	if selected {
		marker = ">"
	}
	sheet.write(x, row, marker, markerStyle)
	x += 2
	name := clipped("<"+entry.Name+">", max(viewport.contentWidth-(x-viewport.contentX), 0))
	sheet.write(x, row, name, nameStyle)
	x += ansi.StringWidth(name)
	sheet.write(x, row, " ", rowStyle)
	x++
	start := entry.Start.String()
	sheet.write(x, row, start, startStyle)
	x += ansi.StringWidth(start)
	interval := " → "
	sheet.write(x, row, interval, rowStyle)
	x += ansi.StringWidth(interval)
	end := "..."
	if entry.End != nil {
		end = entry.End.String()
	}
	if entry.End == nil {
		sheet.write(x, row, end, runningStyle)
	} else {
		sheet.write(x, row, end, endStyle)
	}
	x += ansi.StringWidth(end)
	if entry.Description != "" && x < viewport.contentX+viewport.contentWidth {
		prefix := " · "
		sheet.write(x, row, prefix, rowStyle)
		x += ansi.StringWidth(prefix)
		sheet.write(x, row, clipped(entry.Description, viewport.contentX+viewport.contentWidth-x), descriptionStyle)
	}
}

func (m Model) renderFooter(sheet *canvas, viewport viewport, entries []domain.Entry) {
	statusRow := viewport.height - 2
	hintsRow := viewport.height - 1
	if statusRow < 0 {
		return
	}
	sheet.fill(0, statusRow, viewport.width, 1, m.styles.status)
	selection := "--"
	if m.selected >= 0 && m.selected < len(entries) {
		selection = fmt.Sprintf("#%d %s", m.selected+1, domain.FormatDuration(m.session.EntryDuration(entries[m.selected])))
	}
	sheet.write(1, statusRow, clipped(selection, max(viewport.width-2, 0)), m.styles.statusText)
	summary := fmt.Sprintf("%d entries · %s", len(entries), domain.FormatDuration(m.session.CompletedTotal()))
	if summaryWidth := ansi.StringWidth(summary); summaryWidth+ansi.StringWidth(selection)+3 <= viewport.width {
		sheet.write(viewport.width-summaryWidth-1, statusRow, summary, m.styles.statusText)
	}
	if hintsRow < 0 {
		return
	}
	sheet.fill(0, hintsRow, viewport.width, 1, m.styles.status)
	hints := []struct{ key, label string }{
		{"n", "new"}, {"e", "edit"}, {"s", "stop"}, {"c", "calendar"}, {"?", "help"}, {"q", "quit"},
	}
	x := 1
	for _, hint := range hints {
		needed := ansi.StringWidth(hint.key) + 1 + ansi.StringWidth(hint.label) + 2
		if x+needed > viewport.width {
			if x+3 > viewport.width {
				break
			}
			sheet.write(x, hintsRow, hint.key, m.styles.footerKey)
			x += ansi.StringWidth(hint.key) + 1
			continue
		}
		sheet.write(x, hintsRow, hint.key, m.styles.footerKey)
		x += ansi.StringWidth(hint.key)
		sheet.write(x, hintsRow, " "+hint.label, m.styles.footerLabel)
		x += 1 + ansi.StringWidth(hint.label) + 2
	}
}

func (m Model) renderPopup() (popup, bool) {
	switch m.mode {
	case modeNew, modeEdit:
		return m.renderForm(), true
	case modeCalendar:
		return m.renderCalendar(), true
	case modeHelp:
		return popup{title: "Help", preferredWidth: 64, lines: []popupLine{
			{text: "j/↓ next   k/↑ previous   gg first   G last", style: m.styles.confirmation},
			{text: "n new   e edit   s stop   c calendar", style: m.styles.confirmation},
			{text: "? help   q quit", style: m.styles.confirmation},
			{text: "", style: m.styles.confirmation, optional: true},
			{text: "Esc close", style: m.styles.footerKey},
		}}, true
	case modeConfirm:
		return popup{title: "Stop active entry?", preferredWidth: 60, lines: []popupLine{
			{text: m.errorText, style: m.styles.confirmation},
			{text: "", style: m.styles.confirmation, optional: true},
			{text: "y yes   n/Esc cancel", style: m.styles.footerKey},
		}}, true
	case modeError:
		return popup{title: "Error", preferredWidth: 60, lines: []popupLine{
			{text: m.errorText, style: m.styles.error},
			{text: "", style: m.styles.error, optional: true},
			{text: "Esc close", style: m.styles.footerKey},
		}}, true
	}
	return popup{}, false
}

func (m Model) renderForm() popup {
	title := "New entry"
	if m.mode == modeEdit {
		title = "Edit entry"
	}
	labels := []string{"Name *", "Start", "End", "Description"}
	lines := make([]popupLine, 0, 18)
	for i, label := range labels {
		labelStyle, fieldStyle := m.styles.formLabelInactive, m.styles.formInactive
		marker := " "
		if i == m.focus {
			labelStyle, fieldStyle, marker = m.styles.formLabel, m.styles.formFocused, ">"
		}
		lines = append(lines, popupLine{text: marker + " " + label, style: labelStyle})
		if i == descriptionField {
			lines = append(lines, popupLine{text: m.description.View(), rawANSI: true})
			continue
		}
		input := m.inputs[i]
		input.TextStyle = fieldStyle
		input.Cursor.TextStyle = fieldStyle
		lines = append(lines, popupLine{text: input.View(), rawANSI: true})
	}
	if m.focus == nameField {
		items := m.filteredSuggestions()
		for i, item := range items {
			if i == 4 {
				break
			}
			marker, style := " ", m.styles.suggestion
			if i == m.suggestionIndex {
				marker, style = ">", m.styles.suggestionSelected
			}
			text := marker + " " + item.Name
			if item.Description != "" {
				text += "  " + item.Description
			}
			lines = append(lines, popupLine{text: text, style: style, optional: true})
		}
	}
	if m.formError != "" {
		lines = append(lines, popupLine{text: "Error: " + m.formError, style: m.styles.error})
	}
	lines = append(lines, popupLine{text: "Tab next · Shift-Tab previous · Enter save · Esc cancel", style: m.styles.footerKey})
	return popup{title: title, preferredWidth: 60, lines: lines}
}

func (m Model) renderCalendar() popup {
	month := time.Date(m.calendar.Year(), m.calendar.Month(), 1, 0, 0, 0, 0, time.Local)
	lines := []popupLine{
		{text: month.Format("January 2006"), style: m.styles.popupTitle},
		{text: " Mon  Tue  Wed  Thu  Fri  Sat  Sun", style: m.styles.calendar},
	}
	offset := (int(month.Weekday()) + 6) % 7
	row := make([]popupSegment, 0, 7)
	for range offset {
		row = append(row, popupSegment{text: "     ", style: m.styles.calendar})
	}
	for day := 1; day <= daysInMonth(month); day++ {
		date := time.Date(month.Year(), month.Month(), day, 0, 0, 0, 0, time.Local)
		value := fmt.Sprintf(" %02d  ", day)
		style := m.styles.calendar
		if m.marks[date.Format(time.DateOnly)] {
			value = fmt.Sprintf(" %02d* ", day)
			style = m.styles.calendarMarked
		}
		if domain.SameDate(date, time.Now()) {
			style = m.styles.calendarToday
		}
		if domain.SameDate(date, m.calendar) {
			value = fmt.Sprintf("[%02d] ", day)
			style = m.styles.calendarSelected
		}
		row = append(row, popupSegment{text: value, style: style})
		if (offset+day)%7 == 0 {
			lines = append(lines, popupLine{segments: row})
			row = nil
		}
	}
	if len(row) > 0 {
		lines = append(lines, popupLine{segments: row})
	}
	lines = append(lines,
		popupLine{text: "hjkl move · PgUp/PgDn month · t today", style: m.styles.confirmation, optional: true},
		popupLine{text: "Enter open · Esc cancel", style: m.styles.footerKey},
	)
	return popup{title: "Calendar", preferredWidth: 40, lines: lines}
}

func (m Model) renderIntrinsic() string {
	entries := m.session.List()
	lines := []string{fmt.Sprintf("ttsh — %s", m.date.Format("Mon 2006-01-02")), ""}
	if len(entries) == 0 {
		lines = append(lines, "No entries today", "n create entry")
	} else {
		for index, entry := range entries {
			marker, end := " ", "..."
			if index == m.selected {
				marker = ">"
			}
			if entry.End != nil {
				end = entry.End.String()
			}
			line := fmt.Sprintf("%s <%s> %s → %s", marker, entry.Name, entry.Start, end)
			if entry.Description != "" {
				line += " · " + entry.Description
			}
			lines = append(lines, line)
		}
	}
	selection := "--"
	if m.selected >= 0 && m.selected < len(entries) {
		selection = fmt.Sprintf("#%d %s", m.selected+1, domain.FormatDuration(m.session.EntryDuration(entries[m.selected])))
	}
	lines = append(lines, "", fmt.Sprintf("%s    %d entries · %s", selection, len(entries), domain.FormatDuration(m.session.CompletedTotal())), "n new   e edit   s stop   c calendar   ? help   q quit")
	if panel, ok := m.renderPopup(); ok {
		lines = append(lines, "", panel.title)
		for _, line := range panel.lines {
			lines = append(lines, line.value())
		}
	}
	return strings.Join(lines, "\n")
}

func daysInMonth(month time.Time) int { return month.AddDate(0, 1, -1).Day() }
