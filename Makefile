.PHONY: build test vuln install check clean

build:
	CGO_ENABLED=0 go build -o muxc .

test:
	CGO_ENABLED=0 go test -timeout 120s ./...

vuln:
	govulncheck ./...

install: build
	cp muxc $(HOME)/.local/bin/muxc

check: build test vuln

clean:
	rm -f muxc
