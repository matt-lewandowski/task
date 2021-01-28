SHELL := /bin/bash

.PHONY: coverage
coverage:
	go get -t -v ./...
	go test ./... -coverprofile=coverage.txt -covermode=atomic
	bash <(curl -s https://codecov.io/bash)
