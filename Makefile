ROOT_DIR := $(shell dirname $(realpath $(lastword $(MAKEFILE_LIST))))
VERSION=$(awk '/Version/ { gsub("\"", ""); print $NF }' ${ROOT_DIR}/application/constants.go)

branch = $(shell git rev-parse --abbrev-ref HEAD)
commit = $(shell git log --pretty=format:'%h' -n 1)
now = $(shell date "+%Y-%m-%d %T UTC%z")
compiler = $(shell go version)

BIN_DIR = $(ROOT_DIR)/bin
PICFIT_CONFIG_PATH ?= `pwd`/config.json
BIN = $(BIN_DIR)/picfit
SSL_DIR = $(ROOT_DIR)/ssl
APP_DIR = /go/src/github.com/thoas/picfit

export GO111MODULE=on

test: unit

ci:
	@(docker build -t picfit-ci -f Dockerfile.ci .)
	@(docker run --net=host --rm picfit-ci)

vendorize:
	find vendor/ -type f -not -path "*/.git*" -exec git add {} \;

run-server:
	@PICFIT_CONFIG_PATH=$(PICFIT_CONFIG_PATH) $(BIN)

serve:
	@modd

unit:
	go test -mod=vendor -v -cover ./...

all: picfit
	@(mkdir -p $(BIN_DIR))

build:
	@(echo "-> Compiling picfit binary")
	@(mkdir -p $(BIN_DIR))
	@(go build -mod=vendor -o $(BIN_DIR)/picfit ./cmd/picfit/main.go)
	@(echo "-> picfit binary created successfully")

install:
	@(test ! -f /usr/local/bin/picfit && ln -s $(BIN_DIR)/picfit /usr/local/bin/picfit)
	@(test ! -f /usr/local/bin/picfit.config.json && ln -s $(ROOT_DIR)/config.example.json /usr/local/bin/picfit.config.json)
	@(echo "-> create link from binary file to /usr/local/bin")
	@(test ! -d /home/picfit && mkdir -p /home/picfit)
	@(chmod -R 755 /home/picfit)
	@(echo "-> create directory /home/picfit with rights root:755")
	@(test -f /lib/systemd/system/picfit.service && rm /lib/systemd/system/picfit.service)
	@(cp -p -f $(ROOT_DIR)/picfit.service /lib/systemd/system/picfit.service)
	@(chown root:root /lib/systemd/system/picfit.service)
	@(chmod 644 /lib/systemd/system/picfit.service)
	@(systemctl daemon-reload)
	@(systemctl enable picfit.service)
	@(echo "-> add picfit.service and enable")
	@(echo "-> to start the service, enter `systemctl start picfit.service`")

update:
	@(test ! -f /usr/local/bin/picfit && ln -s $(BIN_DIR)/picfit /usr/local/bin/picfit)
	@(test ! -f /usr/local/bin/picfit.config.json && ln -s $(ROOT_DIR)/config.example.json /usr/local/bin/picfit.config.json)
	@(echo "-> create link from binary file to /usr/local/bin")
	@(systemctl stop picfit.service)
	@(test -f /lib/systemd/system/picfit.service && rm /lib/systemd/system/picfit.service)
	@(cp -p -f $(ROOT_DIR)/picfit.service /lib/systemd/system/picfit.service)
	@(chown root:root /lib/systemd/system/picfit.service)
	@(chmod 644 /lib/systemd/system/picfit.service)
	@(systemctl daemon-reload)
	@(systemctl enable picfit.service)
	@(echo "-> update picfit.service")

remove:
	@(test -f /usr/local/bin/picfit && rm /usr/local/bin/picfit)
	@(test -f /usr/local/bin/picfit.config.json && rm /usr/local/bin/picfit.config.json)
	@(echo "-> delete files on /usr/local/bin")
	@(systemctl stop picfit.service)
	@(systemctl disable picfit.service)
	@(test -f /lib/systemd/system/picfit.service && rm /lib/systemd/system/picfit.service)
	@(systemctl daemon-reload)
	@(echo "-> delete picfit.service")
	@(rm -R /var/log/picfit)
	@(rm -R $(BIN_DIR))
	@(echo "-> delete logs and build files")

format:
	@(go fmt ./...)
	@(go vet ./...)

build-static:
	@(echo "-> Creating statically linked binary...")
	mkdir -p $(BIN_DIR)
	go build -mod=vendor -ldflags "\
		-X github.com/thoas/picfit/constants.Branch=$(branch) \
		-X github.com/thoas/picfit/constants.Revision=$(commit) \
		-X 'github.com/thoas/picfit/constants.BuildTime=$(now)' \
		-X 'github.com/thoas/picfit/constants.Compiler=$(compiler)'" -a -installsuffix cgo -o $(BIN_DIR)/picfit ./cmd/picfit/main.go

docker-build-static: build-static


.PNONY: all test format

docker-build:
	@(echo "-> Preparing builder...")
	@(docker build -t picfit-builder -f Dockerfile.build .)
	@(mkdir -p $(BIN_DIR))
	@(docker run --rm -v $(BIN_DIR):$(APP_DIR)/bin picfit-builder)
