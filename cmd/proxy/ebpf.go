package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"
	"unsafe"

	"github.com/cilium/ebpf"
	"github.com/cilium/ebpf/link"
)

const (
	socketKey    int32  = 0
	echoSocket   string = "/sys/fs/bpf/echo_socket"
	echoPorts    string = "/sys/fs/bpf/echo_ports"
	dispatchProg string = "/sys/fs/bpf/echo_dispatch_prog"
	dispatchLink string = "/sys/fs/bpf/echo_dispatch_link"
	filename     string = "echo_dispatch.bpf.o"
)

func InitializeEbpfProg(app string, listener *net.TCPListener, ports []uint16, ctx context.Context) {
	// Get path where fly-proxy is running
	cmd, err := os.Executable()
	if err != nil {
		panic(err)
	}
	binPath := fmt.Sprintf("%s/%s", filepath.Dir(cmd), filename)

	// Load the eBPF elf binary
	spec, err := ebpf.LoadCollectionSpec(binPath)
	if err != nil {
		panic(err)
	}

	// Initialize vars
	echoSocketApp := fmt.Sprintf("%s-%s", echoSocket, app)
	echoPortsApp := fmt.Sprintf("%s-%s", echoPorts, app)
	dispatchProgApp := fmt.Sprintf("%s-%s", dispatchProg, app)
	dispatchLinkApp := fmt.Sprintf("%s-%s", dispatchLink, app)

	// Load eBPF program and maps
	var objs struct {
		Prog *ebpf.Program `ebpf:"echo_dispatch"`
		Sock *ebpf.Map     `ebpf:"echo_socket"`
		Port *ebpf.Map     `ebpf:"echo_ports"`
	}
	if err := spec.LoadAndAssign(&objs, nil); err != nil {
		panic(err)
	}
	defer objs.Prog.Close()
	defer objs.Port.Close()
	defer objs.Sock.Close()

	// Pin eBPF program and maps
	if err = objs.Prog.Pin(dispatchProgApp); err != nil {
		panic(err)
	}
	defer objs.Prog.Unpin()
	log.Printf("DEBUG Prog %v is pinned: %v\n", objs.Prog, objs.Prog.IsPinned())

	if err = objs.Sock.Pin(echoSocketApp); err != nil {
		panic(err)
	}
	defer objs.Sock.Unpin()
	log.Printf("DEBUG Map %s is pinned: %v\n", objs.Sock, objs.Sock.IsPinned())

	if err = objs.Port.Pin(echoPortsApp); err != nil {
		panic(err)
	}
	defer objs.Port.Unpin()
	log.Printf("DEBUG Map %s is pinned: %v\n", objs.Port, objs.Port.IsPinned())

	// Get FD from listener
	f, _ := listener.File()
	defer f.Close()
	fd := f.Fd()
	if err = objs.Sock.Put(socketKey, unsafe.Pointer(&fd)); err != nil {
		panic(err)
	}
	log.Printf("DEBUG:\tApp: %s\tlistener FD: %v\n", app, int(fd))

	for _, v := range ports {
		log.Printf("DEBUG:\tApp: %s\tadding port: %v\n", app, v)
		if err = objs.Port.Put(v, uint8(0)); err != nil {
			panic(err)
		}
	}

	// Get net namespace
	// Probably a better idea is to use github.com/vishvananda/netns
	netns, err := os.Open("/proc/self/ns/net")
	if err != nil {
		panic(err)
	}
	defer netns.Close()

	// Attach the network namespace to the link
	lnk, err := link.AttachNetNs(int(netns.Fd()), objs.Prog)
	if err != nil {
		panic(err)
	}

	// Pin link
	lnk.Pin(dispatchLinkApp)

	// Don't forget to close everything
	defer lnk.Unpin()
	defer lnk.Close()

	// Wait until done
	<-ctx.Done()
}
