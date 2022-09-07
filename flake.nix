{
  description = "Tethys, a smart load balancer for the blockchain";

  nixConfig = {
    substituters = [
      "https://cache.nixos.org"
      "https://nix-community.cachix.org"
      "https://pre-commit-hooks.cachix.org"
      "https://ethereum-nix.cachix.org"
    ];
    trusted-public-keys = [
      "cache.nixos.org-1:6NCHdD59X431o0gWypbMrAURkbJ16ZPMQFGspcDShjY="
      "nix-community.cachix.org-1:mB9FSh9qf2dCimDSUo8Zy7bkq5CX+/rkCWyvRCYg3Fs="
      "pre-commit-hooks.cachix.org-1:Pkk3Panw5AW24TOv6kz3PvLhlH8puAsJTBbOPmBo7Rc="
      "ethereum-nix.cachix.org-1:mpmQuO1myAs3CXDBLh8uQy4QDFtemaDKLD4UKmVjByE="
    ];
  };

  inputs = {
    # sources
    nixpkgs.url = "github:nixos/nixpkgs/nixpkgs-unstable";
    ethereum-nix.url = "github:41north/ethereum.nix";

    # utilities
    devshell = {
      url = "github:numtide/devshell";
      inputs.nixpkgs.follows = "nixpkgs";
      inputs.flake-utils.follows = "flake-utils";
    };
    flake-utils.url = "github:numtide/flake-utils";
    pre-commit-hooks = {
      url = "github:cachix/pre-commit-hooks.nix";
      inputs.flake-utils.follows = "flake-utils";
    };
  };

  outputs = {
    self,
    devshell,
    nixpkgs,
    ethereum-nix,
    flake-utils,
    pre-commit-hooks,
    ...
  } @ inputs: let
    inherit (flake-utils.lib) eachDefaultSystem flattenTree mkApp;
  in
    eachDefaultSystem
    (system: let
      pkgs = import nixpkgs {
        inherit system;
        overlays = [
          devshell.overlay
          ethereum-nix.overlays.default
          (import ./nix/overlays)
        ];
      };
      ethereumPkgs = ethereum-nix.packages.${system};

      inherit (pkgs) dockerTools buildGoModule;
      inherit (pkgs.stdenv) isLinux;
      inherit (pkgs.lib) lists fakeSha256 licenses platforms;
      inherit (pkgs.devshell) mkShell;
      inherit (import ./nix/lib {inherit pkgs;}) pkgWithCategory buildGoApp;

      linters = with pkgs; [
        alejandra # https://github.com/kamadorueda/alejandra
        gofumpt # https://github.com/mvdan/gofumpt
        nodePackages.prettier # https://prettier.io/
        treefmt # https://github.com/numtide/treefmt
      ];

      # devshell command categories
      dev = pkgWithCategory "dev";
      linter = pkgWithCategory "linters";
      formatter = pkgWithCategory "formatters";
      util = pkgWithCategory "utils";
    in {
      # nix build .#<app>
      packages = let
        vendorSha256 = "sha256-JEBEjjiDRwyNb9woG0QEqbpBXQf0TPeSLrt57trIxXQ=";
      in
        rec {
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
      devShells.default = mkShell {
        # TODO: Not recognized properly, research why
        # inherit (self.checks.${system}.pre-commit-check) shellHook;

        packages = with pkgs;
          [
            (lib.hiPrio ethereumPkgs.prysm) # https://github.com/prysmaticlabs/prysm
            delve # https://github.com/go-delve/delve
            go_1_19 # https://go.dev/
            go-ethereum # https://geth.ethereum.org/
            gotools # https://go.googlesource.com/tools
            hivemind # https://github.com/DarthSim/hivemind
            jq # https://stedolan.github.io/jq/
            just # https://github.com/casey/just
            nats-server # https://github.com/nats-io/nats-server
            nats-top # https://github.com/nats-io/nats-top
            natscli # https://nats.io/
            protobuf # https://github.com/protocolbuffers/protobuf
            protoc-gen-go # https://pkg.go.dev/google.golang.org/protobuf
            websocat # https://github.com/vi/websocat
          ]
          ++ linters
          ++ lists.optionals isLinux [
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
            (linter hadolint)

            (util jq)
            (util just)
          ]
          ++ lists.optionals isLinux [
            (dev docker)
            (dev docker-compose)
          ];
      };

      # nix run .#<app>
      apps = {
        tethys-proxy = mkApp {
          name = "tethys-proxy";
          drv = self.packages.tethys-proxy;
        };
        tethys-sidecar = mkApp {
          name = "tethys-sidecar";
          drv = self.packages.tethys-sidecar;
        };
      };

      # nix flake check
      checks = {
        format =
          pkgs.runCommandNoCC "treefmt" {
            nativeBuildInputs = linters;
          } ''
            # keep timestamps so that treefmt is able to detect mtime changes
            cp --no-preserve=mode --preserve=timestamps -r ${self} source
            cd source
            HOME=$TMPDIR treefmt --fail-on-change
            touch $out
          '';
      };
    });
}
