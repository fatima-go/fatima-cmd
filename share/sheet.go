package share

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

// Sheet styling is shared so every list a fatima command shows - inventories
// inside the TUI and the standalone package picker - reads as one table.
var (
	SheetBorder   = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	SheetHeading  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("75"))
	SheetSelected = lipgloss.NewStyle().Bold(true).
			Foreground(lipgloss.AdaptiveColor{Light: "18", Dark: "230"}).
			Background(lipgloss.AdaptiveColor{Light: "153", Dark: "24"})
	SheetAlternate = lipgloss.NewStyle().Background(lipgloss.AdaptiveColor{Light: "254", Dark: "235"})
)

type SheetColumn struct {
	Name  string
	Width int
}

func sheetClean(s string) string {
	s = ansi.Strip(s)
	return strings.Map(func(r rune) rune {
		if r == '\n' || r == '\t' || !unicode.IsControl(r) {
			return r
		}
		return -1
	}, s)
}
func sheetLine(s string, width int) string {
	return ansi.Truncate(strings.ReplaceAll(sheetClean(s), "\n", " "), max(1, width), "…")
}

func SheetCell(value string, width int) string {
	value = sheetLine(strings.ReplaceAll(value, "\t", " "), width)
	return value + strings.Repeat(" ", max(0, width-ansi.StringWidth(value)))
}

// Grid cells use terminal display widths, so Korean names and ANSI input cannot
// move the next column. Only the selected row carries the cursor background.
func SheetGrid(columns []SheetColumn, rows [][]string, selected int) []string {
	rule := func(left, middle, right string) string {
		parts := make([]string, len(columns))
		for i, c := range columns {
			parts[i] = strings.Repeat("─", c.Width+2)
		}
		return SheetBorder.Render(left + strings.Join(parts, middle) + right)
	}
	row := func(values []string, index int) string {
		parts := make([]string, len(columns))
		for i, c := range columns {
			value := ""
			if i < len(values) {
				value = values[i]
			}
			parts[i] = " " + SheetCell(value, c.Width) + " "
			if index >= 0 && index != selected {
				switch value {
				case "ALIVE", "SUCCEEDED":
					parts[i] = lipgloss.NewStyle().Foreground(lipgloss.Color("42")).Render(parts[i])
				case "DEAD", "FAILED", "UNREACHABLE", "MISMATCH":
					parts[i] = lipgloss.NewStyle().Foreground(lipgloss.Color("203")).Render(parts[i])
				case "UNKNOWN", "RUNNING", "WAITING":
					parts[i] = lipgloss.NewStyle().Foreground(lipgloss.Color("214")).Render(parts[i])
				}
			}
		}
		text := "│" + strings.Join(parts, "│") + "│"
		if index < 0 {
			return SheetHeading.Render(text)
		}
		if index == selected {
			return SheetSelected.Render(text)
		}
		if index%2 == 1 {
			return SheetAlternate.Render(text)
		}
		return text
	}
	headers := make([]string, len(columns))
	for i, c := range columns {
		headers[i] = c.Name
	}
	out := []string{rule("┌", "┬", "┐"), row(headers, -1), rule("├", "┼", "┤")}
	for i, values := range rows {
		out = append(out, row(values, i))
	}
	return append(out, rule("└", "┴", "┘"))
}

func SheetTitle(title string, width int, focused bool) string {
	mark := "○ "
	if focused {
		mark = "● "
	}
	return SheetHeading.Render(SheetCell(mark+title, width))
}

func SheetView(title string, columns []SheetColumn, rows [][]string, cursor, width, height int, focused bool, footer string) string {
	room := max(1, height-6) // title, grid header/borders, footer
	start := min(max(0, cursor-room+1), max(0, len(rows)-room))
	end := min(len(rows), start+room)
	window := append([][]string(nil), rows[start:end]...)
	for len(window) < room {
		window = append(window, nil)
	}
	selected := -1
	if cursor >= start && cursor < end {
		selected = cursor - start
	}
	out := []string{SheetTitle(title, width, focused)}
	out = append(out, SheetGrid(columns, window, selected)...)
	position := "0 / 0"
	if len(rows) > 0 {
		position = fmt.Sprintf("%d–%d / %d", start+1, end, len(rows))
	}
	return strings.Join(append(out, sheetLine(position+"  "+footer, width)), "\n")
}
