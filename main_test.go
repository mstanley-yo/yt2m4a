package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

const (
	testURL         = "https://www.youtube.com/watch?v=wpXmmMtPLIE"
	testDir         = "./testdata/"
	goldenDir       = "./testdata/goldenDir/"
	fileName        = "ロストワンの号哭 [wpXmmMtPLIE].m4a"
	testPlaylistURL = "https://youtube.com/playlist?list=PL8VzEksOSxME6gG9OjQTPNjU4ZZcj9OsK"
)

func TestBuildSet(t *testing.T) {
	got, err := BuildSet(goldenDir)
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]struct{}{"wpXmmMtPLIE": {}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got:\n%v\nwant:\n%v\n", got, want)
	}
}

func TestDownload(t *testing.T) {
	_, err := DownloadOne(testURL, testDir)
	if err != nil {
		t.Fatal(err)
	}

	result, err := os.ReadFile(filepath.Join(testDir, fileName))
	if err != nil {
		t.Fatalf("could not read result file: %s", err)
	}
	expected, err := os.ReadFile(filepath.Join(goldenDir, fileName))
	if err != nil {
		t.Fatalf("could not read golden file: %s", err)
	}

	if !bytes.Equal(expected, result) {
		t.Logf("golden:\n%s\n", expected)
		t.Logf("result:\n%s\n", result)
		t.Error("Result content does not match golden file")
	}

	if err := os.Remove(filepath.Join(testDir, fileName)); err != nil {
		t.Fatalf("failed to clean up: %s", err)
	}
}

func TestParsePlaylist(t *testing.T) {
	got, err := ParsePlaylist(testPlaylistURL)
	if err != nil {
		t.Fatal(err)
	}

	t.Logf("ParsePlaylist(%s) -> %s\n", testPlaylistURL, got)
	want := []string{testURL}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got: %v, but expected %v", got, want)
	}
}

func TestParseURL(t *testing.T) {
	t.Run("TestParseWatch", func(t *testing.T) {
		got, err := ParseURL(testURL)
		if err != nil {
			t.Fatal(err)
		}

		want := []string{testURL}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("got %v; want %v", got, want)
		}
	})

	t.Run("TestParsePlaylist", func(t *testing.T) {
		got, err := ParseURL(testPlaylistURL)
		if err != nil {
			t.Fatal(err)
		}

		want := []string{testURL}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("got %v; want %v", got, want)
		}
	})
}

func TestEntryToID(t *testing.T) {
	id, err := entryToID(fileName)
	if err != nil {
		t.Fatal(err)
	}

	want := "wpXmmMtPLIE"
	if id != want {
		t.Errorf("got %q, but expected %q", id, want)
	}
}

func copyToTestDir(path string) error {
	b, err := os.ReadFile(filepath.Join(goldenDir, path))
	if err != nil {
		return err
	}
	if err = os.WriteFile(filepath.Join(testDir, path), b, 0o644); err != nil {
		return err
	}
	return nil
}

// TestEntries tests the entry system. Should be able to maintain a removed track in a file
func TestEntries(t *testing.T) {
	t.Run("TestReadEmpty", func(t *testing.T) {
		got, err := ReadTracklist(testDir)
		if err != nil {
			t.Fatal(err)
		}

		want := Tracklist{Removed: map[string]struct{}{}}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("got %v; want %v", got, want)
		}
	})

	if err := copyToTestDir(fileName); err != nil {
		t.Fatal(err)
	}

	t.Run("TestReadExisting", func(t *testing.T) {
		got, err := ReadTracklist(testDir)
		if err != nil {
			t.Fatal(err)
		}

		want := Tracklist{Removed: map[string]struct{}{}}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("got %v; want %v", got, want)
		}
	})

	t.Run("TestRemoveEntry", func(t *testing.T) {
		if err := RemoveEntry(testDir, fileName); err != nil {
			t.Fatal(err)
		}

		_, err := os.ReadFile(filepath.Join(testDir, fileName))
		if err == nil || !errors.Is(err, os.ErrNotExist) {
			t.Errorf("%s exists in %s, expected it to be deleted", fileName, testDir)
		}

		b, err := os.ReadFile(filepath.Join(testDir, tracklistName))
		if err != nil {
			t.Fatal(err)
		}
		var got Tracklist
		if err := json.Unmarshal(b, &got); err != nil {
			t.Fatal(err)
		}
		want := Tracklist{Removed: map[string]struct{}{"wpXmmMtPLIE": {}}}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("got %v; want %v", got, want)
		}
	})

	t.Run("TestRemovedDownload", func(t *testing.T) {
		if err := run(testURL, testDir, ""); err != nil {
			t.Fatal(err)
		}

		_, err := os.ReadFile(filepath.Join(testDir, fileName))
		if err == nil || !errors.Is(err, os.ErrNotExist) {
			t.Errorf("%s exists in %s, expected it to be skipped", fileName, testDir)
		}
	})

	if err := os.Remove(filepath.Join(testDir, tracklistName)); err != nil {
		t.Fatal(err)
	}
}

func TestRun(t *testing.T) {
	t.Run("TestDownloadWatch", func(t *testing.T) {
		if err := run(testURL, testDir, ""); err != nil {
			t.Fatal(err)
		}

		result, err := os.ReadFile(filepath.Join(testDir, fileName))
		if err != nil {
			t.Fatalf("could not read result file: %s", err)
		}
		expected, err := os.ReadFile(filepath.Join(goldenDir, fileName))
		if err != nil {
			t.Fatalf("could not read golden file: %s", err)
		}

		if !bytes.Equal(expected, result) {
			t.Logf("golden:\n%s\n", expected)
			t.Logf("result:\n%s\n", result)
			t.Error("Result content does not match golden file")
		}
	})

	if err := os.Remove(filepath.Join(testDir, fileName)); err != nil {
		t.Fatal(err)
	}

	t.Run("TestDownloadPlaylist", func(t *testing.T) {
		if err := run(testPlaylistURL, testDir, ""); err != nil {
			t.Fatal(err)
		}

		result, err := os.ReadFile(filepath.Join(testDir, fileName))
		if err != nil {
			t.Fatalf("could not read result file: %s", err)
		}
		expected, err := os.ReadFile(filepath.Join(goldenDir, fileName))
		if err != nil {
			t.Fatalf("could not read golden file: %s", err)
		}

		if !bytes.Equal(expected, result) {
			t.Logf("golden:\n%s\n", expected)
			t.Logf("result:\n%s\n", result)
			t.Error("Result content does not match golden file")
		}
	})

	t.Run("TestRemoveFile", func(t *testing.T) {
		if err := run("", testDir, fileName); err != nil {
			t.Fatal(err)
		}

		_, err := os.ReadFile(filepath.Join(testDir, fileName))
		if err == nil {
			t.Errorf("%s exists in %s, expected it to be deleted", fileName, testDir)
		}
	})

	t.Run("TestRemoveTracklist", func(t *testing.T) {
		f, err := os.ReadFile(filepath.Join(testDir, ".tracklist.json"))
		if err != nil {
			t.Fatal(err)
		}

		var got Tracklist
		json.Unmarshal(f, &got)
		want := Tracklist{Removed: map[string]struct{}{"wpXmmMtPLIE": {}}}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("got: %v, but expected %v", got, want)
		}
	})

	os.Remove(filepath.Join(testDir, ".tracklist.json"))
}
