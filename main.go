package main

import (
	"flag"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/mmcdole/gofeed"
)

func main() {
	url := flag.String("url", "", "RSS feed URL (required)")
	out := flag.String("out", "downloads", "Directory to save downloaded episodes")
	flag.Parse()

	if *url == "" {
		fmt.Println("Error: --url is required")
		flag.Usage()
		os.Exit(1)
	}

	fp := gofeed.NewParser()
	feed, err := fp.ParseURL(*url)
	if err != nil {
		fmt.Println("Error parsing feed:", err)
		os.Exit(1)
	}

	if err := os.MkdirAll(*out, 0755); err != nil {
		fmt.Println("Error creating output dir:", err)
		os.Exit(1)
	}

	p := tea.NewProgram(newModel(feed, *out), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Println("Error running UI:", err)
		os.Exit(1)
	}
}
