// Package render turns a preflight report into output. Every renderer consumes the same
// []Finding, so adding a presentation never requires changing a check.
package render

import (
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"dbz-mage/ko/preflight"
)

var (
	styleTitle  = lipgloss.NewStyle().Bold(true)
	styleMuted  = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "244", Dark: "245"})
	styleLabel  = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "62", Dark: "111"})
	styleRemedy = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "28", Dark: "78"})

	statusStyles = map[string]lipgloss.Style{
		"passed":  lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "28", Dark: "78"}),
		"note":    lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "62", Dark: "111"}),
		"warning": lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "130", Dark: "214"}),
		"blocker": lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "160", Dark: "203"}),
		"skipped": lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "244", Dark: "245"}),
	}

	statusGlyphs = map[string]string{
		"passed":  "✓",
		"note":    "•",
		"warning": "!",
		"blocker": "✗",
		"skipped": "?",
	}
)

// Text writes a human-readable report. Line-oriented on purpose: the output is meant to be
// greppable, pipeable, and pasteable into a bug report, which a full-screen view is not.
func Text(w io.Writer, report preflight.Report) error {
	var b strings.Builder

	b.WriteString(styleTitle.Render("dmp-lab doctor") + "\n\n")

	// The check ID is a machine identifier. Showing it as the loudest thing on the line
	// puts jargon in front of a reader who may not know Kubernetes, for no benefit — it is
	// still carried by --json and by the documentation link.
	width := titleWidth(report)

	for _, f := range report.Findings {
		status := f.Status.String()
		style := statusStyles[status]
		glyph := statusGlyphs[status]

		fmt.Fprintf(&b, "%s %s  %s\n",
			style.Render(glyph),
			styleLabel.Render(pad(f.Title, width)),
			f.Summary,
		)

		// Verbatim, before the prose: an enumeration is usually the substance of the finding,
		// and re-wrapping it would break the alignment that makes it scannable.
		for _, item := range f.Items {
			b.WriteString("    " + item + "\n")
		}

		if f.Detail != "" {
			for _, line := range wrap(f.Detail, 76) {
				b.WriteString("    " + styleMuted.Render(line) + "\n")
			}
		}

		for _, cmd := range f.Remedy {
			b.WriteString("    " + styleRemedy.Render(cmd) + "\n")
		}

		// Only the link a check opted into. The troubleshooting URL derived from every ID
		// stays in --json rather than being printed by default: each finding already carries
		// its own fix, so an unconditional link promises more help than it delivers.
		if f.Link != "" {
			b.WriteString("    " + styleMuted.Render(f.Link) + "\n")
		}

		b.WriteString("\n")
	}

	b.WriteString(summaryLine(report) + "\n")

	_, err := io.WriteString(w, b.String())
	return err
}

// summaryOrder lists statuses worst-first, so the tally reads in the order the operator
// cares about rather than alphabetically.
var summaryOrder = []preflight.Status{
	preflight.StatusFail,
	preflight.StatusWarn,
	preflight.StatusInfo,
	preflight.StatusPass,
	preflight.StatusSkip,
}

func summaryLine(report preflight.Report) string {
	counts := report.Counts()

	parts := make([]string, 0, len(counts))
	for _, status := range summaryOrder {
		name := status.String()
		count := counts[name]
		if count == 0 {
			continue
		}
		parts = append(parts, statusStyles[name].Render(fmt.Sprintf("%d %s", count, displayName(status, count))))
	}

	if len(parts) == 0 {
		return styleMuted.Render("no checks ran")
	}
	return strings.Join(parts, styleMuted.Render(" · "))
}

// displayName pluralises a status name for a count. "passed" and "skipped" are participles
// and are already correct for any number; only the nouns take an "s".
func displayName(s preflight.Status, count int) string {
	name := s.String()
	if count == 1 {
		return name
	}
	switch s {
	case preflight.StatusFail, preflight.StatusWarn, preflight.StatusInfo:
		return name + "s"
	default:
		return name
	}
}

func titleWidth(report preflight.Report) int {
	width := 0
	for _, f := range report.Findings {
		if n := len([]rune(f.Title)); n > width {
			width = n
		}
	}
	return width
}

func pad(s string, width int) string {
	if n := len([]rune(s)); n < width {
		return s + strings.Repeat(" ", width-n)
	}
	return s
}

func wrap(s string, width int) []string {
	words := strings.Fields(s)
	if len(words) == 0 {
		return nil
	}

	var (
		lines []string
		line  strings.Builder
	)
	for _, word := range words {
		if line.Len() > 0 && line.Len()+1+len(word) > width {
			lines = append(lines, line.String())
			line.Reset()
		}
		if line.Len() > 0 {
			line.WriteByte(' ')
		}
		line.WriteString(word)
	}
	if line.Len() > 0 {
		lines = append(lines, line.String())
	}
	return lines
}
