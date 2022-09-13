package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"

	"github.com/fly-hiring/platform-challenge/pkg/config"
)

func main() {
	ctx := newCancelableContext()

	cfgStore := config.NewConfigStore("./config.json")

	// watch for changes to the config
	ch, err := cfgStore.StartWatcher()
	if err != nil {
		log.Fatalln(err)
	}
	defer cfgStore.Close()

	go func() {
		for cfg := range ch {
			fmt.Println("got config change:", cfg)
			// TODO: pm.ReloadProxy(cfg)
		}
	}()

	NewProxy().InitializeProxy(cfgStore, ctx)

	<-ctx.Done()
}

// newCancelableContext returns a context that gets canceled by a SIGINT
func newCancelableContext() context.Context {
	doneCh := make(chan os.Signal, 1)
	signal.Notify(doneCh, os.Interrupt)

	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)

	go func() {
		<-doneCh
		log.Println("signal recieved")
		cancel()
	}()

	return ctx
}

// getCurrCfg returns the current config.Config
func getCurrCfg(cfg *config.ConfigStore) config.Config {
	currCfg, err := cfg.Read()
	if err != nil {
		log.Fatalln(err)
	}

	return currCfg
}
