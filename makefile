CI_RAW_URL = https://raw.githubusercontent.com/yeencloud/dpl-ci/refs/heads/main

update:
	curl -O $(CI_RAW_URL)/makefile \
         -O $(CI_RAW_URL)/.golangci.yml

lint:
	golangci-lint run ./...

test:
	go test -race -v ./...
