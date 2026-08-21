package tui

import (
	tea "github.com/charmbracelet/bubbletea"
)

// Run 启动 Bubble Tea TUI 终端工作台
func Run(defaultBaseDir string) error {
	p := tea.NewProgram(
		InitialModel(defaultBaseDir),
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)
	_, err := p.Run()
	return err
}
