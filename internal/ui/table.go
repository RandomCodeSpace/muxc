package ui

import (
	"fmt"

	"charm.land/lipgloss/v2"
)

// SessionRow is a simple struct to decouple from the claude package.
type SessionRow struct {
	Status   string // "active" or "detached"
	Name     string
	ShortID  string // first 8 chars of session ID
	Cwd      string
	Accessed string // relative time string like "2m ago"
}

// RenderSessionTable prints a styled session table to stdout.
func RenderSessionTable(sessions []SessionRow) {
	headerStyle := lipgloss.NewStyle().Faint(true)
	nameStyle := lipgloss.NewStyle().Bold(true)
	dimStyle := lipgloss.NewStyle().Faint(true)

	const (
		colStatus   = 4
		colName     = 24
		colID       = 10
		colCwd      = 32
		colAccessed = 12
	)

	fmt.Printf("  %s  %s  %s  %s  %s\n",
		headerStyle.Render(fmt.Sprintf("%-*s", colStatus, "")),
		headerStyle.Render(fmt.Sprintf("%-*s", colName, "NAME")),
		headerStyle.Render(fmt.Sprintf("%-*s", colID, "ID")),
		headerStyle.Render(fmt.Sprintf("%-*s", colCwd, "DIRECTORY")),
		headerStyle.Render(fmt.Sprintf("%-*s", colAccessed, "MODIFIED")),
	)

	for _, s := range sessions {
		icon := StatusIcon(s.Status)
		name := nameStyle.Render(fmt.Sprintf("%-*s", colName, s.Name))
		id := dimStyle.Render(fmt.Sprintf("%-*s", colID, s.ShortID))
		cwd := dimStyle.Render(fmt.Sprintf("%-*s", colCwd, s.Cwd))
		accessed := dimStyle.Render(fmt.Sprintf("%-*s", colAccessed, s.Accessed))

		fmt.Printf("  %-*s  %s  %s  %s  %s\n", colStatus, icon, name, id, cwd, accessed)
	}

	fmt.Println()
	fmt.Printf("  %s\n", dimStyle.Render(fmt.Sprintf("%d session(s)", len(sessions))))
}
