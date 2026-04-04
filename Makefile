# ccbox Makefile

INSTALL_DIR  ?= /usr/local/bin

BINARIES := ccbox ccptproxy ccclipd ccdebug

DOCKER_IMAGE := ghcr.io/ccdevkit/ccbox-base:dev

.PHONY: build install docker run clean test $(BINARIES)

## build: Compile all binaries into bin/
build: $(BINARIES)

$(BINARIES):
	go build -o bin/$@ ./cmd/$@/

## install: Copy ccbox binary to INSTALL_DIR
install: build
	install -d $(INSTALL_DIR)
	install -m 755 bin/ccbox $(INSTALL_DIR)/ccbox

## docker: Build base Docker image
docker:
	docker build -t $(DOCKER_IMAGE) .

## run: Run ccbox locally (use ARGS= to pass arguments)
run: build
	./bin/ccbox $(ARGS)

## test: Run all tests
test:
	go test ./...

## clean: Remove build artifacts
clean:
	rm -rf bin/
