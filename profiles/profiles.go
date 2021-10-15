package profiles

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"path"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"
)

var profilePaths = map[string]string{
	"allocs":    "/debug/pprof/allocs",
	"heap":      "/debug/pprof/heap",
	"goroutine": "/debug/pprof/goroutine",
	"profile":   "/debug/pprof/profile",
}

var client = &http.Client{
	Timeout: time.Minute,
}

// CollectAndDump will collect all provided profiles and dump into given folder.
func CollectAndDump(ctx context.Context, baseURLs []string, profiles []string) error {
	g, ctx := errgroup.WithContext(ctx)
	for _, baseURL := range baseURLs {
		base := baseURL

		g.Go(func() error {
			u, err := url.Parse(base)
			if err != nil {
				return err
			}

			collectedProfiles, err := Collect(ctx, base, profiles)
			if err != nil {
				return err
			}

			folder := u.Host + "/" + time.Now().Format("2006.01.02/15:04:05")
			if err := os.MkdirAll(folder, os.ModePerm); err != nil {
				return fmt.Errorf("cannot create %q: %w", folder, err)
			}

			for profile, content := range collectedProfiles {
				filePath := path.Join(folder, profile)
				if err := ioutil.WriteFile(filePath, content, os.ModePerm); err != nil {
					return fmt.Errorf("cannot write %q profile: %w", profile, err)
				}
				log.Printf("[%s] wrote %s to ./%s\n", base, profile, filePath)
			}

			return nil
		})
	}

	return g.Wait()
}

// Collect will collect all provided profiles.
func Collect(ctx context.Context, baseURL string, profiles []string) (map[string][]byte, error) {
	for _, profile := range profiles {
		if _, exists := profilePaths[profile]; !exists {
			return nil, fmt.Errorf("profile %q doesnt exist", profile)
		}
	}

	var mx sync.Mutex
	result := make(map[string][]byte, len(profiles))

	g, ctx := errgroup.WithContext(ctx)
	for _, profile := range profiles {
		path := profilePaths[profile]
		url := baseURL + path
		profileName := profile
		g.Go(func() error {
			req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
			if err != nil {
				return err
			}
			log.Printf("[%s] collecting %s\n", baseURL, profileName)
			resp, err := client.Do(req)
			if err != nil {
				return err
			}
			defer resp.Body.Close()

			bytes, err := io.ReadAll(resp.Body)
			if err != nil {
				return err
			}

			mx.Lock()
			result[profileName] = bytes
			mx.Unlock()

			log.Printf("[%s] successfully collected %s\n", baseURL, profileName)
			return nil
		})
	}

	return result, g.Wait()
}
