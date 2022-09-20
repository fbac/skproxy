package main

import (
	"errors"
	"log"
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
