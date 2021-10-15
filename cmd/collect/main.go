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
)

var cfg struct {
	Hosts    []string      `short:"u" description:"hosts from which profiles will be collected" required:"true"`
	Profiles []string      `short:"p" description:"profiles to collect. Possible options: allocs/heap/goroutine/profile/trace." default:"allocs" default:"heap" default:"goroutine" default:"profile"`
	Loop     bool          `short:"l" description:"collect many times (until Ctrl-C)"`
	Interval time.Duration `short:"i" description:"interval between collecting (use with -l)" default:"60s"`
}

func main() {
	if _, err := flags.Parse(&cfg); err != nil {
		return
	}

	log.Println("collecting profiles. hit Ctrl-C any time to stop.")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if !cfg.Loop {
		ctx := context.Background()
		if err := profiles.CollectAndDump(ctx, cfg.Hosts, cfg.Profiles); err != nil {
			log.Fatalln(err)
		}

		return
	}

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		<-c
		cancel()
	}()

	for {
		if err := profiles.CollectAndDump(ctx, cfg.Hosts, cfg.Profiles); err != nil {
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