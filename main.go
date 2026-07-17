package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/mmcdole/gofeed"
)

const pageSize = 10

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

	fmt.Printf("Podcast: %s\n", feed.Title)
	fmt.Printf("Episodes: %d\n\n", len(feed.Items))

	if err := os.MkdirAll(*out, 0755); err != nil {
		fmt.Println("Error creating output dir:", err)
		os.Exit(1)
	}

	runUI(feed.Items, *out)
}

// runUI is a minimal, dependency-free terminal UI: it paginates the episode
// list (feed.Items already holds everything gofeed parsed from the RSS doc)
// and lets the user pick episodes to download.
func runUI(items []*gofeed.Item, outDir string) {
	reader := bufio.NewReader(os.Stdin)
	page := 0
	totalPages := (len(items) + pageSize - 1) / pageSize
	if totalPages == 0 {
		totalPages = 1
	}

	for {
		printPage(items, page, totalPages)

		fmt.Print("\n[n]ext, [p]rev, <number> to download, [a] download all on page, [q]uit: ")
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(strings.ToLower(input))

		switch {
		case input == "n":
			if page < totalPages-1 {
				page++
			} else {
				fmt.Println("Already on last page.")
			}
		case input == "p":
			if page > 0 {
				page--
			} else {
				fmt.Println("Already on first page.")
			}
		case input == "q":
			fmt.Println("Bye!")
			return
		case input == "a":
			start, end := pageBounds(page, len(items))
			for i := start; i < end; i++ {
				downloadEpisode(items[i], i+1, outDir)
			}
		default:
			num, err := strconv.Atoi(input)
			if err != nil || num < 1 || num > len(items) {
				fmt.Println("Invalid input.")
				continue
			}
			downloadEpisode(items[num-1], num, outDir)
		}
	}
}

func pageBounds(page, total int) (start, end int) {
	start = page * pageSize
	end = start + pageSize
	if end > total {
		end = total
	}
	return
}

func printPage(items []*gofeed.Item, page, totalPages int) {
	start, end := pageBounds(page, len(items))

	fmt.Printf("\n--- Page %d of %d ---\n", page+1, totalPages)
	for i := start; i < end; i++ {
		item := items[i]
		duration := ""
		if item.ITunesExt != nil {
			duration = item.ITunesExt.Duration
		}
		fmt.Printf("%3d. %s", i+1, item.Title)
		if duration != "" {
			fmt.Printf(" (%s)", duration)
		}
		fmt.Println()
	}
}

func downloadEpisode(item *gofeed.Item, num int, outDir string) {
	if len(item.Enclosures) == 0 {
		fmt.Printf("  Episode %d has no downloadable audio.\n", num)
		return
	}

	audioURL := item.Enclosures[0].URL
	ext := filepath.Ext(audioURL)
	if idx := strings.IndexAny(ext, "?#"); idx != -1 {
		ext = ext[:idx]
	}
	if ext == "" {
		ext = ".mp3"
	}

	filename := sanitizeFilename(fmt.Sprintf("%03d_%s%s", num, item.Title, ext))
	path := filepath.Join(outDir, filename)

	if _, err := os.Stat(path); err == nil {
		fmt.Printf("  Skipping (already exists): %s\n", filename)
		return
	}

	fmt.Printf("  Downloading: %s ... ", filename)

	resp, err := http.Get(audioURL)
	if err != nil {
		fmt.Println("failed:", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		fmt.Println("failed: bad status", resp.Status)
		return
	}

	f, err := os.Create(path)
	if err != nil {
		fmt.Println("failed:", err)
		return
	}
	defer f.Close()

	written, err := io.Copy(f, resp.Body)
	if err != nil {
		fmt.Println("failed:", err)
		return
	}

	fmt.Printf("done (%.1f MB)\n", float64(written)/1024/1024)
}

func sanitizeFilename(name string) string {
	replacer := strings.NewReplacer(
		"/", "-", "\\", "-", ":", "-", "*", "-",
		"?", "", "\"", "", "<", "", ">", "", "|", "-",
	)
	return replacer.Replace(name)
}
