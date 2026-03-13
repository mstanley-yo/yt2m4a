package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

const tracklistName = ".tracklist.json"

func main() {
	url := flag.String("url", "", "youtube video/playlist url to download")
	rm := flag.String("rm", "", "remove downloaded file from directory and tracklist")
	flag.Parse()

	wd, err := os.Getwd()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	if err := run(*url, wd, *rm); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(url, dir, rm string) error {
	if rm != "" {
		if err := RemoveEntry(dir, rm); err != nil {
			return err
		}
		return nil
	}

	if url == "" {
		flag.Usage()
		return nil
	}

	seen, err := BuildSet(dir)
	if err != nil {
		return err
	}

	tl, err := ReadTracklist(dir)
	if err != nil {
		return err
	}

	if err = updateTool(); err != nil {
		return err
	}

	urls, err := ParseURL(url)
	if err != nil {
		return err
	}

	for _, u := range urls {
		id, err := urlToID(u)
		if err != nil {
			return err
		}

		if _, ok := seen[id]; ok {
			continue
		}

		if _, ok := tl.Removed[id]; ok {
			continue
		}

		_, err = DownloadOne(u, dir)
		if err != nil {
			return err
		}
	}
	return nil
}

func urlToID(url string) (string, error) {
	re := regexp.MustCompile(`youtube\.com/watch\?v=(.{11})`)
	m := re.FindStringSubmatch(url)
	if len(m) != 2 {
		return "", fmt.Errorf("failed to parse id from %q", url)
	}
	return m[1], nil
}

// DownloadOne downloads one youtube url as a mp3 into dir.
// Returns the downloaded filename and errors
func DownloadOne(url, dir string) (string, error) {
	workDir, err := os.MkdirTemp(dir, "download_*")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(workDir)

	cmd := exec.Command(
		"yt-dlp", "-f", "bestaudio[ext=m4a]/bestaudio", "--audio-format", "m4a",
		"--postprocessor-args", "ffmpeg:-c:a aac -b:a 256k",
		"--embed-metadata", "--embed-thumbnail",
		"-o", filepath.Join(workDir, "%(title)s [%(id)s].%(ext)s"),
		url,
	)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	cmd.Stdout = os.Stdout
	if err = cmd.Run(); err != nil {
		return "", fmt.Errorf("download failed: %s\n%s", err, stderr.String())
	}

	entr, err := os.ReadDir(workDir)
	if err != nil {
		return "", err
	}
	if ln := len(entr); ln != 1 {
		return "", fmt.Errorf("expected 1 file in tempdir after working, but found %d", ln)
	}
	name := entr[0].Name()
	if err := os.Rename(filepath.Join(workDir, name), filepath.Join(dir, name)); err != nil {
		return name, err
	}
	return name, nil
}

// BuildSet builds set of ids in dir
func BuildSet(dir string) (map[string]struct{}, error) {
	etrs, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	set := map[string]struct{}{}
	for _, entry := range etrs {
		id, err := entryToID(entry.Name())
		if err != nil {
			continue
		}
		set[id] = struct{}{}
	}
	return set, nil
}

func entryToID(entry string) (string, error) {
	if filepath.Ext(entry) != ".m4a" {
		return "", fmt.Errorf("got: %s, but expected a .m4a file", entry)
	}
	re := regexp.MustCompile(`\[(.{11})\]\.m4a`)
	m := re.FindStringSubmatch(entry)
	if len(m) != 2 {
		return "", fmt.Errorf("failed to parse id from %s", entry)
	}
	id := m[1]
	return id, nil
}

// updateTool updates yt-dlp on your machine
func updateTool() error {
	_, err := exec.LookPath("yt-dlp")
	if err != nil {
		return fmt.Errorf("failed to find yt-dlp in PATH: %s", err)
	}

	cmd := exec.Command("brew", "upgrade", "yt-dlp")
	cmd.Stdout = os.Stdout
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to upgrade yt-dlp: %s\n%s", err, stderr.String())
	}
	return nil
}

// ParseURL parses whether the link is a playlist or not, and converts into a slice of urls accordingly.
func ParseURL(url string) ([]string, error) {
	if strings.Contains(url, "playlist?list") {
		out, err := ParsePlaylist(url)
		if err != nil {
			return out, err
		}
		return out, nil
	}
	return []string{url}, nil
}

// ParsePlaylist parses a youtube playlist into a slice of youtube links
func ParsePlaylist(url string) ([]string, error) {
	out := []string{}

	cPath, err := exec.LookPath("yt-dlp")
	if err != nil {
		return out, fmt.Errorf("failed to find yt-dlp in PATH: %s", err)
	}

	cmd := exec.Command(cPath, "--cookies-from-browser", "chrome", "--dump-single-json", "--flat-playlist", url)
	j, err := cmd.Output()
	if err != nil {
		return out, err
	}

	type entry struct {
		ID string `json:"ID"`
	}
	var resp struct {
		Entries []entry `json:"entries"`
	}
	if err := json.Unmarshal(j, &resp); err != nil {
		return out, err
	}

	for _, e := range resp.Entries {
		if e.ID == "" {
			return out, fmt.Errorf("got %q, but expected a valid youtube id", e.ID)
		}
		out = append(out, idToURL(e.ID))
	}
	return out, nil
}

func idToURL(id string) string {
	return fmt.Sprintf("https://www.youtube.com/watch?v=%s", id)
}

// Tracklist represents the fields of the tracklist
// The tracklist doesn't need to keep a list of existing ids. The directory should be the authority on that.
type Tracklist struct {
	Removed map[string]struct{} `json:"removed"`
}

// ReadTracklist reads the tracklist file if it exists to get a set of removed ids
func ReadTracklist(dir string) (Tracklist, error) {
	tl := Tracklist{Removed: map[string]struct{}{}}
	b, err := os.ReadFile(filepath.Join(dir, tracklistName))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return tl, nil
		}
		return tl, err
	}

	if err := json.Unmarshal(b, &tl); err != nil {
		return tl, err
	}
	return tl, nil
}

// RemoveEntry removes an entry from the tracklist file.
// Adds it to Tracklist.Removed, so it doesn't get downloaded again.
func RemoveEntry(dir, entry string) error {
	tl, err := ReadTracklist(dir)
	if err != nil {
		return err
	}

	id, err := entryToID(entry)
	if err != nil {
		return err
	}

	tl.Removed[id] = struct{}{}
	js, err := json.Marshal(tl)
	if err != nil {
		return err
	}

	filename := filepath.Join(dir, tracklistName)
	if err := os.WriteFile(filename, js, 0o644); err != nil {
		return err
	}

	if err := os.Remove(filepath.Join(dir, entry)); err != nil {
		return err
	}

	fmt.Printf("Removed: %s\n", entry)
	return nil
}
