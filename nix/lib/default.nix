{pkgs}: let
  inherit (pkgs.stdenv) isDarwin isLinux hostPlatform buildPlatform;

  goArch =
    if isLinux
    then "amd64"
    else "arm64";

  buildGo119Module =
    if isLinux
    then pkgs.buildGo119Module
    else
      pkgs.buildGo119Module.override {
        stdenv = pkgs.pkgsCross.aarch64-multiplatform.stdenv;
        go =
          pkgs.go_1_19
          // {
            GOOS = "linux";
            GOARCH = goArch;
          };
      };

  buildLayeredImage =
    if isLinux
    then pkgs.dockerTools.buildLayeredImage
    else pkgs.pkgsCross.aarch64-multiplatform.dockerTools.buildLayeredImage;
in {
  inherit buildLayeredImage;

  # Creates a devshell category with a given pkg
  pkgWithCategory = category: package: {inherit package category;};

  # Builds a go application with common settings
  buildGoApp = {
    pname,
    version ? "dev",
    src,
    vendorSha256,
    package,
  }:
    buildGo119Module {
      inherit pname version src vendorSha256;
      ldflags = ["-s" "-w"];
      # glibc 2.34 instroduced a stricter check, causing the program
      # to bork inside the docker image with:
      #   > runtime/cgo: pthread_create failed: Operation not permitted
      # See: https://github.com/elastic/apm-server/issues/6238
      # For now, we can get away with this but a proper research needs to happen
      CGO_ENABLED = 0;
      subPackages = [package];
      meta = {
        homepage = "https://github.com/41north/tethys";
        description = "";
        license = pkgs.lib.licenses.unlicense;
      };
    };
}
