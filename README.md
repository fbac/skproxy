# README - PLEASE READ

## Usage and Requirements

**IMPORTANT!**

- Check the Makefile carefully!
  - Always run the proxy with root user, otherwise the ebpf can't be loaded.
  - Make sure `echo_dispatch.bpf.o` exist in the bin folder.
- To run the proxy manually, run `sudo go run cmd/proxy/*.go` or `sudo make run`.
- To run a full e2e test, run `sudo make all`

### Versions tested

The proxy has been tested in the following OS, with the respective kernel and bpf tools versions.

Also, it's **required** to run it as **root** user.

The system must be able to run BPF programs.

#### **Ubuntu 22.04.1 LTS - Jammy**

- Kernel `5.15.0-47-generic`

- golang 1.18

- BPF packages:

```bash
binutils-bpf/jammy 2.38-2ubuntu1+3 amd64
bpftrace/jammy 0.14.0-1 amd64
libbpf-dev/jammy 1:0.5.0-1 amd64
libbpf0/jammy,now 1:0.5.0-1 amd64 [installed,automatic]
```

#### **Fedora release 36 (Thirty Six)**

- Kernel `5.18.17-200.fc36.x86_64`
- golang 1.18
- BPF packages:

```bash
libbpf-0.7.0-3.fc36.x86_64
libbpf-devel-0.7.0-3.fc36.x86_64
bpftrace-0.14.1-1.fc36.x86_64
bpftool-5.19.4-200.fc36.x86_64
```

##### Demonstration

- Create a new vm (ubuntu 22.04), and scan it with nmap

```bash
$ nmap -sT -p 1-10000 192.168.122.172                                                                                                                                                                                                          
Starting Nmap 7.92 ( https://nmap.org ) at 2022-09-16 00:51 CEST
Nmap scan report for 192.168.122.172
Host is up (0.00024s latency).
Not shown: 9999 closed tcp ports (conn-refused)
PORT   STATE SERVICE
22/tcp open  ssh

Nmap done: 1 IP address (1 host up) scanned in 0.57 seconds
```

- This is an ubuntu 22.04

```bash
root@proxy-last:~# cat /etc/os-release 
PRETTY_NAME="Ubuntu 22.04.1 LTS"
```

- Install required packages (commands extracted from Dockerfile, it includes more pkg than needed)

```bash
$ apt-get update && export DEBIAN_FRONTEND=noninteractive && apt-get install --no-install-recommends -y ca-certificates clang curl git llvm libelf-dev make netcat openssh-server openssl golang && rm -rf /var/lib/apt/lists/*

$ apt-get update && export DEBIAN_FRONTEND=noninteractive && apt-get install --no-install-recommends -y autoconf bison cmake dkms flex gawk gcc python3 rsync libiberty-dev libncurses-dev libpci-dev libssl-dev libudev-dev \
  && curl https://cdn.kernel.org/pub/linux/kernel/v5.x/linux-5.13.tar.gz | tar -xz \
  && make -C /linux-5.13 headers_install INSTALL_HDR_PATH=/usr \
  && make -C /linux-5.13/tools/lib/bpf install INSTALL_HDR_PATH=/usr \
  && make -C /linux-5.13/tools/bpf/bpftool install \
  && apt-get remove -y \
  autoconf bison cmake dkms flex gawk gcc python3 rsync \
  libiberty-dev libncurses-dev libpci-dev libssl-dev libudev-dev \
  && apt autoremove -y \
  && rm -rf /var/lib/apt/lists/* \
  && rm -rf /linux-5.13
```

- Clone the repository

```bash
git clone https://github.com/fbac/proxy-tcp-roundrobin/
Cloning into 'proxy-tcp-roundrobin'...
remote: Enumerating objects: 78, done.
```

- Run as a background process
  Notice the debug messages, these are for checking the backends ignored/added, and also the pinning of ebpf programs, maps and fd's

