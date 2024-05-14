/*
Copyright © 2024 Syahrul <noturaun@timelang.com>
*/
package cmd

import (
	"dev.noturaun/baymax/http"
	"fmt"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
	"os"
	"strconv"
)

var (
	path     string
	format   string
	checkUrl string
	checkCmd = &cobra.Command{
		Use:   "check",
		Short: "Check dependency vulnerability status",
		Run: func(cmd *cobra.Command, args []string) {
			buff, f := Check(path)
			res := http.Check(buff, f)
			Render(res)
		},
	}
	baseStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color("240"))
)

func init() {
	checkCmd.Flags().StringVar(&path, "path", "", "Dependency path to check")
}

type model struct {
	table table.Model
}

func (m model) Init() tea.Cmd { return nil }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			if m.table.Focused() {
				m.table.Blur()
			} else {
				m.table.Focus()
			}
		case "q", "ctrl+c":
			return m, tea.Quit
		}
	}
	m.table, cmd = m.table.Update(msg)
	return m, cmd
}

func (m model) View() string {
	return baseStyle.Render(m.table.View()) + "\n"
}

func Render(result http.Request) {
	columns := []table.Column{
		{Title: "No", Width: 4},
		{Title: "Group Id", Width: 25},
		{Title: "Artifact Id", Width: 25},
		{Title: "Version", Width: 25},
		{Title: "Status", Width: 25},
	}
	var rows []table.Row
	i := 1
	for _, c := range result.Components {
		if c.ComponentIdentifier.Detection == "block" {
			rows = append(rows, []string{
				strconv.Itoa(i),
				c.ComponentIdentifier.Coordinates.GroupId,
				c.ComponentIdentifier.Coordinates.ArtifactId,
				c.ComponentIdentifier.Coordinates.Version,
				c.ComponentIdentifier.Detection,
			})
			i++
		}
	}

	t := table.New(
		table.WithColumns(columns),
		table.WithRows(rows),
		table.WithFocused(true),
		table.WithHeight(7),
	)

	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")).
		BorderBottom(true).
		Bold(false)
	s.Selected = s.Selected.
		Foreground(lipgloss.Color("229")).
		Background(lipgloss.Color("57")).
		Bold(false)
	t.SetStyles(s)

	m := model{t}
	if _, err := tea.NewProgram(m).Run(); err != nil {
		fmt.Println("Error running program:", err)
		os.Exit(1)
	}
}
