GOPATH := $(shell pwd)

all: tmugs


tmugs:
	GOPATH=$(GOPATH) go get $@
	GOPATH=$(GOPATH) go build $@

clean:
	GOPATH=$(GOPATH) go clean
	${RM} -r pkg/

.PHONY: tmugs
