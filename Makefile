.PHONY: build run test stop clean

# main
BIN_DIR=bin
BIN_NAME=skproxy
PID=$(pidof bin/skproxy)

build:
	@echo -e "# skproxy build started"
	go build -o ${BIN_DIR}/${BIN_NAME} cmd/*.go

run: build
	@echo -e "\n# running skproxy"
	${BIN_DIR}/${BIN_NAME} &
	@echo -e "\n# wait until all listeners are ready"
	@sleep 0.5

test: run
	@echo -e "\n# executing test-target.sh"
	@./test/test-target.sh

stop:
	@echo -e "\n# kill skproxy"
	@pkill ${BIN_NAME}

clean: stop
	@echo -e "# clean skproxy"
	@rm ${BIN_DIR}/${BIN_NAME}

all: test clean
	@echo -e "# all done"
