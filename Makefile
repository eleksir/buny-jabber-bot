#!/usr/bin/env gmake -f

BUILDOPTS=-ldflags="-s -w" -a -gcflags=all=-l -trimpath -pgo=auto
FILELIST=collection.go types.go globals.go lib.go event_parser.go buny.go commands.go main.go

BINARY=buny-jabber-bot

# На windows имя бинарника может зависеть не только от платформы, но и от выранной цели, для linux-а суффикс .exe
# не нужен
ifeq ($(OS),Windows_NT)
ifeq ($(strip $(GOOS)),)
BINARY=buny-jabber-bot.exe
endif
endif

# Если мы собираем бинарь под windows не на windows, то надо сделать бинарь с суффиксом .exe
ifeq ($(strip $(GOOS)),windows)
BINARY=buny-jabber-bot.exe
endif

# Явно определяем символ новой строки, чтобы избежать неоднозначности на windows
define IFS

endef

# Используем классические таргеты, где первый встречаемый является таргетом по-умолчанию
all: clean build


build:
# Ну и дальше просто билдим бинарник средствами гошки
ifeq ($(OS),Windows_NT)
# вариант с powershell на windows
ifeq ($(SHELL),sh.exe)
	SET CGO_ENABLED=0
	go build ${BUILDOPTS} -o ${BINARY} ${FILELIST}
else
# вариант с jetbrains golang на windows
	CGO_ENABLED=0
	go build ${BUILDOPTS} -o ${BINARY} ${FILELIST}
endif
# вариант с bash/git (windows) и bash (linux)
else
	CGO_ENABLED=0 go build ${BUILDOPTS} -o ${BINARY} ${FILELIST}
endif


# Удаляем бинарник средствами go
clean:
	go clean


# Служебный таргет, для целей разработки. Обновляет завендоренные либы, брутальным образом.
upgrade:
ifeq ($(OS),Windows_NT)
# вариант с jetbrains golang на windows или powershell на windows
ifeq ($(SHELL),sh.exe)
	if exist vendor del /F /S /Q vendor >nul
# вариант с git/bash на windows
else
	$(RM) -r vendor
endif
# вариант с bash на linux
else
	$(RM) -r vendor
endif
	go get -d -u -t ./...
	go mod tidy
	go mod vendor

# vim: set ft=make noet ai ts=4 sw=4 sts=4:
