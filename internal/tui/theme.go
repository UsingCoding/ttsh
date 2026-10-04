package tui

import "github.com/charmbracelet/lipgloss"

// Theme assigns semantic colors to the TUI. Renderers use styles rather than
// palette literals so visual changes remain isolated here.
type Theme struct {
	Background       lipgloss.Color
	Panel            lipgloss.Color
	Foreground       lipgloss.Color
	Muted            lipgloss.Color
	Border           lipgloss.Color
	Accent           lipgloss.Color
	Cursor           lipgloss.Color
	Selected         lipgloss.Color
	StatusBar        lipgloss.Color
	StatusText       lipgloss.Color
	EntryName        lipgloss.Color
	EntryStart       lipgloss.Color
	EntryEnd         lipgloss.Color
	EntryDescription lipgloss.Color
	Running          lipgloss.Color
	Error            lipgloss.Color
}

type styles struct {
	canvas             lipgloss.Style
	header             lipgloss.Style
	headerTitle        lipgloss.Style
	date               lipgloss.Style
	divider            lipgloss.Style
	entry              lipgloss.Style
	entrySelected      lipgloss.Style
	entryName          lipgloss.Style
	entryStart         lipgloss.Style
	entryEnd           lipgloss.Style
	entryDescription   lipgloss.Style
	entrySelectedName  lipgloss.Style
	entrySelectedStart lipgloss.Style
	entrySelectedEnd   lipgloss.Style
	entrySelectedDesc  lipgloss.Style
	running            lipgloss.Style
	runningSelected    lipgloss.Style
	cursor             lipgloss.Style
	status             lipgloss.Style
	statusText         lipgloss.Style
	footerKey          lipgloss.Style
	footerLabel        lipgloss.Style
	popup              lipgloss.Style
	popupTitle         lipgloss.Style
	popupBorder        lipgloss.Style
	formLabel          lipgloss.Style
	formLabelInactive  lipgloss.Style
	formFocused        lipgloss.Style
	formInactive       lipgloss.Style
	suggestion         lipgloss.Style
	suggestionSelected lipgloss.Style
	calendar           lipgloss.Style
	calendarSelected   lipgloss.Style
	calendarToday      lipgloss.Style
	calendarMarked     lipgloss.Style
	confirmation       lipgloss.Style
	error              lipgloss.Style
	empty              lipgloss.Style
}

func defaultMutedSlateTheme() Theme {
	return Theme{
		Background:       lipgloss.Color("#14212F"),
		Panel:            lipgloss.Color("#1D2C3D"),
		Foreground:       lipgloss.Color("#E7EDF0"),
		Muted:            lipgloss.Color("#8A9AA7"),
		Border:           lipgloss.Color("#536878"),
		Accent:           lipgloss.Color("#58B7D3"),
		Cursor:           lipgloss.Color("#80D5E8"),
		Selected:         lipgloss.Color("#273B4D"),
		StatusBar:        lipgloss.Color("#213345"),
		StatusText:       lipgloss.Color("#C9D8DF"),
		EntryName:        lipgloss.Color("#F0F4F5"),
		EntryStart:       lipgloss.Color("#AFC3CD"),
		EntryEnd:         lipgloss.Color("#B8CBD4"),
		EntryDescription: lipgloss.Color("#91A3B0"),
		Running:          lipgloss.Color("#78C6B5"),
		Error:            lipgloss.Color("#D98787"),
	}
}

func newStyles(theme Theme) styles {
	base := lipgloss.NewStyle().Background(theme.Background).Foreground(theme.Foreground)
	selected := base.Background(theme.Selected)
	return styles{
		canvas:             base,
		header:             base,
		headerTitle:        base.Foreground(theme.Accent).Bold(true),
		date:               base.Foreground(theme.Muted),
		divider:            base.Foreground(theme.Border),
		entry:              base,
		entrySelected:      selected,
		entryName:          base.Foreground(theme.EntryName),
		entryStart:         base.Foreground(theme.EntryStart),
		entryEnd:           base.Foreground(theme.EntryEnd),
		entryDescription:   base.Foreground(theme.EntryDescription),
		entrySelectedName:  selected.Foreground(theme.EntryName).Bold(true),
		entrySelectedStart: selected.Foreground(theme.EntryStart),
		entrySelectedEnd:   selected.Foreground(theme.EntryEnd),
		entrySelectedDesc:  selected.Foreground(theme.EntryDescription),
		running:            base.Foreground(theme.Running),
		runningSelected:    selected.Foreground(theme.Running),
		cursor:             selected.Foreground(theme.Cursor).Bold(true),
		status:             base.Background(theme.StatusBar),
		statusText:         base.Background(theme.StatusBar).Foreground(theme.StatusText),
		footerKey:          base.Background(theme.StatusBar).Foreground(theme.Accent).Bold(true),
		footerLabel:        base.Background(theme.StatusBar).Foreground(theme.Muted),
		popup:              base.Background(theme.Panel).Foreground(theme.Foreground),
		popupTitle:         base.Background(theme.Panel).Foreground(theme.Accent).Bold(true),
		popupBorder:        base.Background(theme.Panel).Foreground(theme.Border),
		formLabel:          base.Background(theme.Panel).Foreground(theme.Accent),
		formLabelInactive:  base.Background(theme.Panel).Foreground(theme.Muted),
		formFocused:        base.Background(theme.Selected).Foreground(theme.Foreground),
		formInactive:       base.Background(theme.Panel).Foreground(theme.Foreground),
		suggestion:         base.Background(theme.Panel).Foreground(theme.Muted),
		suggestionSelected: base.Background(theme.Selected).Foreground(theme.Foreground),
		calendar:           base.Background(theme.Panel).Foreground(theme.Foreground),
		calendarSelected:   base.Background(theme.Selected).Foreground(theme.Cursor).Bold(true),
		calendarToday:      base.Background(theme.Panel).Foreground(theme.Accent).Underline(true),
		calendarMarked:     base.Background(theme.Panel).Foreground(theme.Running),
		confirmation:       base.Background(theme.Panel).Foreground(theme.Foreground),
		error:              base.Background(theme.Panel).Foreground(theme.Error),
		empty:              base.Foreground(theme.Muted),
	}
}
