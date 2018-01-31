COVERFILENAME:=cover
BINARY_NAME:=gozzmock

default: lint vet test

.PHONY: test
test:
	go test -coverprofile=$(COVERFILENAME).out `go list ./... | grep -v /vendor/`
	go tool cover -html=$(COVERFILENAME).out -o $(COVERFILENAME)_all.html
	rm $(COVERFILENAME).out

.PHONY: lint
lint:
	golint `go list ./... | grep -v /vendor/`

.PHONY: vet
vet:
	go vet `go list ./... | grep -v /vendor/`

.PHONY: clean
clean:
	go clean -i ./...
	rm -f $(BINARY_NAME)
	rm -f $(BINARY_NAME).exe

.PHONY: update
update:
	go get -u github.com/golang/dep/cmd/dep
	dep ensure

.PHONY: build-linux
build-linux:
	CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o $(BINARY_NAME) .

.PHONY: build-windows
build-windows:
	CGO_ENABLED=0 GOOS=windows go build -a -installsuffix cgo -o $(BINARY_NAME).exe .
