package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

const (
	testURL      = "https://www.youtube.com/watch?v=wpXmmMtPLIE"
	testDir      = "./testdata/"
	goldenDir    = "./testdata/goldenDir/"
	fileName     = "ロストワンの号哭 [wpXmmMtPLIE].m4a"
	testPlaylist = "https://youtube.com/playlist?list=PL8VzEksOSxME6gG9OjQTPNjU4ZZcj9OsK"
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
	got, err := ParsePlaylist(testPlaylist)
	if err != nil {
		t.Fatal(err)
	}

	want := []string{"https://www.youtube.com/watch?v=wpXmmMtPLIE"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got: %v, but expected %v", got, want)
	}
}

func TestRecordEntries(t *testing.T) {
	if err := RecordEntries(goldenDir); err != nil {
		t.Fatal(err)
	}

	f, err := os.ReadFile(filepath.Join(goldenDir, ".tracklist.json"))
	if err != nil {
		t.Fatal(err)
	}

	var got map[string]struct{}
	json.Unmarshal(f, &got)
	want := map[string]struct{}{"wpXmmMtPLIE": {}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got: %v, but expected %v", got, want)
	}
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

func TestRemoveEntry(t *testing.T) {
	b, err := os.ReadFile(filepath.Join(goldenDir, fileName))
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(filepath.Join(testDir, fileName), b, 0o644); err != nil {
		t.Fatal(err)
	}

	_, err = os.ReadFile(filepath.Join(testDir, fileName))
	if err != nil {
		t.Errorf("%s does not exist in %s, expected it to exist", fileName, testDir)
	}

	if err = RemoveEntry(testDir, fileName); err != nil {
		t.Fatal(err)
	}

	_, err = os.ReadFile(filepath.Join(testDir, fileName))
	if err == nil {
		t.Errorf("%s exists in %s, expected it to be deleted", fileName, testDir)
	}
}

func TestRun(t *testing.T) {
	t.Run("TestDownload", func(t *testing.T) {
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

	t.Run("TestTracklist", func(t *testing.T) {
		f, err := os.ReadFile(filepath.Join(testDir, ".tracklist.json"))
		if err != nil {
			t.Fatal(err)
		}

		var got map[string]struct{}
		json.Unmarshal(f, &got)
		want := map[string]struct{}{"wpXmmMtPLIE": {}}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("got: %v, but expected %v", got, want)
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

	t.Run("TestRemoveJSON", func(t *testing.T) {
		f, err := os.ReadFile(filepath.Join(testDir, ".tracklist.json"))
		if err != nil {
			t.Fatal(err)
		}

		var got map[string]struct{}
		json.Unmarshal(f, &got)
		want := map[string]struct{}{}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("got: %v, but expected %v", got, want)
		}
	})

	os.Remove(filepath.Join(testDir, ".tracklist.json"))
}
