.PHONY: build run test stop clean

# main
BIN_DIR=bin
BIN_NAME=fly-proxy
PID=$(pidof bin/fly-proxy)

# ebpf
KERNEL := linux-5.9.1
KERNEL_INC := $(KERNEL)/usr/include
LIBBPF_INC := $(KERNEL)/tools/lib
LIBBPF_LIB := $(KERNL)/tools/lib/bpf/libbpf.a

CC := clang
CFLAGS := -g -O2 -Wall -Wextra
CPPFLAGS := -I$(KERNEL_INC) -I$(LIBBPF_INC)

##########
# golang #
##########

build: bin/echo_dispatch.bpf.o
	@echo -e "# fly-proxy build started"
	go build -o ${BIN_DIR}/${BIN_NAME} cmd/proxy/*.go

run: build
	@echo -e "\n# running fly-proxy"
	${BIN_DIR}/${BIN_NAME} &
	@echo -e "\n# wait until all listerners are ready"
	@sleep 0.5

test: run
	@echo -e "\n# executing test-target.sh"
	@./test-target.sh

stop:
	@echo -e "\n# kill fly-proxy"
	@pkill ${BIN_NAME}

clean: stop
	@echo -e "# clean fly-proxy"
	@rm ${BIN_DIR}/${BIN_NAME}

clean-leftovers: clean
	@echo -e "# cleaning fly-proxy and ebpf leftovers"
	@rm /sys/fs/bpf/*-five-thousand
	@rm /sys/fs/bpf/*-six-thousand
	@rm /sys/fs/bpf/*-seven-thousand

all: test clean clean-leftovers
	@echo -e "# all done"

########
# ebpf #
########

bin/echo_dispatch.bpf.o: src/ebpf/echo_dispatch.bpf.c $(KERNEL_INC) $(LIBBPF_INC) $(LIBBPF_LIB)
	$(CC) $(CPPFLAGS) $(CFLAGS) -target bpf -c -o $@ $<

$(KERNEL).tar.xz:
	curl -O https://cdn.kernel.org/pub/linux/kernel/v5.x/$(KERNEL).tar.xz

# Unpack kernel sources
$(KERNEL): $(KERNEL).tar.xz
	tar axf $<

# Install kernel headers
$(KERNEL_INC): $(KERNEL)
	make -C $< headers_install INSTALL_HDR_PATH=$@

# Build libbpf to generate helper definitions header
$(LIBBPF_LIB): $(KERNEL)
	make -C $</tools/lib/bpf

# Build bpftool
bpftool: $(KERNEL)
	make -C $</tools/bpf/bpftool
	cp $</tools/bpf/bpftool/bpftool $@

.PHONY: clean
clean-ebpf:
	rm -f bin/echo_dispatch.bpf.o

.PHONY: dist-clean
dist-clean: clean
	rm -rf $(KERNEL) $(KERNEL).tar.xz