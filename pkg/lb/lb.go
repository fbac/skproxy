package main

import (
	"errors"
	"fmt"
	"log"
	"net"
	"strconv"
	"time"
)

const (
	MaxTimeout = 2 * time.Second
)

// RoundRobinLB represents a simple round robin loadbalancer
// By accesing Backend with NextBackend%len(Backend) as index
// We access the next server
// TODO: Use mutex to guarantee atomic changes
type RoundRobinLB struct {
	NextBackend int
	Backend     []string
}

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
func (lb *RoundRobinLB) SelectBackend() string {
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
