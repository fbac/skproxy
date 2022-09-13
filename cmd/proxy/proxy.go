package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"strconv"
	"time"

	"github.com/fly-hiring/platform-challenge/pkg/config"
)

const (
	MaxTimeout = 2 * time.Second
)

// ProxyMap represents a proxy map, where the key is the app to be proxied
// the value is a config.App struct, which holds the frontends and backends
// this way we can identify and access apps independently, with complexity O(1)
type ProxyMap map[string]config.App

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
func (pm ProxyMap) InitializeProxy(cfg *config.ConfigStore, ctx context.Context) {
	pm.populateProxyMap(cfg)

	for app := range pm {
		frontends := pm.getFrontends(app)
		backends := pm.getBackends(app)
		doProxy(app, frontends, backends, ctx)
	}
}

// prepareProxy is an intermediate step to configure the loadbalancer
// also makes the app extensible for the future, to add additional checks here
// TODO this should be run in a context
func doProxy(app string, fe []int, be []string, ctx context.Context) {
	lb := NewLB(be)
	go proxy(app, fe, *lb, ctx)
}

// proxy handles the proxy logic
// it copies the net.Conn datastream from src to dst, and back from dst to src
// meaning that in the low level what it does is redirecting the socket file descriptor, so the connection is copied/proxied transparently
// this way we make sure the correct listeners will be tied to the correct backend apps
// TODO Each proxy should be run in a context
func proxy(app string, fe []int, lb RoundRobinLB, ctx context.Context) {
	// Create listener and ports
	l := fmt.Sprintf(":%v", fe[0])
	var p []uint16

	for _, v := range fe[1:] {
		p = append(p, uint16(v))
	}

	// Resolve the TCP Addr to get *net.TCPAddr
	addr, err := net.ResolveTCPAddr("tcp", fmt.Sprintf("%v", l))
	if err != nil {
		log.Fatalln(err)
	}

	// ListenTCP to get a *net.TCPListener
	listener, err := net.ListenTCP("tcp", addr)
	if err != nil {
		log.Fatalln(err)
	}
	defer listener.Close()
	log.Println("started listener in", l)
	go InitializeEbpfProg(app, listener, p, ctx)

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
