self: super:
with self.lib; let
  inherit (self.pkgs) callPackage;
in {
  # https://github.com/mvdan/gofumpt
  # gofumpt = super.gofumpt.overrideAttrs (attrs: rec {
  #   version = "master-70d743";
  #   src = super.fetchFromGitHub {
  #     owner = "mvdan";
  #     repo = attrs.pname;
  #     rev = "70d7433507d8d92bfa78a923e1f48de9b9e17203";
  #     sha256 = "sha256-X+IdHmOpY2h9AooRh/Ly1DIn3oGhm6yxxekEBknLR3o=";
  #   };
  #   doCheck = false;
  #   vendorSha256 = fakeSha256;
  #   ldflags = [
  #     "-s"
  #     "-w"
  #     "-X mvdan.cc/gofumpt/internal/version.version=${version}"
  #   ];
  # });
}
