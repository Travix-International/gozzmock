COVERFILENAME:=cover
BINARY_NAME:=gozzmock

default: lint vet test

.PHONY: test
test:
	go mod vendor
	go test -coverprofile=$(COVERFILENAME).out ./...
	go tool cover -html=$(COVERFILENAME).out -o $(COVERFILENAME)_all.html
	rm $(COVERFILENAME).out

.PHONY: lint
lint:
	golint `go list ./...`

.PHONY: vet
vet:
	go vet `go list ./...`

.PHONY: clean
clean:
	go clean -i ./...
	rm -f $(BINARY_NAME)
	rm -f $(BINARY_NAME).exe
	rm -f $(COVERFILENAME)*

.PHONY: update
update:
	go get -u 

.PHONY: build-linux
build-linux:
	GO111MODULE=on CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -mod vendor -o $(BINARY_NAME) .

.PHONY: build-windows
build-windows:
	GO111MODULE=on CGO_ENABLED=0 GOOS=windows go build -a -installsuffix cgo -mod vendor -o $(BINARY_NAME).exe .
