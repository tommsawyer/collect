package profiles

import (
	"context"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"
)

type testLogger struct {
	t *testing.T
}

func (w *testLogger) LogTo(t *testing.T) {
	w.t = t
}

func (w *testLogger) Write(msg []byte) (int, error) {
	w.t.Log(string(msg))
	return len(msg), nil
}

func TestCollectProfiles(t *testing.T) {
	testLog := &testLogger{t}
	log.SetOutput(testLog)
	defer log.SetOutput(os.Stdout)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, strings.TrimPrefix(r.URL.String(), "/debug/pprof/"))
	}))
	defer srv.Close()

	t.Run("collect one profile", func(t *testing.T) {
		testLog.LogTo(t)
		profiles, err := Collect(context.Background(), srv.URL, []string{"allocs"}, false)
		if err != nil {
			t.Errorf("error should be nil, but got %v", err)
		}

		if string(profiles["allocs"]) != "allocs" {
			t.Errorf("should download allocs profile but got %s", string(profiles["allocs"]))
		}
	})

	t.Run("collect many profiles", func(t *testing.T) {
		testLog.LogTo(t)
		profiles, err := Collect(context.Background(), srv.URL, []string{"allocs", "profile"}, false)
		if err != nil {
			t.Errorf("error should be nil, but got %v", err)
		}

		if string(profiles["allocs"]) != "allocs" {
			t.Errorf("should download allocs profile but got %s", string(profiles["allocs"]))
		}
		if string(profiles["profile"]) != "profile" {
			t.Errorf("should dowlnoad cpu profile but got %s", string(profiles["profile"]))
		}
	})

	t.Run("collect profile with query parameters", func(t *testing.T) {
		testLog.LogTo(t)
		profiles, err := Collect(context.Background(), srv.URL, []string{"trace?seconds=5"}, false)
		if err != nil {
			t.Errorf("error should be nil, but got %v", err)
		}

		if string(profiles["trace"]) != "trace?seconds=5" {
			t.Errorf("should download trace profile")
		}
	})

	t.Run("returns error when cannot build url", func(t *testing.T) {
		_, err := Collect(context.Background(), "http://wrong.wrong\n", []string{"allocs"}, false)
		if err == nil || !strings.Contains(err.Error(), "cannot build url:") {
			t.Error("should return error when wrong url provided")
		}
	})

	t.Run("returns error when server is unreachable", func(t *testing.T) {
		_, err := Collect(context.Background(), "http://wrong.wrong", []string{"allocs"}, false)
		if err == nil || !strings.Contains(err.Error(), "cannot collect allocs:") {
			t.Error("should return error when server is unreachable")
		}
	})

	t.Run("returns no error when server is unreachable and ignore network errors is true", func(t *testing.T) {
		_, err := Collect(context.Background(), "http://wrong.wrong", []string{"allocs"}, true)
		if err != nil {
			t.Errorf("error should be nil, but got %v", err)
		}
	})
}

func TestDump(t *testing.T) {
	testLog := &testLogger{t}
	log.SetOutput(testLog)
	defer log.SetOutput(os.Stdout)

	testDir := path.Join(os.TempDir(), "test")
	err := os.Mkdir(path.Join(os.TempDir(), "test"), os.ModePerm)
	if err != nil {
		t.Fatalf("cannot create test directory: %v", err)
	}
	defer os.RemoveAll(testDir)

	profiles := map[string][]byte{
		"allocs": []byte("allocs"),
		"heap":   []byte("heap"),
	}
	err = Dump(context.Background(), testDir, "http://localhost:8080", profiles)
	if err != nil {
		t.Fatalf("error should be nil, but got %v", err)
	}

	fileContents := map[string][]byte{}
	err = filepath.Walk(testDir, func(path string, info fs.FileInfo, err error) error {
		if !info.IsDir() {
			content, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			fileContents[info.Name()] = content
		}
		return nil
	})
	if err != nil {
		t.Fatalf("cannot walk through files: %v", err)
	}

	for profile, content := range profiles {
		if string(fileContents[profile]) != string(content) {
			t.Errorf("file %s contains wrong data: %q", profile, string(fileContents[profile]))
		}
	}
}

func TestDumpParseURL(t *testing.T) {
	testLog := &testLogger{t}
	log.SetOutput(testLog)
	defer log.SetOutput(os.Stdout)

	err := Dump(context.Background(), "./", "http://localhost:8080\n", nil)
	if err == nil {
		t.Fatalf("should return an error when cannot parse url")
	}
}

func TestDumpCannotCreateDirectory(t *testing.T) {
	testLog := &testLogger{t}
	log.SetOutput(testLog)
	defer log.SetOutput(os.Stdout)

	err := Dump(context.Background(), "/dev/null/:", "http://localhost:8080", nil)
	if err == nil {
		t.Fatalf("should return an error when cannot create directory")
	}
}
