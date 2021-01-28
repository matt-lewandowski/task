SHELL := /bin/bash

.PHONY: coverage
coverage:
	go get -t -v ./...
	go test ./... -coverprofile=coverage.txt -covermode=atomic
	bash <(curl -s https://codecov.io/bash)
	go tool cover -func coverage.txt
	go tool cover -html=coverage.txt -o cover.html
	open cover.html
