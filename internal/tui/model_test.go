package tui

import (
	"context"
	"errors"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/stretchr/testify/require"
	"github.com/usingcoding/ttsh/internal/app"
	"github.com/usingcoding/ttsh/internal/domain"
	"github.com/usingcoding/ttsh/internal/storage"
	"github.com/usingcoding/ttsh/internal/suggestions"
)

func key(value string) tea.KeyMsg { return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(value)} }

type structuredProvider struct {
	items []suggestions.Suggestion
}

func (p structuredProvider) Suggestions(context.Context) []suggestions.Suggestion {
	return p.items
}

func TestModelStructuredSuggestionDisplaysAndAcceptsOnlyName(t *testing.T) {
	service := app.New(storage.New(storage.Paths{SheetsDir: t.TempDir()}), nil, false)
	model, err := New(service, structuredProvider{items: []suggestions.Suggestion{{
		Name: "YT-183", Description: "Migration to new db",
	}}}, nil)
	require.NoError(t, err)
	defer func() { require.NoError(t, model.session.Close()) }()

	model = resizeModel(t, model, 100, 32)
	updated, _ := model.Update(key("n"))
	model = updated.(Model)
	view := model.View()
	require.Contains(t, view, "YT-183")
	require.Contains(t, view, "Migration to new db")

	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyTab})
	model = updated.(Model)
	require.Equal(t, "YT-183", model.inputs[0].Value())
	require.Empty(t, model.inputs[3].Value())

	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = updated.(Model)
	require.Equal(t, modeSheet, model.mode)
	entries := model.session.List()
	require.Len(t, entries, 1)
	require.Equal(t, "YT-183", entries[0].Name)
	require.Empty(t, entries[0].Description)
}

func TestModelNavigationFormsAndConfirmation(t *testing.T) {
	service := app.New(storage.New(storage.Paths{SheetsDir: t.TempDir()}), nil, false)
	now := time.Now()
	session, err := service.Open(context.Background(), now)
	require.NoError(t, err)
	start := domain.TimeOfDayAt(now)
	_, _, err = session.Add(domain.EntryInput{Name: "Active", Start: start}, app.RejectActive)
	require.NoError(t, err)
	require.NoError(t, session.Close())

	model, err := New(service, suggestions.None{}, nil)
	require.NoError(t, err)
	defer func() { _ = model.session.Close() }()
	require.Equal(t, 0, model.selected)
	updated, _ := model.Update(key("j"))
	model = updated.(Model)
	require.Equal(t, 0, model.selected)
	updated, _ = model.Update(key("n"))
	model = updated.(Model)
	require.Equal(t, modeNew, model.mode)
	require.NotEmpty(t, model.inputs[1].Value())
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = updated.(Model)
	require.Equal(t, modeNew, model.mode)
	require.Contains(t, model.formError, "name is required")
	model.inputs[0].SetValue("Next")
	model.submitForm()
	require.Equal(t, modeConfirm, model.mode)
	updated, _ = model.Update(key("n"))
	model = updated.(Model)
	require.Equal(t, modeNew, model.mode)
	require.Equal(t, "Next", model.inputs[0].Value())
	model.submitForm()
	require.Equal(t, modeConfirm, model.mode)
	updated, _ = model.Update(key("y"))
	model = updated.(Model)
	require.Equal(t, modeSheet, model.mode)
	require.Equal(t, 1, model.selected)
	require.Len(t, model.session.List(), 2)
	updated, _ = model.Update(key("n"))
	model = updated.(Model)
	require.Equal(t, modeNew, model.mode)
	updated, _ = model.Update(key("esc"))
	model = updated.(Model)
	require.Equal(t, modeSheet, model.mode)
	updated, _ = model.Update(key("?"))
	model = updated.(Model)
	require.Equal(t, modeHelp, model.mode)
	updated, _ = model.Update(key("j"))
	model = updated.(Model)
	require.Equal(t, modeHelp, model.mode)
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyEsc})
	model = updated.(Model)
	require.Equal(t, modeSheet, model.mode)
	updated, _ = model.Update(key("c"))
	model = updated.(Model)
	require.Equal(t, modeCalendar, model.mode)
	updated, _ = model.Update(key("l"))
	model = updated.(Model)
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = updated.(Model)
	require.Equal(t, modeSheet, model.mode)
	require.Equal(t, domain.DateOnly(now).AddDate(0, 0, 1), model.date)
}

