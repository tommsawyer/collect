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
	"strings"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"
)

// Dump will dump every profile into given folder following this structure:
// - provided directory
//   - host port
//    - YYYY MM DD
//      - HH MM SS
//        - profile
func Dump(ctx context.Context, dir, base string, profiles map[string][]byte) error {
	u, err := url.Parse(base)
	if err != nil {
		return err
	}

	folder := path.Join(dir, u.Hostname()+" "+u.Port()) + "/" + time.Now().Format("2006 01 02/15 04 05")
	if err := os.MkdirAll(folder, os.ModePerm); err != nil {
		return fmt.Errorf("cannot create directory %q: %w", folder, err)
	}

	for profile, content := range profiles {
		filePath := path.Join(folder, profile)
		if err := ioutil.WriteFile(filePath, content, os.ModePerm); err != nil {
			return fmt.Errorf("cannot write %q profile: %w", profile, err)
		}
		log.Printf("[%s] wrote %s to %s\n", base, profile, filePath)
	}

	return nil
}

// Collect will collect all provided profiles.
//
// You can add query parameters to profile like so:
//  Collect(ctx, "http://localhost:8080", []string{"trace?seconds=5"})
func Collect(ctx context.Context, baseURL string, profiles []string) (map[string][]byte, error) {
	client := &http.Client{
		Timeout: time.Minute,
	}

	var mx sync.Mutex
	collectedProfiles := make(map[string][]byte, len(profiles))

	g, ctx := errgroup.WithContext(ctx)
	for _, profile := range profiles {
		url := baseURL + "/debug/pprof/" + profile
		profileName := strings.Split(profile, "?")[0]
		g.Go(func() error {
			req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
			if err != nil {
				return fmt.Errorf("cannot build url: %w", err)
			}

			log.Printf("[%s] collecting %s\n", baseURL, profileName)
			resp, err := client.Do(req)
			if err != nil {
				return fmt.Errorf("cannot collect %s: %w", profileName, err)
			}
			defer resp.Body.Close()

			bytes, err := io.ReadAll(resp.Body)
			if err != nil {
				return fmt.Errorf("cannot collect %s: %w", profileName, err)
			}

			mx.Lock()
			collectedProfiles[profileName] = bytes
			mx.Unlock()

			log.Printf("[%s] successfully collected %s\n", baseURL, profileName)
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		return nil, err
	}

	return collectedProfiles, nil
}
