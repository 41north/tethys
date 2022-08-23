{
  description = "web3 a smart load balancer for the blockchain";

  nixConfig = {
    substituters = [
      "https://cache.nixos.org"
      "https://nix-community.cachix.org"
    ];
    trusted-public-keys = [
      "cache.nixos.org-1:6NCHdD59X431o0gWypbMrAURkbJ16ZPMQFGspcDShjY="
      "nix-community.cachix.org-1:mB9FSh9qf2dCimDSUo8Zy7bkq5CX+/rkCWyvRCYg3Fs="
    ];
  };

  inputs = {
    devshell = {
      url = "github:numtide/devshell";
      inputs.nixpkgs.follows = "nixpkgs";
      inputs.flake-utils.follows = "flake-utils";
    };
    flake-utils.url = "github:numtide/flake-utils";
    nixpkgs.url = "github:nixos/nixpkgs/nixpkgs-unstable";
    go_1_19.url = "github:qowoz/nixpkgs/go119";
    pre-commit-hooks = {
      url = "github:cachix/pre-commit-hooks.nix";
      inputs.flake-utils.follows = "flake-utils";
    };
  };

  outputs = {
    self,
    devshell,
    nixpkgs,
    flake-utils,
    pre-commit-hooks,
    ...
  } @ inputs: let
    inherit (flake-utils.lib) eachSystem flattenTree mkApp;
  in
    eachSystem
    [
      "aarch64-linux"
      "aarch64-darwin"
      "x86_64-darwin"
      "x86_64-linux"
    ]
    (system: let
      pkgs = import nixpkgs {
        inherit system;
        overlays = [
          devshell.overlay
          (import ./nix/overlays)
        ];
      };
      pkgs_go119 = import inputs.go_1_19 {
        inherit system;
      };

      inherit (pkgs) dockerTools;
      inherit (pkgs_go119) buildGoModule;
      inherit (pkgs.stdenv) isLinux;
      inherit (pkgs.lib) lists fakeSha256 licenses platforms;
      inherit (import ./nix/lib {pkgs = pkgs_go119;}) pkgWithCategory buildGoApp;

      # devshell command categories
      dev = pkgWithCategory "dev";
      linter = pkgWithCategory "linters";
      formatter = pkgWithCategory "formatters";
      util = pkgWithCategory "utils";
    in rec {
      # nix build .#<app>
      packages = let
        vendorSha256 = "sha256-7MpZnG+KtBuGpcWy/01sx+m6h3AB8qCfH88g6z948dU=";
      in
        flattenTree rec {
          tethys-proxy = buildGoApp {
            inherit vendorSha256;
            name = "tethys-proxy";
            src = self;
            package = "cmd/proxy";
          };
          tethys-proxy-docker = dockerTools.buildLayeredImage {
            name = "41north/${tethys-proxy.name}";
            tag = "dev";
            maxLayers = 15;
            created = "now";
            config.Entrypoint = ["${tethys-proxy}/bin/proxy"];
          };
          tethys-sidecar = buildGoApp {
            inherit vendorSha256;
            name = "tethys-sidecar";
            src = self;
            package = "cmd/sidecar";
          };
          tethys-sidecar-docker = dockerTools.buildLayeredImage {
            name = "41north/${tethys-sidecar.name}";
            tag = "dev";
            maxLayers = 15;
            created = "now";
            config.Entrypoint = ["${tethys-sidecar}/bin/sidecar"];
          };
        };

      # nix develop
      devShell = pkgs.devshell.mkShell {
        env =
          [
            # disable CGO for now
            {
              name = "CGO_ENABLED";
              value = "0";
            }
          ]
          ++ lists.optionals isLinux [
            # override default behavior for avalanche-cli
            {
              name = "AVALANCHEGO_EXEC_PATH";
              value = "${pkgs.avalanchego}/bin/avalanchego";
            }
            # override behavior for avalanche-cli
            {
              name = "AVALANCHE_SUBNET_EVM_PATH";
              value = "${pkgs.avalanche-subnet-cli}/bin/subnet-cli";
            }
          ];
        packages = with pkgs;
          [
            alejandra # https://github.com/kamadorueda/alejandra
            delve # https://github.com/go-delve/delve
            #            go_1_18 # https://go.dev/
            pkgs_go119.go_1_19
            gotools # https://go.googlesource.com/tools
            go-ethereum # https://geth.ethereum.org/
            gofumpt # https://github.com/mvdan/gofumpt
            jq # https://stedolan.github.io/jq/
            just # https://github.com/casey/just
            nats-server # https://github.com/nats-io/nats-server
            nats-top # https://github.com/nats-io/nats-top
            natscli # https://nats.io/
            nodePackages.prettier # https://prettier.io/
            protobuf # https://github.com/protocolbuffers/protobuf
            protoc-gen-go # https://pkg.go.dev/google.golang.org/protobuf
            treefmt # https://github.com/numtide/treefmt
          ]
          ++ lists.optionals isLinux [
            # for now, these local packages doesn't work on darwin
            avalanche-cli # https://github.com/ava-labs/avalanche-cli
            avalanche-network-runner # https://github.com/ava-labs/avalanche-network-runner
            avalanche-subnet-cli # https://github.com/ava-labs/subnet-cli
            avalanchego # https://github.com/ava-labs/avalanchego

            # for Darwin docker should be installed separately
            docker
            docker-compose
          ];
        commands = with pkgs;
          [
            (dev go-ethereum)
            (dev nats-server)
            (dev nats-top)
            (dev natscli)
            (dev protobuf)

            (formatter alejandra)
            (formatter gofumpt)
            (formatter nodePackages.prettier)

            (linter golangci-lint)

            (util jq)
            (util just)
          ]
          ++ lists.optionals isLinux [
            # for now, we only add these utilities if on linux
            (dev avalanche-cli)
            (dev avalanche-network-runner)
            (dev avalanche-subnet-cli)
            (dev avalanchego)

            (dev docker)
            (dev docker-compose)
          ];
      };

      # nix run .#<app>
      apps = {
        tethys-proxy = mkApp {
          name = "tethys-proxy";
          drv = packages.tethys-proxy;
        };
        tethys-sidecar = mkApp {
          name = "tethys-sidecar";
          drv = packages.tethys-sidecar;
        };
      };

      # nix flake check
      # TODO: Once CI is configured, add proper hooks and checks
      checks = {
        pre-commit-check = pre-commit-hooks.lib.${system}.run {
          src = ./.;
          hooks = {
            alejandra.enable = true;
          };
        };
      };
    });
}
