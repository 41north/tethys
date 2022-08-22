self: super: let
  callPackage = self.pkgs.callPackage;
  lib = self.lib;
in {
  # https://github.com/mvdan/gofumpt
  # Notes:
  #   - It seems latest commit 8dda8068d9f339047fc1777b688afb66a0a0db17,
  #     hasn't executed properly go mod tidy. Using previous one instead.
  #   - This thread contains list of tooling supporting Go Generics: https://github.com/golang/go/issues/50558/
  #   - It seems, that support in gofumpt is not completely done yet.
  gofumpt = super.gofumpt.overrideAttrs (attrs: rec {
    version = "master-900c61";
    src = super.fetchFromGitHub {
      owner = "mvdan";
      repo = attrs.pname;
      rev = "900c61a4cb83bedde751dd2aedf2fc1c73de5e40";
      sha256 = "sha256-TZMRsSfyL7G7SuLeUpfnAufzYp6XTj4MFzURkk9t9pM=";
    };
    vendorSha256 = lib.fakeSha256;
    ldflags = [
      "-s"
      "-w"
      "-X mvdan.cc/gofumpt/internal/version.version=${version}"
    ];
  });

  # https://github.com/ava-labs/avalanchego/
  avalanchego = callPackage ./local/pkgs/avalanchego {};

  # https://github.com/ava-labs/avalanche-cli
  avalanche-cli = callPackage ./local/pkgs/avalanche-cli {};

  # https://github.com/ava-labs/avalanche-network-runner
  avalanche-network-runner = callPackage ./local/pkgs/avalanche-network-runner {};

  # https://github.com/ava-labs/subnet-cli
  avalanche-subnet-cli = callPackage ./local/pkgs/avalanche-subnet-cli {};
}
