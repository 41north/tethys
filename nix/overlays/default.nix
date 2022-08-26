self: super:
with self.lib; let
  inherit (self.pkgs) callPackage;
in {
  # https://github.com/mvdan/gofumpt
  # Notes:
  #   - It seems latest commit 8dda8068d9f339047fc1777b688afb66a0a0db17,
  #     hasn't executed properly go mod tidy. Using previous one instead.
  #   - This thread contains list of tooling supporting Go Generics: https://github.com/golang/go/issues/50558/
  #   - It seems, that support in gofumpt is not completely done yet.
  gofumpt = super.gofumpt.overrideAttrs (attrs: rec {
    version = "master-70d743";
    src = super.fetchFromGitHub {
      owner = "mvdan";
      repo = attrs.pname;
      rev = "70d7433507d8d92bfa78a923e1f48de9b9e17203";
      sha256 = "sha256-TZMRsSfyL7G7SuLeUpfnAufzYp6XTj4MFzURkk9t9pM=";
    };
    doCheck = false;
    vendorSha256 = fakeSha256;
    ldflags = [
      "-s"
      "-w"
      "-X mvdan.cc/gofumpt/internal/version.version=${version}"
    ];
  });
}
