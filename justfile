run *ARGS:
  @go run . -debug -config odyssey.cfg {{ ARGS }}

build:
  @go build .

fmt:
  @go fmt

test:
  @go test ./...
