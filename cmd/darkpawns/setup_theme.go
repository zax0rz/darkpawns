package main

import (
	"fmt"
	"io"

	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
)

// The terminal keeps the operator's font and background. Active controls take
// the site's oxblood accent; normal text and the original art keep terminal contrast.
var setupAccent = lipgloss.AdaptiveColor{Light: "#A8201A", Dark: "#CF4B42"}

// setupLoginArt is the nine-line artwork from GREETINGS (src/config.c:235),
// also used by session.GreetingsLogo. The source fixture test prevents drift
// without making the portable launcher depend on the game/session packages.
const setupLoginArt = "         (_____)           (_)    (_____)\n" +
	"   _     /  __ \\           | |    |  __ \\                            _\n" +
	"  ;*;   /| |  | | __ _ _ __| | __ | |__) |_ _(_      _)_ __ (___)   ;*;\n" +
	"   =    /| |  | |/ _` | '__| |/ / |  ___/ _` \\ \\ /\\ / / '_ \\/ __|    =\n" +
	" .***.  /| |__| | (_| | |  |   <  | |  | (_| |\\ V  V /| | | \\__ \\  .***.\n" +
	" ~~~~~  /|_____/ \\__,_|_|  |_|\\_\\ |||   \\__,_| \\_/\\_/ |_| |_|___/  ~~~~~\n" +
	"                                  |||\n" +
	"                                  |||\n" +
	"                                  `.'"

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
