.PHONY: install

install:
	go get ./...
	go build
	mv -v gcaler $(GOPATH)/bin
	cp -v config.json client_secret.json $(GOPATH)/bin