```bash
root@proxy-last:~/proxy-tcp-roundrobin# make run

-e # fly-proxy build started
go build -o bin/fly-proxy cmd/proxy/*.go
-e 
# running fly-proxy
bin/fly-proxy &
-e 
# wait until all listeners are ready
2022/09/15 22:58:35 started listener in :5001
2022/09/15 22:58:35 DEBUG Prog SkLookup(echo_dispatch)#17 is pinned: true
2022/09/15 22:58:35 DEBUG Map SockMap(echo_socket)#16 is pinned: true
2022/09/15 22:58:35 DEBUG Map Hash(echo_ports)#15 is pinned: true
2022/09/15 22:58:35 DEBUG: App: five-thousand listener FD: 14
2022/09/15 22:58:35 DEBUG: App: five-thousand adding port: 5200
2022/09/15 22:58:35 DEBUG: App: five-thousand adding port: 5300
2022/09/15 22:58:35 DEBUG: App: five-thousand adding port: 5400
2022/09/15 22:58:35 skipping unhealthy backend: bad.target.for.testing: lookup bad.target.for.testing: no such host
2022/09/15 22:58:35 started listener in :6001
2022/09/15 22:58:35 DEBUG Prog SkLookup(echo_dispatch)#26 is pinned: true
2022/09/15 22:58:35 DEBUG Map SockMap(echo_socket)#25 is pinned: true
2022/09/15 22:58:35 DEBUG Map Hash(echo_ports)#24 is pinned: true
2022/09/15 22:58:35 DEBUG: App: six-thousand listener FD: 23
2022/09/15 22:58:35 DEBUG: App: six-thousand adding port: 6200
2022/09/15 22:58:35 DEBUG: App: six-thousand adding port: 6300
2022/09/15 22:58:35 DEBUG: App: six-thousand adding port: 6400
2022/09/15 22:58:35 started listener in :7001
2022/09/15 22:58:35 DEBUG Prog SkLookup(echo_dispatch)#34 is pinned: true
2022/09/15 22:58:35 DEBUG Map SockMap(echo_socket)#33 is pinned: true
2022/09/15 22:58:35 DEBUG Map Hash(echo_ports)#32 is pinned: true
2022/09/15 22:58:35 DEBUG: App: seven-thousand listener FD: 31
2022/09/15 22:58:35 DEBUG: App: seven-thousand adding port: 7200
2022/09/15 22:58:35 DEBUG: App: seven-thousand adding port: 7300
2022/09/15 22:58:35 DEBUG: App: seven-thousand adding port: 7400
root@proxy-last:~/proxy-tcp-roundrobin#
```

- Scan ports again

```bash
[aranda@hyperion :: ~ ] $ nmap -sT -p 1-10000 192.168.122.172            

Starting Nmap 7.92 ( https://nmap.org ) at 2022-09-16 00:59 CEST
Nmap scan report for 192.168.122.172
Host is up (0.00023s latency).
Not shown: 9987 closed tcp ports (conn-refused)
PORT     STATE SERVICE
22/tcp   open  ssh
5001/tcp open  commplex-link
5200/tcp open  targus-getdata
5300/tcp open  hacl-hb
5400/tcp open  pcduo-old
6001/tcp open  X11:1
6200/tcp open  lm-x
6300/tcp open  bmc-grx
6400/tcp open  crystalreports
7001/tcp open  afs3-callback
7200/tcp open  fodms
7300/tcp open  swx
7400/tcp open  rtps-discovery

Nmap done: 1 IP address (1 host up) scanned in 0.60 seconds
```

- Also, on the proxy vm, some tcp connection debug messages will popup
  This is just for debugging purposes, here we can also check the load balancing is working properly in round robin fashion.

```bash
root@proxy-last:~/proxy-tcp-roundrobin# 
2022/09/15 22:59:31 proxying data from :7001 to tcp-echo.fly.dev:7001
2022/09/15 22:59:31 proxying data from :6001 to tcp-echo.fly.dev:6001
2022/09/15 22:59:31 proxying data from :5001 to tcp-echo.fly.dev:5001
2022/09/15 22:59:31 proxying data from :7001 to tcp-echo.fly.dev:7002
2022/09/15 22:59:31 proxying data from :7001 to tcp-echo.fly.dev:7001
2022/09/15 22:59:31 proxying data from :5001 to tcp-echo.fly.dev:5002
2022/09/15 22:59:31 proxying data from :7001 to tcp-echo.fly.dev:7002
2022/09/15 22:59:31 proxying data from :6001 to tcp-echo.fly.dev:6002
2022/09/15 22:59:31 proxying data from :6001 to tcp-echo.fly.dev:6001
2022/09/15 22:59:31 proxying data from :6001 to tcp-echo.fly.dev:6002
2022/09/15 22:59:31 proxying data from :5001 to tcp-echo.fly.dev:5001
2022/09/15 22:59:31 proxying data from :5001 to tcp-echo.fly.dev:5002
```

- Test manually with netcat from your localhost

```bash
[aranda@hyperion :: ~ ] $ echo "test" | nc -v -4 192.168.122.172 7200

Ncat: Version 7.92 ( https://nmap.org/ncat )
Ncat: Connected to 192.168.122.172:7200.
TEST
```

- The debug message will popup as well

```bash
2022/09/15 23:03:14 proxying data from :7001 to tcp-echo.fly.dev:7001
```
