/*
Copyright © 2022 Francisco de Borja Aranda Castillejo me@fbac.dev

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with this program. If not, see <http://www.gnu.org/licenses/>.
*/
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"

	"github.com/fbac/skproxy/pkg/config"
	"github.com/fbac/skproxy/pkg/proxy"
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
		}
	}()

	proxy.NewProxy().InitializeProxy(cfgStore, ctx)

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
