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

# Start an Avalanche local test network via avalanche-cli:
#
#   - https://docs.avax.network/subnets/create-a-local-subnet
#   - https://docs.avax.network/quickstart/create-a-local-test-network
#
#   The tool will try to download the avalanchego & subnet-evm in following path: ~/.avalanche-cli
#   as nix, doesn't support unpatched binaries, we rely on using symlinks from the current store.
avax-up SUBNET='evm': avax-cli-create-dir avax-cli-link-binaries
  avalanche-cli subnet create {{SUBNET}} || true
  avalanche-cli subnet deploy {{SUBNET}} || true

# Destroys a local Avalanche test network via avalanche-cli
avax-down SUBNET='evm':
  avalanche-cli subnet delete {{SUBNET}} || true
  avalanche-cli network stop || true
  just avax-cli-delete-dir

# Creates necessary dir used by avalanche-cli
avax-cli-create-dir:
  [ ! -d $AVALANCHE_CLI_DIR ] && mkdir -p ${AVALANCHE_CLI_DIR}/bin/{avalanchego-v${AVALANCHEGO_VERSION}/plugins,subnet-evm-v${AVALANCHE_SUBNET_EVM_VERSION}}/

# Creates symbolic links to avalanchego and subnet-evm pointing to our nix-store,
# that way we avoid downloading the binaries from avalanche-cli
avax-cli-link-binaries:
  ln -s "${AVALANCHEGO_EXEC_PATH}" ~/.avalanche-cli/bin/avalanchego-v${AVALANCHEGO_VERSION}/avalanchego || true
  ln -s "${AVALANCHE_SUBNET_EVM_PATH}" ~/.avalanche-cli/bin/subnet-evm-v${AVALANCHE_SUBNET_EVM_VERSION}/subnet-evm || true

# Deletes the local directory that avalanche-cli uses
avax-cli-delete-dir:
  rm -rf $AVALANCHE_CLI_DIR || true

dev: avax-up

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