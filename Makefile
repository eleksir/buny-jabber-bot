#!/usr/bin/env gmake -f

CGO_ENABLED=0
BUILDOPTS=-ldflags="-s -w" -a -gcflags=all=-l -trimpath

all: clean build

build:
	go build ${BUILDOPTS} -o "buny-jabber-bot" collection.go types.go globals.go lib.go event_parser.go buny.go main.go

clean:
	go clean

upgrade:
	rm -rf vendor
	go get -d -u -t ./...
	go mod tidy
	go mod vendor

# vim: set ft=make noet ai ts=4 sw=4 sts=4:
