#!/usr/bin/env gmake -f

BUILDOPTS=-ldflags="-s -w" -a -gcflags=all=-l -trimpath -pgo=auto

all: clean build

build:
ifeq ($(OS),Windows_NT)
# powershell
ifeq ($(SHELL),sh.exe)
	SET CGO_ENABLED=0
	go build ${BUILDOPTS} -o "buny-jabber-bot" collection.go types.go globals.go lib.go event_parser.go buny.go main.go
else
# jetbrains golang
	CGO_ENABLED=0
	go build ${BUILDOPTS} -o "buny-jabber-bot" collection.go types.go globals.go lib.go event_parser.go buny.go main.go
endif
# bash/git bash
else
	CGO_ENABLED=0 go build ${BUILDOPTS} -o "buny-jabber-bot" collection.go types.go globals.go lib.go event_parser.go buny.go main.go
endif

clean:
	go clean

upgrade:
ifeq ($(OS),Windows_NT)
# jetbrains golang, powershell
ifeq ($(SHELL),sh.exe)
	if exist vendor del /F /S /Q vendor >nul
# git bash case
else
	RM -r vendor
endif
else
	RM -r vendor
endif
	go get -d -u -t ./...
	go mod tidy
	go mod vendor

# vim: set ft=make noet ai ts=4 sw=4 sts=4:
