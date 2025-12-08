CI_RAW_URL = https://raw.githubusercontent.com/yeencloud/dpl-ci/refs/heads/main

update:
	mkdir -p .github/scripts
	curl -O $(CI_RAW_URL)/makefile \
         -O $(CI_RAW_URL)/.golangci.yml \
		 -O $(CI_RAW_URL)/.github/scripts/openapi-generate.sh \
		 -O $(CI_RAW_URL)/.github/config/oapi-codegen.yml
lint:
	golangci-lint run ./...

test:
	go test -race -v ./...

openapi:
	go install github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@latest

air:
	go tool air --build.cmd "go build -o $(notdir $(CURDIR))exe cmd/main.go" --build.entrypoint "./$(notdir $(CURDIR))exe" --build.include_file ".env" --build.include_dir "../../libs"