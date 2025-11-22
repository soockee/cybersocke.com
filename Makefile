TEMPLCMD=templ
# Go parameters
GOCMD=go
GOBUILD=$(TEMPLCMD) generate & $(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
SWAGCMD=swag
SWAGFLAGS=--generalInfo ./api.go --output ./docs --parseInternal

# Binary name
BINARY_NAME=main

all: test build

build: swag
	$(GOBUILD) -o $(BINARY_NAME) -v

test:
	$(GOTEST) -v ./

clean:
	$(GOCLEAN)
	rm -f $(BINARY_NAME)

run: swag
	$(GOBUILD) -o $(BINARY_NAME) -v ./
	./$(BINARY_NAME)

swag:
	$(SWAGCMD) init $(SWAGFLAGS)
