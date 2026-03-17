package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// Tracklist records ids removed from the directory and blacklists them so they don't get downloaded again.
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

// SaveTracklist saves json to the tracklist file
func SaveTracklist(dir string, data []byte) error {
	filename := filepath.Join(dir, tracklistName)
	return os.WriteFile(filename, data, 0o644)
}

// RemoveEntry removes an entry from the directory.
func RemoveEntry(dir, entry string) error {
	if err := os.Remove(filepath.Join(dir, entry)); err != nil {
		return err
	}
	fmt.Printf("Removed: %s\n", entry)
	return nil
}

// AddBlacklist adds an id to Tracklist.Removed, so it doesn't get downloaded again.
func AddBlacklist(dir, id string) error {
	tl, err := ReadTracklist(dir)
	if err != nil {
		return err
	}

	tl.Removed[id] = struct{}{}
	js, err := json.Marshal(tl)
	if err != nil {
		return err
	}

	return SaveTracklist(dir, js)
}

// RemoveBlacklist removes an id from tl.Removed and saves the tracklist
func RemoveBlacklist(dir, id string) error {
	tl, err := ReadTracklist(dir)
	if err != nil {
		return err
	}

	delete(tl.Removed, id)
	js, err := json.Marshal(tl)
	if err != nil {
		return err
	}

	return SaveTracklist(dir, js)
}

