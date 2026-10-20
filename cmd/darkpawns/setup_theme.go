package main

import (
	"fmt"
	"io"

	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
)

// The terminal keeps the operator's font and background. Only PAWNS and active
// controls take the site's oxblood accent; normal text keeps terminal contrast.
var setupAccent = lipgloss.AdaptiveColor{Light: "#A8201A", Dark: "#CF4B42"}

func setupWordmark() string {
	dark := ` ____    _    ____  _  __
|  _ \  / \  |  _ \| |/ /
| | | |/ _ \ | |_) | ' /
| |_| / ___ \|  _ <| . \
|____/_/   \_\_| \_\_|\_\`
	pawns := ` ____   _ __        ___   _ ____
|  _ \ / \\ \      / / \ | / ___|
| |_) / _ \\ \ /\ / /|  \| \___ \
|  __/ ___ \\ V  V / | |\  |___) |
|_| /_/   \_\\_/\_/  |_| \_|____/`
	return dark + "\n" + lipgloss.NewStyle().Foreground(setupAccent).Render(pawns)
}

func setupTheme() *huh.Theme {
	t := huh.ThemeBase()
	t.FieldSeparator = lipgloss.NewStyle().SetString("\n")
	t.Group.Title = lipgloss.NewStyle().Bold(true).MarginBottom(1)
	t.Group.Description = lipgloss.NewStyle().MarginBottom(1)
	t.Focused.Base = lipgloss.NewStyle().BorderStyle(lipgloss.NormalBorder()).BorderLeft(true).BorderForeground(setupAccent).PaddingLeft(1)
	t.Focused.Title = lipgloss.NewStyle().Bold(true)
	t.Focused.ErrorIndicator = t.Focused.ErrorIndicator.Foreground(setupAccent)
	t.Focused.ErrorMessage = t.Focused.ErrorMessage.Foreground(setupAccent)
	t.Focused.TextInput.Prompt = lipgloss.NewStyle().Foreground(setupAccent)
	t.Focused.TextInput.Placeholder = lipgloss.NewStyle()
	t.Focused.FocusedButton = lipgloss.NewStyle().Foreground(lipgloss.Color("#EFE7D6")).Background(lipgloss.Color("#A8201A")).Padding(0, 2).MarginRight(1)
	t.Focused.BlurredButton = lipgloss.NewStyle().Padding(0, 2).MarginRight(1)
	t.Blurred = t.Focused
	t.Blurred.Base = t.Focused.Base.BorderStyle(lipgloss.HiddenBorder())
	// Navigation is essential text, including on terminals with unusual palettes.
	t.Help.ShortKey = t.Help.ShortKey.UnsetForeground()
	t.Help.ShortDesc = t.Help.ShortDesc.UnsetForeground()
	t.Help.ShortSeparator = t.Help.ShortSeparator.UnsetForeground()
	t.Help.FullKey = t.Help.FullKey.UnsetForeground()
	t.Help.FullDesc = t.Help.FullDesc.UnsetForeground()
	t.Help.FullSeparator = t.Help.FullSeparator.UnsetForeground()
	t.Help.Ellipsis = t.Help.Ellipsis.UnsetForeground()
	return t
}

func setupPlainHeading(out io.Writer) error {
	_, err := fmt.Fprint(out, "Dark Pawns setup\nA local world with its own players and settings.\n\n")
	return err
}
