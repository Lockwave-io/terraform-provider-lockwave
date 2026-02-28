BINARY   := terraform-provider-lockwave
INSTALL_DIR := $(HOME)/.terraform.d/plugins/registry.terraform.io/lockwave-io/lockwave/0.1.0/$$(go env GOOS)_$$(go env GOARCH)

.PHONY: build test testacc lint fmt install clean

build:
	go build -o $(BINARY)

test:
	go test ./... -v

testacc:
	TF_ACC=1 go test ./... -v

lint:
	golangci-lint run

fmt:
	gofmt -s -w .

install: build
	mkdir -p $(INSTALL_DIR)
	mv $(BINARY) $(INSTALL_DIR)/$(BINARY)

clean:
	rm -f $(BINARY)
