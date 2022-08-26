self: super: let
  inherit (self.pkgs) callPackage;
in {
  # https://github.com/ava-labs/avalanchego/
  avalanchego = callPackage ./avalanchego {};

  # https://github.com/ava-labs/avalanche-cli
  avalanche-cli = callPackage ./avalanche-cli {};

  # https://github.com/ava-labs/avalanche-network-runner
  avalanche-network-runner = callPackage ./avalanche-network-runner {};

  # https://github.com/ava-labs/subnet-cli
  avalanche-subnet-cli = callPackage ./avalanche-subnet-cli {};
}
