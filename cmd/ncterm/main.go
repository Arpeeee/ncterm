package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/Arpeeee/ncterm/internal/nc"
	"github.com/Arpeeee/ncterm/internal/tui"
)

func main() {
	if len(os.Args) < 3 {
		printUsage()
		os.Exit(1)
	}

	cmd, path := os.Args[1], os.Args[2]

	switch cmd {
	case "info":
		runInfo(path)
	case "view":
		runView(path)
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", cmd)
		printUsage()
		os.Exit(1)
	}
}

func runInfo(path string) {
	f, err := nc.Open(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	defer f.Close()

	fmt.Print(tui.RenderInfo(f))
}

func runView(path string) {
	f, err := nc.Open(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	defer f.Close()

	p := tea.NewProgram(tui.NewViewer(f), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Fprintln(os.Stderr, "usage: ncterm <info|view> <file.nc>")
}
