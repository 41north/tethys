# just is a handy way to save and run project-specific commands.
#
# https://github.com/casey/just

# list all tasks
default:
  just --list

# Regenerate proto bindings
protogen:
  protoc -I . --go_out=:$GOPATH/src api/protobuf/*.proto

# Format the code
fmt:
  treefmt
alias f := fmt

# Start IDEA in this folder
idea:
  nohup idea-ultimate . > /dev/null 2>&1 &

# Start VsCode in this folder
code:
  code .

# Builds a concrete binary using go
go-build PROGRAM:
  go build cmd/{{PROGRAM}}/{{PROGRAM}}.go

# Builds all binaries using go
go-build-all: (go-build "proxy") (go-build "sidecar")

# Builds a concrete program using nix
nix-build PROGRAM:
  nix build .#{{PROGRAM}}

# Builds all nix output targets
nix-build-all: (nix-build-docker-images) (nix-build-binaries)

# Builds only the binaries using nix
nix-build-binaries:
  nix build -o result-tethys-proxy .#tethys-proxy
  nix build -o result-tethys-sidecar .#tethys-sidecar

# Builds the docker images and loads them into docker
nix-build-docker-images:
  nix build -o result-tethys-proxy-docker .#tethys-proxy-docker && docker load < result-tethys-proxy-docker
  nix build -o result-tethys-sidecar-docker .#tethys-sidecar-docker && docker load < result-tethys-sidecar-docker

# Checks the source with nix
nix-check:
  nix flake check

# Cleans all outputs
clean:
  rm -rf result*