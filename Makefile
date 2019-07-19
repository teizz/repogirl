MKFILE_PATH := $(abspath $(lastword $(MAKEFILE_LIST)))
PACKAGE := $(notdir $(patsubst %/,%,$(dir $(MKFILE_PATH))))

BUILD_TIME=`date +%FT%T%z`
VERSION	= $(shell git rev-parse --short HEAD)
RELEASE_DIR  = $(dir $(MKFILE_PATH))
LDFLAGS = -ldflags "-X main.version=$(VERSION) -X main.buildtime=$(BUILD_TIME) -s -w"
GCFLAGS = -gcflags "-trimpath $(GOPATH)"

.PHONY: clean depend nothing all
.DEFAULT_GOAL=nothing

all: $(PACKAGE)

$(PACKAGE):
	@echo "### GO BUILD binaries for $(PACKAGE)-$(VERSION)"
	@env CGO_ENABLED=0 go build -a $(GCFLAGS) $(LDFLAGS) -a -o $(PACKAGE)

depend:
	@echo "### GO GET dependencies for $(PACKAGE)-$(VERSION)"
	@go get -u github.com/thoas/stats
	@go get -u github.com/sirupsen/logrus

clean:
	@echo "### DELETE binaries for $(PACKAGE)"
	@find $(RELEASE_DIR) -name $(PACKAGE)-* -delete
	@rm -f $(RELEASE_DIR)$(PACKAGE)

nothing:
	@echo "Doing nothing (use 'all' target to build package)"
