# Design notes

- [Design notes](#design-notes)
  - [Usage - important to read](#usage---important-to-read)
  - [**1. A short summary of what you built, how it works, and how you landed on this design.**](#1-a-short-summary-of-what-you-built-how-it-works-and-how-you-landed-on-this-design)
  - [**2. How you might add hot config reloading that doesn't break existing connections if apps and targets change**](#2-how-you-might-add-hot-config-reloading-that-doesnt-break-existing-connections-if-apps-and-targets-change)
  - [**3. What might break under a production load? What needs to happen before your proxy is production ready?**](#3-what-might-break-under-a-production-load-what-needs-to-happen-before-your-proxy-is-production-ready)
  - [**4. If this were deployed to production at Fly.io, is there anything you could do with your proxy that would make our customers happy?**](#4-if-this-were-deployed-to-production-at-flyio-is-there-anything-you-could-do-with-your-proxy-that-would-make-our-customers-happy)
  - [**5. If you were starting over, is there anything you'd do differently?**](#5-if-you-were-starting-over-is-there-anything-youd-do-differently)
  - [**6. How would you make a global, clustered version of your proxy?**](#6-how-would-you-make-a-global-clustered-version-of-your-proxy)
  - [**7. What you did to add BPF steering**](#7-what-you-did-to-add-bpf-steering)
  - [**8. How you'd update the BPF maps when configuration changes**](#8-how-youd-update-the-bpf-maps-when-configuration-changes)

## Usage - important to read

**IMPORTANT!**

- Check the Makefile carefully! All the commands must be run as user.
- Always run the proxy with root user, otherwise the ebpf can't be loaded.
- Make sure `echo_dispatch.bpf.o` exist in the bin folder.
- To run the proxy manually, run `sudo go run cmd/proxy/*.go` or `sudo make run`.
- Don't forget to clean everything with: `sudo make clean` AND `sudo make clean-leftovers`
- To run a full e2e test, run `sudo make all`

## **1. A short summary of what you built, how it works, and how you landed on this design.**

The whole proxy I've built relies on the data structures ProxyMap and RoundRobinLB:

- **ProxyMap** holds a data structure which identifies each backend with the respective config.App structure
- **RoundRobinLB** represents a simple round robin loadbalancer, which simply returns the next backend in a slice of healthy backends.

The fundamental aspect of this assesment was proxying tcp request to a set of local ports, and balance them across a set of healthy backends.

With that premise in mind, my first approach was thinking about the datastructures I'd want to implement and how they interact with each other. Also, since this is a home assesment, those should be the minimal viable implementation, and I'm certain that this proxy have a million ways of being improved.

The main logic the code flows is as follows:

- A new proxy instance is built with a given and valid config.json. This instance is a singleton, as we don't want more than just one instance of proxy.
- The proxy is initialized with its own public method **InitializeProxy**, which in turns populates de ProxyMap instance (also a singleton).
- Once populated, the real fun can begin. We simply return all the frontend and backends for each application and call **prepareProxy**.
- **prepareProxy** is an intermediate step, which I introduced for middle factory work (so it can be extended in the future with more operations)
  - **prepareProxy** creates a new **LB** instance (one LB instance will be available per App)
  - finally, it creates a new goroutine of **proxy**.
- **proxy** should always be run as a goroutine, and makes sure that:
  - each listener for the App is up and listening.
  - incoming data is accepted indefinitely.
  - for each incoming data, it selects a healthy backend and pipes the inc data to the destination, and viceversa, returning the server's message.

There's more logic involved, such as; helper functions, LB's operations, etc.

## **2. How you might add hot config reloading that doesn't break existing connections if apps and targets change**

Probably, since the proxy simply pipes a file descriptor around, from src to dst, what I'd do is using SCM_RIGHTS.

This is the mechanism used by Haproxy nowadays, and basically what it does is allowing passing a file descriptor owned by one process to another. This keeps the status of the connections.

In addition to that, it would be nice to signal de current process to stop taking SYN requests, and pass them to the next process while we're reloading.

A rough implementation of this, in a mix of pseudocode and go, would be:

- Configuration change detected, we need to reload:

```go
 go func() {
  for cfg := range ch {
   fmt.Println("got config change:", cfg)
   pm.ReloadProxy(cfg)
  }
 }()
```

- Immediately after that:

```text
- fork the current process
- stop answering to SYN connections (this would require code rework), and redirect them to the new process once it's ready (a nice API for /health needed here)
- call SCM_RIGHTS and pass all the socket file descriptors we own to the next process
- monitor in the new process that we own the connections
- monitor in the old process that the connections are passed and we don't have any data left. Kill it.
```

Probably this process needs some deeper thought and refinement, so please treat it only as a first approach taken from the top of my mind!

## **3. What might break under a production load? What needs to happen before your proxy is production ready?**

This is a very rough implementation of a tcp proxy, it would be extremely slow for http/s requests, it doesn't support tcp over tls (with sni), etc... the use cases are pretty limited.

The data structures should be performant enough, since ProxyMap is just a map, involving high speed access and O(1) complexity. Also it's a singletong so the space it takes in memory is minimized.

On the other hand, the goroutines are pretty cheap in go, going down to 2KB as a minimum.

Without further testing I'd say this proxy *should be able* to accept medium workloads. But when it comes to high traffic or a data structure big enough probably the process would be slowed down significantly.

## **4. If this were deployed to production at Fly.io, is there anything you could do with your proxy that would make our customers happy?**

Well, a million things.

It comes to mind:

- Supporting different protocols, such as; http, https, tcp over tls using SNI. In the end, whatever protocol they might need to run apps.
- Seamless reloads, as discussed previously.
- More possibilities of configuring the proxy.
- Serve proxy metrics, maybe exposed using prometheus or opentelemetry. (requests,status, avg time, min time, max time, percentiles, ...)
- Different ways of load balancing: round robin, random, stickyness.
- Ability to terminate tls in the proxy (some sort of reencrypt) or in the backend (passthrough tls)
- Implement a /healthz API for the proxy, so it can be accesses programatically from a client, and act based on data.

## **5. If you were starting over, is there anything you'd do differently?**

If I were about to start over this assesment, and would have enough time to spend in polishing the sharp edges, I'd:

- Create a nice production-ready tree structure: internal/ folder holding internal packages.
- Divide everything by package: package proxy, package lb.
- Every package should expose only public methods and types, and be implemented in the most generic and idiomatic way.
- Finish the implementation of `func (pm ProxyMap) ReloadProxy(cfg config.Config) {}`.
- Parametrize the healthCheck in config.json, so every backend has it's own healthchecks and MaxTimeouts.
- Call the healthCheck inside the `proxy()` loop with a time.Ticker, so for long lasting connections (implement a method to check if some conn is long lasting) we can check the backend health.
- Following the last bullet point, if a backend is unhealthy in the middle of a conn, implement a method so the LB can be called immediately.
- Use a context for the whole proxy, starting at `InitializeProxy() -> doProxy() -> proxy()`, so a proxy() function and all its goroutines is killed based on a signal (if something really bad happens)
- New struct Proxy which holds a ctx, LoadBalancer struct and ProxyMap struct
  - The ctx should be used in the function proxy(), so every goroutine inside the function rungs in the same context.

## **6. How would you make a global, clustered version of your proxy?**

Some different approachs come to mind:

- **First approach**:
  
  Leave the codebase as it is, and rely on third services for the clustering. For example keepalived. This is probably my preferred approach as it's based on UNIX philosophy. (small services that do X only, but they do it pretty damn well!)

  By running keepalived a virtual IP can be balanced through a set of N frontends, where the proxy is running and listening on X ports.

  The dns pointing to the platform would be pointing to this virtual IP, and keepalived on the background would be checking the health of the different proxy frontends.

  On failure, keepalived balances to the next healthy proxy and keep serving requests.

- **Second approach**:
  
  Use a distributed and really fast database to store proxy mappings, which also allows atomic operations, such as redis, scyllaDB or mongodb.

  With this approach a set of N proxy can run at the same time, accessing and updating the database to build the internal data structure, and proxy requests adequately.

  Pros: There can be multiple DNS pointing to different frontends, as they all provide the same backends.

  Downside: maintaing db's and all the problems they introduce. (data access latency, etc)

## **7. What you did to add BPF steering**

I used the code [echo_dispatch.bpf.c](https://github.com/jsitnicki/ebpf-summit-2020/blob/master/echo_dispatch.bpf.c) published by Jakub Sitnicki, as it works as expected for our use case.

After that, I just played a bit with it and the eBPF library provided by Cillium, specifically the ebpf and ebpf/link, as along the way we'll have to pin the eBPF program, maps and links.

Then, the rest is straight forward and it's documented in ebpf.go:

- Load the ELF binary from path.
- Load and pin the program and the maps, as they have to be pinned in order to access them.
- Pass to this function the listener file, so we can access the fd.
- Add the socket fd to the sockMap (BPF_MAP_TYPE_SOCKMAP).
- Add all the ports to the portMap (BPF_MAP_TYPE_HASH), which is a hash so we have to add the port as a key, and the value is 0.
- Then, get the current network namespace:
  - Create a eBPF link to our program, and attach the network namespace to it.
- Finally, don't forget to Unpin and Close everything, so we don't left anything when returning from this function.
- Also, this function has to run in a context for the same reason.

## **8. How you'd update the BPF maps when configuration changes**

In order to update the BPF maps when the configuration changes, there's a method of updating the maps instead of deleting them.

This update gets flags as parameters:

```go
const (
// UpdateAny creates a new element or update an existing one.
UpdateAny MapUpdateFlags = iota
// UpdateNoExist creates a new element.
UpdateNoExist MapUpdateFlags = 1 << (iota - 1)
// UpdateExist updates an existing element.
UpdateExist
// UpdateLock updates elements under bpf_spin_lock.
UpdateLock
)
```

So when the configuration changes, we would simply update the maps with UpdateAny
