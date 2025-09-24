.PHONY: build test pr

build:
	multimod build

test:
	multimod test

lint:
	multimod lint

deps:
	multimod update

pr:
	go install cloudeng.io/go/cmd/goannotate@latest \
		cloudeng.io/go/cmd/gousage@latest \
		cloudeng.io/go/cmd/gomarkdown@latest
	go install golang.org/x/tools/cmd/goimports@latest
	multimod usage annotate markdown
