BUILD_PATH := ./artifacts
LINUX_BUILD_PATH = $(BUILD_PATH)/linux/gva
LINUX_ARM_BUILD_PATH = $(BUILD_PATH)/arm/gva
WINDOWS_BUILD_PATH = $(BUILD_PATH)/windows/gva.exe
MAC_BUILD_PATH = $(BUILD_PATH)/darwin/gva
VERSION?=$(shell gogitver)
COMMIT_HASH:=$(shell git rev-parse HEAD)
BUILD_DATE:=$(shell date +%Y-%m-%dT%T%z)
BUILD_FLAGS:=\
	-X github.com/thzinc/gunviolencearchive-client/package/cmd/cli.version=$(VERSION) \
	-X github.com/thzinc/gunviolencearchive-client/package/cmd/cli.commit=$(COMMIT_HASH) \
	-X github.com/thzinc/gunviolencearchive-client/package/cmd/cli.date=$(BUILD_DATE)

.PHONY: clean
clean:
	rm -Rf ./artifacts

.PHONY: test
test:
	go test -v ./...

.PHONY: build
build: clean test
	mkdir -p artifacts/linux artifacts/arm artifacts/windows artifacts/darwin
	GOOS=linux GOARCH=amd64 go build -o $(LINUX_BUILD_PATH) -ldflags "$(BUILD_FLAGS)" cmd/gva/main.go
	GOOS=linux GOARCH=arm go build -o $(LINUX_ARM_BUILD_PATH) -ldflags "$(BUILD_FLAGS)" cmd/gva/main.go
	GOOS=darwin GOARCH=amd64 go build -o $(MAC_BUILD_PATH) -ldflags "$(BUILD_FLAGS)" cmd/gva/main.go
	GOOS=windows GOARCH=amd64 go build -o $(WINDOWS_BUILD_PATH) -ldflags "$(BUILD_FLAGS)" cmd/gva/main.go

package: build
	cd $(BUILD_PATH)/darwin && tar -zcvf ../darwin.tar.gz *
	cd $(BUILD_PATH)/linux && tar -zcvf ../linux.tar.gz *
	cd $(BUILD_PATH)/arm && tar -zcvf ../arm.tar.gz *
	cd $(BUILD_PATH)/windows && zip -r ../windows.zip *
	rm -R $(BUILD_PATH)/darwin $(BUILD_PATH)/linux $(BUILD_PATH)/arm $(BUILD_PATH)/windows

generate:
	go run internal/cobraDocs.go