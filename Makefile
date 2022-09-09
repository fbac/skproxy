.PHONY: build run test stop clean

BIN_DIR=bin
BIN_NAME=fly-proxy
PID=$(pidof bin/fly-proxy)

build:
	go build -o ${BIN_DIR}/${BIN_NAME} cmd/proxy/main.go

run: build
	${BIN_DIR}/${BIN_NAME} &

test: run
	./test-target.sh

stop:
	kill ${BIN_NAME}

clean: stop
	rm ${BIN_DIR}/${BIN_NAME}
