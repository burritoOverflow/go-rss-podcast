package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/mmcdole/gofeed"
)

// downloadEpisode fetches an episode's audio enclosure and writes it to
// outDir, skipping the download if the destination file already exists.
func downloadEpisode(item *gofeed.Item, num int, outDir string) (string, int64, error) {
	if len(item.Enclosures) == 0 {
		return "", 0, fmt.Errorf("no downloadable audio")
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
		return filename, 0, nil
	}

	resp, err := http.Get(audioURL)
	if err != nil {
		return filename, 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return filename, 0, fmt.Errorf("bad status %s", resp.Status)
	}

	f, err := os.Create(path)
	if err != nil {
		return filename, 0, err
	}
	defer f.Close()

	written, err := io.Copy(f, resp.Body)
	if err != nil {
		return filename, 0, err
	}

	return filename, written, nil
}

func sanitizeFilename(name string) string {
	replacer := strings.NewReplacer(
		"/", "-", "\\", "-", ":", "-", "*", "-",
		"?", "", "\"", "", "<", "", ">", "", "|", "-",
	)
	return replacer.Replace(name)
}
