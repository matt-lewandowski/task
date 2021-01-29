SHELL := /bin/bash

.PHONY: coverage
coverage:
	go test ./... -coverprofile coverage.out
	go tool cover -func coverage.out
	go tool cover -html=coverage.out -o cover.html
	open cover.html
