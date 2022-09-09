.PHONY: build run test stop clean

BIN_DIR=bin
BIN_NAME=fly-proxy
PID=$(pidof bin/fly-proxy)

build:
	@echo -e "# fly-proxy build started"
	go build -o ${BIN_DIR}/${BIN_NAME} cmd/proxy/main.go

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
	@kill ${BIN_NAME}

clean: stop
	@echo -e "# clean fly-proxy"
	@rm ${BIN_DIR}/${BIN_NAME}

all: test clean
	@echo -e "# all done"
