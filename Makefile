MAKEFLAGS += --always-make

all: godogen godogen-lint custom-gcl

BUILDFLAGS ?= -ldflags="-s -w"

godogen:
	go build $(BUILDFLAGS) ./cmd/godogen

godogen-lint:
	go build $(BUILDFLAGS) ./cmd/godogen-lint

custom-gcl:
	make -C gcl-plugin custom-gcl
