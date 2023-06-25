package main

import (
	"context"
	"errors"
	"log"
	"os"
	"os/signal"
	"time"

	"github.com/jessevdk/go-flags"
	"github.com/tommsawyer/collect/profiles"
	"golang.org/x/sync/errgroup"
)

var cfg struct {
	Hosts     []string      `short:"u" description:"hosts from which profiles will be collected" required:"true"`
	Profiles  []string      `short:"p" description:"profiles to collect. Possible options: allocs/heap/goroutine/profile/trace." default:"allocs" default:"heap" default:"goroutine" default:"profile"`
	Loop      bool          `short:"l" description:"collect many times (until Ctrl-C)"`
	Interval  time.Duration `short:"i" description:"interval between collecting (use with -l)" default:"60s"`
	Directory string        `short:"d" description:"directory to put the pprof files in" default:"."`
	KeepGoing bool          `short:"k" description:"keep going collect profiles if some requests failed"`
}

func main() {
	if _, err := flags.Parse(&cfg); err != nil {
		return
	}

	log.Println("collecting profiles. hit Ctrl-C any time to stop.")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)
	go func() {
		<-interrupt
		cancel()
	}()

	if !cfg.Loop {
		if err := collectAndDump(ctx, cfg.Directory, cfg.Hosts, cfg.Profiles, cfg.KeepGoing); err != nil {
			log.Fatalln(err)
		}

		return
	}

	for {
		if err := collectAndDump(ctx, cfg.Directory, cfg.Hosts, cfg.Profiles, cfg.KeepGoing); err != nil {
			if errors.Is(err, context.Canceled) {
				return
			}

			log.Fatalln(err)
		}

		log.Printf("sleeping for %v before next collect...", cfg.Interval)
		select {
		case <-time.After(cfg.Interval):
		case <-ctx.Done():
			return
		}
	}
}

func collectAndDump(ctx context.Context, baseDir string, hosts []string, profilesToCollect []string, ignoreNetworkErrors bool) error {
	g, ctx := errgroup.WithContext(ctx)

	for _, host := range hosts {
		h := host
		g.Go(func() error {
			collected, err := profiles.Collect(ctx, h, profilesToCollect, ignoreNetworkErrors)
			if err != nil {
				return err
			}

			return profiles.Dump(ctx, baseDir, h, collected)
		})
	}

	return g.Wait()
}
