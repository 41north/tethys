{pkgs}:
with (pkgs.lib); let
  inherit (pkgs) buildGo119Module;
in {
  # Creates a devshell category with a given pkg
  pkgWithCategory = category: package: {inherit package category;};

  # Builds a go application with common settings
  buildGoApp = {
    name,
    src,
    vendorSha256,
    package,
  }:
    buildGo119Module {
      inherit name src vendorSha256;
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
        license = licenses.unlicense;
      };
    };
}
