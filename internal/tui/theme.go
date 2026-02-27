package tui

import "github.com/charmbracelet/lipgloss"

// palette holds the accent colours for one named theme.
type palette struct {
	output   lipgloss.Color // assistant text bullet
	answer   lipgloss.Color // user answer / tool status
	success  lipgloss.Color // completion bullet
	errColor lipgloss.Color // error bullet
	question lipgloss.Color // option number highlight
	bg       lipgloss.Color // status bar / panel background
	faint    lipgloss.Color // dim text
}

var palettes = map[string]palette{
	"github-dark": {
		output:   "#c9d1d9",
		answer:   "#58a6ff",
		success:  "#3fb950",
		errColor: "#f85149",
		question: "#58a6ff",
		bg:       "#21262d",
		faint:    "#8b949e",
	},
	"dracula": {
		output:   "#f8f8f2",
		answer:   "#8be9fd",
		success:  "#50fa7b",
		errColor: "#ff5555",
		question: "#bd93f9",
		bg:       "#44475a",
		faint:    "#6272a4",
	},
	"nord": {
		output:   "#d8dee9",
		answer:   "#88c0d0",
		success:  "#a3be8c",
		errColor: "#bf616a",
		question: "#81a1c1",
		bg:       "#3b4252",
		faint:    "#4c566a",
	},
	"solarized": {
		output:   "#839496",
		answer:   "#268bd2",
		success:  "#859900",
		errColor: "#dc322f",
		question: "#2aa198",
		bg:       "#073642",
		faint:    "#586e75",
	},
	"monokai": {
		output:   "#f8f8f2",
		answer:   "#66d9e8",
		success:  "#a6e22e",
		errColor: "#f92672",
		question: "#e6db74",
		bg:       "#3e3d32",
		faint:    "#75715e",
	},
}

var themeOrder = []string{"dracula", "github-dark", "nord", "solarized", "monokai"}

// glamourStyle maps our theme name to a glamour built-in style.
func glamourStyle(name string) string {
	switch name {
	case "dracula":
		return "dracula"
	default:
		return "dark"
	}
}
