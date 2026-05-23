run *ARGS:
  @go run . --config odyssey.cfg {{ ARGS }}

build:
  @go build .

fmt:
  @go fmt

test:
  @go test ./... -v
