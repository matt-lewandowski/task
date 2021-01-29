.PHONY: coverage
coverage:
	go test ./... -coverprofile coverage.out
	go tool cover -func coverage.out
	go tool cover -html=coverage.out -o cover.html
	open cover.html

.PHONY: clockMock
clockMock:
	go run github.com/vektra/mockery/cmd/mockery \
	-dir $(PWD)/internal/limiter/clock \
 	-name Clock \
 	-case=underscore \
 	-output $(PWD)/internal/limiter/mock \
 	-outpkg mock\

.PHONY: limiterMock
limiterMock:
	go run github.com/vektra/mockery/cmd/mockery \
	-dir $(PWD)/internal/limiter \
 	-name Limiter \
 	-case=underscore \
 	-output $(PWD)/internal/limiter/mock \
 	-outpkg mock\

# Create all of the mock files
.PHONY: mocks
mocks: clockMock limiterMock
