# README - PLEASE READ!

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
