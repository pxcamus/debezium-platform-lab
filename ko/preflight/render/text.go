// Package render turns a preflight report into output. Every renderer consumes the same
// []Finding, so adding a presentation never requires changing a check.
package render

import (
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"dbz-mage/ko/preflight"
)

var (
	styleTitle  = lipgloss.NewStyle().Bold(true)
	styleMuted  = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "244", Dark: "245"})
	styleID     = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "62", Dark: "111"})
	styleRemedy = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "28", Dark: "78"})

	statusStyles = map[string]lipgloss.Style{
		"pass": lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "28", Dark: "78"}),
		"info": lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "62", Dark: "111"}),
		"warn": lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "130", Dark: "214"}),
		"fail": lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "160", Dark: "203"}),
		"skip": lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "244", Dark: "245"}),
	}

	statusGlyphs = map[string]string{
		"pass": "✓",
		"info": "•",
		"warn": "!",
		"fail": "✗",
		"skip": "?",
	}
)

// Text writes a human-readable report. Line-oriented on purpose: the output is meant to be
// greppable, pipeable, and pasteable into a bug report, which a full-screen view is not.
func Text(w io.Writer, report preflight.Report) error {
	var b strings.Builder

	b.WriteString(styleTitle.Render("dmp-lab doctor") + "\n\n")

	for _, f := range report.Findings {
		status := f.Status.String()
		style := statusStyles[status]
		glyph := statusGlyphs[status]

		fmt.Fprintf(&b, "%s %s  %s\n",
			style.Render(glyph),
			styleID.Render(f.ID),
			f.Summary,
		)

		if f.Detail != "" {
			for _, line := range wrap(f.Detail, 76) {
				b.WriteString("    " + styleMuted.Render(line) + "\n")
			}
		}

		for _, cmd := range f.Remedy {
			b.WriteString("    " + styleRemedy.Render(cmd) + "\n")
		}

		if f.Status != preflight.StatusPass {
			b.WriteString("    " + styleMuted.Render(f.DocURL) + "\n")
		}

		b.WriteString("\n")
	}

	b.WriteString(summaryLine(report) + "\n")

	_, err := io.WriteString(w, b.String())
	return err
}

func summaryLine(report preflight.Report) string {
	counts := report.Counts()

	keys := make([]string, 0, len(counts))
	for k := range counts {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, statusStyles[k].Render(fmt.Sprintf("%d %s", counts[k], k)))
	}

	if len(parts) == 0 {
		return styleMuted.Render("no checks ran")
	}
	return strings.Join(parts, styleMuted.Render(" · "))
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
