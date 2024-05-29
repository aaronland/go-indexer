GOMOD=$(shell test -f "go.work" && echo "readonly" || echo "vendor")
LDFLAGS=-s -w

cli:
	go build -mod $(GOMOD) -ldflags="$(LDFLAGS)" -o bin/index cmd/index/main.go
	go build -mod $(GOMOD) -ldflags="$(LDFLAGS)" -o bin/search cmd/search/main.go
