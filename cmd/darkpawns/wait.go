package main

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
)

type (
	phaseView struct {
		name, detail          string
		fraction              float64
		complete, skip, known bool
	}
	operationDone struct{ err error }
	operationTick time.Time
	waitModel     struct {
		title                string
		spinner              spinner.Model
		bar                  progress.Model
		phases               func() []phaseView
		done                 <-chan struct{}
		result               func() error
		cancel               context.CancelFunc
		finished, cancelling bool
		err                  error
		width                int
	}
)

func (m waitModel) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, func() tea.Msg { <-m.done; return operationDone{m.result()} }, tea.Tick(150*time.Millisecond, func(t time.Time) tea.Msg { return operationTick(t) }))
}

func (m waitModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = max(20, msg.Width)
		m.bar.Width = max(8, min(40, m.width-24))
	case operationDone:
		m.finished = true
		m.err = msg.err
		return m, tea.Quit
	case operationTick:
		return m, tea.Tick(150*time.Millisecond, func(t time.Time) tea.Msg { return operationTick(t) })
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" && !m.cancelling {
			m.cancelling = true
			m.cancel()
		}
	}
	var cmd tea.Cmd
	m.spinner, cmd = m.spinner.Update(msg)
	return m, cmd
}

func (m waitModel) View() string {
	s := cliTitle.Render(terminalText(m.title)) + "\n\n"
	detail := "Waiting for the operation"
	if m.finished {
		detail = "Operation returned"
	}
	phases := []phaseView{{name: "Working", detail: detail}}
	if m.phases != nil {
		phases = m.phases()
	}
	for _, p := range phases {
		marker := m.spinner.View()
		if p.complete {
			marker = "✓"
		}
		if p.skip {
			marker = "·"
		}
		if m.finished && !p.complete && !p.skip {
			marker = "·"
			if m.err == nil && m.phases == nil {
				marker = "✓"
			}
		}
		s += fmt.Sprintf("%s %s\n", marker, lipgloss.NewStyle().Bold(true).Render(p.name))
		if m.phases != nil {
			ratio := p.fraction
			if p.complete {
				ratio = 1
			}
			bar := m.bar
			bar.ShowPercentage = p.known || p.complete
			s += "  " + bar.ViewAs(ratio) + "\n"
		}
		s += lipgloss.NewStyle().Width(max(16, m.width-4)).PaddingLeft(2).Render(terminalText(p.detail)) + "\n"
	}
	if m.cancelling && !m.finished {
		s += "\nCancelling. Waiting for the operation and recovery to finish.\n"
	} else if !m.finished {
		s += "\nCtrl+C cancel · waiting for confirmed results\n"
	}
	if m.err != nil {
		s += "\n" + cliRule.Render(terminalText(m.err.Error())) + "\n"
	}
	return lipgloss.NewStyle().Padding(1, 2).Render(strings.TrimSuffix(s, "\n")) + "\n"
}

// The task starts independently of terminal initialization. Even if rendering
// fails or is interrupted, do not return (and close an update plan) until its
// operation has actually stopped, including portable's recovery path.
func runWait[T any](cmd *cobra.Command, title string, phases func() []phaseView, work func(context.Context) (T, error)) (T, error) {
	ctx, cancel := context.WithCancel(cmd.Context())
	defer cancel()
	var value T
	var workErr error
	done := make(chan struct{})
	go func() { defer close(done); value, workErr = work(ctx) }()
	sp := spinner.New(spinner.WithSpinner(spinner.Dot))
	sp.Style = cliRule
	model := waitModel{title: title, spinner: sp, bar: progress.New(progress.WithSolidFill("#A8201A"), progress.WithWidth(36)), phases: phases, done: done, result: func() error { return workErr }, cancel: cancel, width: cliWidth(cmd)}
	_, uiErr := tea.NewProgram(model, tea.WithInput(cmd.InOrStdin()), tea.WithOutput(cmd.OutOrStdout()), tea.WithoutSignalHandler()).Run()
	if uiErr != nil {
		cancel()
	}
	<-done
	return value, errors.Join(workErr, uiErr, ctx.Err())
}
