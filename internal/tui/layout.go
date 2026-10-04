package tui

import (
	"strings"
	"unicode/utf8"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

const (
	headerRows = 1
	footerRows = 2
	bodyInset  = 2
	maxContent = 96
)

type viewport struct {
	width        int
	height       int
	bodyTop      int
	bodyBottom   int
	contentX     int
	contentWidth int
}

func newViewport(width, height int) (viewport, bool) {
	if width <= 0 || height <= 0 {
		return viewport{}, false
	}
	contentWidth := width - bodyInset*2
	if contentWidth < 1 {
		contentWidth = 1
	}
	if contentWidth > maxContent {
		contentWidth = maxContent
	}
	return viewport{
		width: width, height: height, bodyTop: headerRows,
		bodyBottom: height - footerRows, contentX: min(bodyInset, max(width-1, 0)),
		contentWidth: contentWidth,
	}, true
}

func (v viewport) bodyHeight() int { return max(v.bodyBottom-v.bodyTop, 0) }

type canvasCell struct {
	text         string
	style        lipgloss.Style
	continuation bool
}

type canvas struct {
	width  int
	height int
	cells  []canvasCell
}

func newCanvas(width, height int, background lipgloss.Style) *canvas {
	cells := make([]canvasCell, width*height)
	for i := range cells {
		cells[i] = canvasCell{text: " ", style: background}
	}
	return &canvas{width: width, height: height, cells: cells}
}

func (c *canvas) cell(x, y int) *canvasCell {
	if x < 0 || x >= c.width || y < 0 || y >= c.height {
		return nil
	}
	return &c.cells[y*c.width+x]
}

func (c *canvas) fill(x, y, width, height int, style lipgloss.Style) {
	for row := max(y, 0); row < min(y+height, c.height); row++ {
		for column := max(x, 0); column < min(x+width, c.width); column++ {
			*c.cell(column, row) = canvasCell{text: " ", style: style}
		}
	}
}

func (c *canvas) write(x, y int, value string, style lipgloss.Style) {
	if y < 0 || y >= c.height || x >= c.width {
		return
	}
	if x < 0 {
		value = ansi.Cut(value, -x, ansi.StringWidth(value))
		x = 0
	}
	value = ansi.Truncate(value, c.width-x, "")
	for len(value) > 0 && x < c.width {
		r, size := utf8.DecodeRuneInString(value)
		if r == utf8.RuneError && size == 0 {
			break
		}
		cluster := value[:size]
		value = value[size:]
		width := ansi.StringWidth(cluster)
		if width == 0 {
			continue
		}
		if width > c.width-x {
			break
		}
		*c.cell(x, y) = canvasCell{text: cluster, style: style}
		for offset := 1; offset < width; offset++ {
			*c.cell(x+offset, y) = canvasCell{style: style, continuation: true}
		}
		x += width
	}
}

func (c *canvas) render() string {
	lines := make([]string, c.height)
	for y := 0; y < c.height; y++ {
		var line strings.Builder
		for x := 0; x < c.width; x++ {
			cell := c.cells[y*c.width+x]
			if !cell.continuation {
				line.WriteString(cell.style.Render(cell.text))
			}
		}
		lines[y] = line.String()
	}
	return strings.Join(lines, "\n")
}

func clipped(value string, width int) string {
	if width <= 0 {
		return ""
	}
	return ansi.Truncate(value, width, "…")
}

type entryWindow struct {
	start int
	end   int
	above bool
	below bool
}

func visibleEntries(count, selected, rows int) entryWindow {
	if count == 0 || rows <= 0 {
		return entryWindow{}
	}
	if count <= rows {
		return entryWindow{end: count}
	}
	if rows == 1 {
		return entryWindow{start: min(max(selected, 0), count-1), end: min(max(selected, 0), count-1) + 1, above: selected > 0, below: selected < count-1}
	}
	// Retain room for explicit omitted-entry indicators above and below the
	// contiguous window. The selected row remains in the data window.
	capacity := max(rows-2, 1)
	start := min(max(selected-capacity/2, 0), count-capacity)
	end := start + capacity
	return entryWindow{start: start, end: end, above: start > 0, below: end < count}
}

type popupSegment struct {
	text  string
	style lipgloss.Style
}

type popupLine struct {
	text     string
	style    lipgloss.Style
	segments []popupSegment
	optional bool
}

type popup struct {
	title          string
	preferredWidth int
	lines          []popupLine
}

func (c *canvas) popup(v viewport, panel popup, s styles) {
	if c.width == 0 || c.height == 0 {
		return
	}
	maxWidth := max(v.width-4, 1)
	width := min(panel.preferredWidth, maxWidth)
	width = min(width, v.width)
	if width < 1 {
		return
	}
	maxHeight := max(v.height-4, 1)
	contentLimit := maxHeight - 2
	lines := append([]popupLine(nil), panel.lines...)
	for popupContentHeight(lines, width) > contentLimit {
		removed := false
		for i := len(lines) - 1; i >= 0; i-- {
			if lines[i].optional {
				lines = append(lines[:i], lines[i+1:]...)
				removed = true
				break
			}
		}
		if !removed {
			break
		}
	}
	contentHeight := min(popupContentHeight(lines, width), contentLimit)
	height := min(contentHeight+2, maxHeight)
	height = min(height, v.height)
	x := max((v.width-width)/2, 0)
	y := max((v.height-height)/2, 0)
	c.fill(x, y, width, height, s.popup)
	if width >= 2 && height >= 2 {
		c.write(x, y, "╭"+strings.Repeat("─", max(width-2, 0))+"╮", s.popupBorder)
		c.write(x, y+height-1, "╰"+strings.Repeat("─", max(width-2, 0))+"╯", s.popupBorder)
		for row := y + 1; row < y+height-1; row++ {
			c.write(x, row, "│", s.popupBorder)
			c.write(x+width-1, row, "│", s.popupBorder)
		}
	}
	innerWidth := max(width-2, 1)
	row := y + 1
	if row < y+height-1 {
		c.write(x+1, row, clipped(panel.title, innerWidth), s.popupTitle)
		row++
	}
	for _, line := range lines {
		if row >= y+height-1 {
			break
		}
		value := line.value()
		if len(line.segments) > 0 && ansi.StringWidth(value) <= innerWidth {
			column := x + 1
			for _, segment := range line.segments {
				c.write(column, row, segment.text, segment.style)
				column += ansi.StringWidth(segment.text)
			}
			row++
			continue
		}
		for _, segment := range wrapLine(value, innerWidth) {
			if row >= y+height-1 {
				break
			}
			c.write(x+1, row, segment, line.style)
			row++
		}
	}
}

func (line popupLine) value() string {
	if len(line.segments) == 0 {
		return line.text
	}
	var value strings.Builder
	for _, segment := range line.segments {
		value.WriteString(segment.text)
	}
	return value.String()
}

func popupContentHeight(lines []popupLine, width int) int {
	if width <= 2 {
		return 1
	}
	height := 1 // title
	for _, line := range lines {
		height += len(wrapLine(line.value(), width-2))
	}
	return height
}

func wrapLine(value string, width int) []string {
	if value == "" {
		return []string{""}
	}
	if width <= 0 {
		return nil
	}
	var lines []string
	for ansi.StringWidth(value) > width {
		part := ansi.Truncate(value, width, "")
		if part == "" {
			break
		}
		lines = append(lines, part)
		value = ansi.Cut(value, ansi.StringWidth(part), ansi.StringWidth(value))
	}
	return append(lines, value)
}

func min(left, right int) int {
	if left < right {
		return left
	}
	return right
}

func max(left, right int) int {
	if left > right {
		return left
	}
	return right
}
