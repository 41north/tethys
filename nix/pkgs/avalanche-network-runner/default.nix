{
  lib,
  buildGoModule,
  fetchFromGitHub,
}:
buildGoModule rec {
  pname = "avalanche-network-runner";
  version = "1.1.0";

  src = fetchFromGitHub {
    owner = "ava-labs";
    repo = pname;
    rev = "v${version}";
    sha256 = "sha256-ZSjEn4qQk9kl+pcMwVxj23SPU5ncpxBqrkCYC15bl2I=";
  };

  vendorSha256 = "sha256-nEGd40WIPakt0pWGppXVCffl9k1r80xA0Yr64KzxTjM=";

  doCheck = true;

  ldflags = ["-s" "-w"];

  subPackages = ["cmd/avalanche-network-runner"];

  meta = with lib; {
    homepage = "https://github.com/ava-labs/avalanche-network-runner";
    description = "Tool to run and interact with an Avalanche network locally";
    license = licenses.bsd3;
    maintainers = with maintainers; [aldoborrero];
  };
}
