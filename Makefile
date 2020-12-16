DIR := $(PWD)
GIT_ORIGIN=origin
BUILD := $(shell git rev-parse --short HEAD)
PROJECTNAME := $(shell basename "$(PWD)")

GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOPATH:=$(shell $(GOCMD) env GOPATH 2> /dev/null)
GOOS:=$(shell $(GOCMD) env GOOS 2> /dev/null)
GOARCH:=$(shell $(GOCMD) env GOARCH 2> /dev/null)

VERSION=$(shell git describe --tags --abbrev=0)
DATE=$(shell date +%FT%T%z)
VERSION_FORMATTED=$(shell git describe --tags --abbrev=0 | sed 's/\./-/g')

all: check-go install test pre-build build

check-go:
ifndef GOPATH
	$(error "go is not available, please install")
endif

.PHONY: clean
clean:
	$(GOCLEAN)
	rm -rf dist/
	rm -rf builds/

deps:
	${GOCMD} get -v

embed-assets:
	go-embed -compress=false -input binaries/ -output assets/main.go

install: deps embed-assets

test: 
	$(GOTEST) -v ./...

pre-build: embed-assets

reset-version:
	printf $(VERSION) > ./binaries/.version


# Build
build: local-build
local-build: pre-build
	GO111MODULE=on CGO_ENABLED=0 GOOS=$(GOOS) GOARCH=$(GOARCH) $(GOCMD) build -o builds/${PROJECTNAME}-${VERSION_FORMATTED}-${GOOS} -v
	@echo "> Build complied to: "
	@echo "builds/${PROJECTNAME}-${VERSION_FORMATTED}-${GOOS}"

local-build-linux: pre-build
	GO111MODULE=on CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GOCMD) build -o builds/${PROJECTNAME}-${VERSION_FORMATTED}-linux -v
	@echo "> Build complied to: "
	@echo "builds/${PROJECTNAME}-${VERSION_FORMATTED}-linux"

local-build-darwin: pre-build
	GO111MODULE=on CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 $(GOCMD) build -o builds/${PROJECTNAME}-${VERSION_FORMATTED}-darwin -v
	@echo "> Build complied to: "
	@echo "builds/${PROJECTNAME}-${VERSION_FORMATTED}-darwin"

# Release
release-test: pre-build
	goreleaser release --skip-publish --skip-sign --rm-dist

# Ref: https://github.com/fmahnke/shell-semver
release-patch:
	$(eval VERSION=$(shell ${PWD}/increment_version.sh -p $(shell git describe --abbrev=0 --tags)))
	git tag $(VERSION)
	goreleaser release --skip-publish --skip-sign --rm-dist
	printf $(VERSION) > ./binaries/.version
	git add ./binaries/.version && git commit -m "Bumping version"
	git tag $(VERSION) -f	
	git push $(GIT_ORIGIN) main --tags

release-minor:
	$(eval VERSION=$(shell ${PWD}/increment_version.sh -m $(shell git describe --abbrev=0 --tags)))
	git tag $(VERSION)
	goreleaser release --skip-publish --skip-sign --rm-dist
	printf $(VERSION) > ./binaries/.version
	git add ./binaries/.version && git commit -m "Bumping version"
	git tag $(VERSION) -f	
	git push $(GIT_ORIGIN) main --tags

release-major:
	$(eval VERSION=$(shell ${PWD}/increment_version.sh -M $(shell git describe --abbrev=0 --tags)))
	git tag $(VERSION)
	goreleaser release --skip-publish --skip-sign --rm-dist
	printf $(VERSION) > ./binaries/.version
	git add ./binaries/.version && git commit -m "Bumping version"
	git tag $(VERSION) -f	
	git push $(GIT_ORIGIN) main --tags