type fixedClock struct{ now time.Time }

func (clock fixedClock) Now() time.Time { return clock.now }

func TestModelViewUsesViewportAndPreservesStateMarkers(t *testing.T) {
	now := domain.DateOnly(time.Now()).Add(14*time.Hour + 30*time.Minute)
	model := modelWithCompletedAndActiveEntries(t, now)
	defer func() { require.NoError(t, model.session.Close()) }()

	model = resizeModel(t, model, 80, 24)
	view := model.View()
	require.Equal(t, 80, lipgloss.Width(view))
	require.Equal(t, 24, lipgloss.Height(view))
	require.Contains(t, view, ">")
	require.Contains(t, view, "...")
	require.Contains(t, view, "#1 1h:30m")
	for _, key := range []string{"n", "e", "s", "c", "?", "q"} {
		require.Contains(t, view, key)
	}

	model = resizeModel(t, model, 124, 42)
	view = model.View()
	require.Equal(t, 124, lipgloss.Width(view))
	require.Equal(t, 42, lipgloss.Height(view))
	require.Contains(t, view, "<Completed>")
	require.Contains(t, view, "Active")
}

func TestModelPopupViewsRemainWithinViewportAndKeepSheetVisible(t *testing.T) {
	now := domain.DateOnly(time.Now()).Add(14*time.Hour + 30*time.Minute)
	model := resizeModel(t, modelWithCompletedAndActiveEntries(t, now), 80, 24)
	defer func() { require.NoError(t, model.session.Close()) }()

	assertPopup := func(name string) {
		t.Helper()
		view := model.View()
		require.Equalf(t, 80, lipgloss.Width(view), "%s width", name)
		require.Equalf(t, 24, lipgloss.Height(view), "%s height", name)
		require.Containsf(t, view, "ttsh", "%s retains base header", name)
		require.Containsf(t, view, "#1 1h:30m", "%s retains base footer", name)
	}

	updated, _ := model.Update(key("n"))
	model = updated.(Model)
	assertPopup("new")
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyEsc})
	model = updated.(Model)

	updated, _ = model.Update(key("e"))
	model = updated.(Model)
	assertPopup("edit")
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyEsc})
	model = updated.(Model)

	updated, _ = model.Update(key("c"))
	model = updated.(Model)
	assertPopup("calendar")
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyEsc})
	model = updated.(Model)

	updated, _ = model.Update(key("?"))
	model = updated.(Model)
	assertPopup("help")
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyEsc})
	model = updated.(Model)

	model.mode = modeConfirm
	model.errorText = "Another entry is active"
	assertPopup("confirm")

	model.showError(errors.New("save failed"))
	assertPopup("error")
}

func modelWithCompletedAndActiveEntries(t *testing.T, now time.Time) Model {
	t.Helper()
	service := app.New(storage.New(storage.Paths{SheetsDir: t.TempDir()}), fixedClock{now: now}, false)
	session, err := service.Open(context.Background(), now)
	require.NoError(t, err)
	completedEnd, err := domain.ParseTimeOfDay("11:30")
	require.NoError(t, err)
	_, _, err = session.Add(domain.EntryInput{
		Name: "Completed", Start: domain.TimeOfDay(10 * 60), End: &completedEnd, Description: "finished work",
	}, app.RejectActive)
	require.NoError(t, err)
	_, _, err = session.Add(domain.EntryInput{Name: "Active", Start: domain.TimeOfDay(13 * 60)}, app.RejectActive)
	require.NoError(t, err)
	require.NoError(t, session.Close())
	model, err := New(service, suggestions.None{}, nil)
	require.NoError(t, err)
	return model
}

func resizeModel(t *testing.T, model Model, width, height int) Model {
	t.Helper()
	updated, _ := model.Update(tea.WindowSizeMsg{Width: width, Height: height})
	return updated.(Model)
}
