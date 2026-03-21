package ui

import (
	"fmt"

	"charm.land/lipgloss/v2"
)

// SessionRow is a simple struct to decouple from the store package.
type SessionRow struct {
	Status   string // "active", "detached", "archived"
	Name     string
	Cwd      string
	Accessed string // relative time string like "2m ago"
	Tags     string // comma-separated or "-"
}

// RenderSessionTable prints a styled session table to stdout.
func RenderSessionTable(sessions []SessionRow) {
	headerStyle := lipgloss.NewStyle().Faint(true)
	nameStyle := lipgloss.NewStyle().Bold(true)
	dimStyle := lipgloss.NewStyle().Faint(true)

	// Column widths
	const (
		colStatus   = 4
		colName     = 24
		colCwd      = 36
		colAccessed = 12
		colTags     = 16
	)

	// Print header
	fmt.Printf("  %s  %s  %s  %s  %s\n",
		headerStyle.Render(fmt.Sprintf("%-*s", colStatus, "")),
		headerStyle.Render(fmt.Sprintf("%-*s", colName, "NAME")),
		headerStyle.Render(fmt.Sprintf("%-*s", colCwd, "DIRECTORY")),
		headerStyle.Render(fmt.Sprintf("%-*s", colAccessed, "ACCESSED")),
		headerStyle.Render(fmt.Sprintf("%-*s", colTags, "TAGS")),
	)

	// Print rows
	for _, s := range sessions {
		icon := StatusIcon(s.Status)
		name := nameStyle.Render(fmt.Sprintf("%-*s", colName, s.Name))
		cwd := dimStyle.Render(fmt.Sprintf("%-*s", colCwd, s.Cwd))
		accessed := dimStyle.Render(fmt.Sprintf("%-*s", colAccessed, s.Accessed))
		tags := dimStyle.Render(fmt.Sprintf("%-*s", colTags, s.Tags))

		fmt.Printf("  %-*s  %s  %s  %s  %s\n", colStatus, icon, name, cwd, accessed, tags)
	}

	// Print count
	fmt.Println()
	fmt.Printf("  %s\n", dimStyle.Render(fmt.Sprintf("%d session(s)", len(sessions))))
}
