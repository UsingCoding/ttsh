package tui

import (
	"context"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/require"
	"github.com/usingcoding/ttsh/internal/app"
	"github.com/usingcoding/ttsh/internal/domain"
	"github.com/usingcoding/ttsh/internal/storage"
	"github.com/usingcoding/ttsh/internal/suggestions"
)

func key(value string) tea.KeyMsg { return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(value)} }

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
