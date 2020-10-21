GOCMD=go
GOBUILD=$(GOCMD) build
BINARY=distributed-service-discovery

build:
	$(GOBUILD) -o $(BINARY)

install: build
	sudo cp $(BINARY) /usr/local/bin
	sudo cp ./distributed-service-discovery.service /etc/systemd/system

run:
	sudo -E ./distributed-service-discovery -d -p 22 -o mysql