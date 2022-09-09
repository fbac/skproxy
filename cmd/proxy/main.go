package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/signal"
	"strconv"
	"time"

	"github.com/fbac/proxy-tcp-roundrobin/pkg/config"
)

const (
	MaxTimeout = 2 * time.Second
)

// ProxyMap represents a proxy map, where the key is the app to be proxied
// the value is a config.App struct, which holds the frontends and backends
// this way we can identify and access apps independently, with complexity O(1)
type ProxyMap map[string]config.App

// RoundRobinLB represents a simple round robin loadbalancer
// By accesing Backend with NextBackend%len(Backend) as index
// We access the next server
// TODO: Use mutex to guarantee atomic changes
type RoundRobinLB struct {
	NextBackend int
	Backend     []string
}

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

	NewProxy().InitializeProxy(cfgStore)

	<-ctx.Done()
}

/* Config related functions */

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

/* Proxy related functions */

// NewProxyMap returns a new ProxyMap instance
// TODO this should be run in a context
func NewProxy() *ProxyMap {
	pm := make(ProxyMap)
	return &pm
}

// populateProxyMap converts a given ConfigStore to a ProxyMap
func (pm ProxyMap) populateProxyMap(cfg *config.ConfigStore) {
	currCfg := getCurrCfg(cfg)
	for _, v := range currCfg.Apps {
		pm[v.Name] = v
	}
}

// getFrontends returns all the listener ports for a given App
func (pm ProxyMap) getFrontends(host string) []int {
	var fe []int
	fe = append(fe, pm[host].Ports...)
	return fe
}

// getBackends returns all the backends for a given App
func (pm ProxyMap) getBackends(host string) []string {
	var be []string
	be = append(be, pm[host].Targets...)
	return be
}

// InitializeProxy initializes the proxy for a given ConfigStore
// TODO this should be run in a context
func (pm ProxyMap) InitializeProxy(cfg *config.ConfigStore) {
	pm.populateProxyMap(cfg)

	for app := range pm {
		frontends := pm.getFrontends(app)
		backends := pm.getBackends(app)
		doProxy(frontends, backends)
	}
}

// prepareProxy is an intermediate step to configure the loadbalancer
// also makes the app extensible for the future, to add additional checks here
// TODO this should be run in a context
func doProxy(fe []int, be []string) {
	lb := NewLB(be)

	for _, f := range fe {
		go proxy(f, *lb)
	}
}

// proxy handles the proxy logic
// it copies the net.Conn datastream from src to dst, and back from dst to src
// meaning that in the low level what it does is redirecting the socket file descriptor, so the connection is copied/proxied transparently
// this way we make sure the correct listeners will be tied to the correct backend apps
// TODO Each proxy should be run in a context
func proxy(fe int, lb RoundRobinLB) {
	// start listener
	l := fmt.Sprintf(":%v", fe)
	listener, err := net.Listen("tcp", l)
	if err != nil {
		log.Fatalln(err)
	}
	defer listener.Close()
	log.Println("started listener in", l)

	// Loop indefinitely to catch new connections
	for {
		// Get new data
		c, err := listener.Accept()
		if err != nil {
			log.Printf("failed to accept connection: %s", err)
		}

		// Select next backend
		backend := lb.selectBackend()
		log.Printf("proxying data from %v to %s", l, backend)
		go func() {
			b, err := net.Dial("tcp", backend)
			if err != nil {
				log.Fatalln(err)
			}
			go io.Copy(b, c)
			go io.Copy(c, b)
		}()
	}
}

/* LB related functions */

// NewLB returns a new RoundRobinLB instance
func NewLB(be []string) *RoundRobinLB {
	var lb []string
	for _, rawUri := range be {
		host, port := getHostPort(rawUri)
		if healthCheck(host, port) {
			lb = append(lb, rawUri)
		}
	}
	return &RoundRobinLB{NextBackend: 0, Backend: lb}
}

// selectBackend selects the next backend
// it implements a round robin logic (roughly tbh)
func (lb *RoundRobinLB) selectBackend() string {
	var be string
	if len(lb.Backend) > 0 {
		be = lb.Backend[lb.NextBackend%len(lb.Backend)]
		lb.incNextBackend()
	} else {
		log.Fatalln(errors.New("no available backends"))
	}

	return be
}

// incNextBackend increases NextBackend, just a helper method for cleaner code
func (lb *RoundRobinLB) incNextBackend() {
	lb.NextBackend += 1
}

/* Helper functions */

// getHostPort returns host and port separatedly given a raw uri
func getHostPort(rawUri string) (string, int) {
	host, p, err := net.SplitHostPort(rawUri)
	if err != nil {
		log.Fatalln(err)
	}

	port, err := strconv.Atoi(p)
	if err != nil {
		log.Fatalln(err)
	}

	return host, port
}

// getFullUri returns a raw uri given a host and port
func getFullUri(host string, port int) string {
	return fmt.Sprintf("%s:%v", host, port)
}

// healthCheck implements a really basic network healthCheck; return true means healthy address
//
// Ideally, in a prod-ready proxy:
// - It also checks a path exposed by the backend, given at configuration time
// - Serves the connection status via channel, so it's handled by the proxy in a select loop
// - It's called from doProxy (in a loop, with a ticker) while the conn is active
// - The ticker is configurable with HealthCheckPeriod
func healthCheck(address string, port int) bool {
	// check for dns resolution
	_, err := net.LookupIP(address)
	if err != nil {
		log.Printf("skipping unhealthy backend: %s: %v\n", address, err)
		return false
	}

	// check for tcp connectivity
	// if this would be a prod-ready app, it's a good idea to set a MaxTimeout as a flag, not a const
	_, err = net.DialTimeout("tcp", getFullUri(address, port), MaxTimeout)
	if err != nil {
		log.Printf("skipping unhealthy backend: %s: %v\n", address, err)
		return false
	}

	return true
}

/* TODO functions */

// TODO: ReloadProxy reloads all the data structures when the configuration is reloaded
// The process of proxying the applications starts all over when reloadCfg is called
// Existing connections might be dropped if the data is not handled correctly
func (pm ProxyMap) ReloadProxy(cfg config.Config) {}
