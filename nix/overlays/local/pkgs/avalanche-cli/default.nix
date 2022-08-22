{
  lib,
  buildGoModule,
  fetchFromGitHub,
}:
buildGoModule rec {
  pname = "avalanche-cli";
  version = "0.1.3";

  src = fetchFromGitHub {
    owner = "ava-labs";
    repo = pname;
    rev = "v${version}";
    sha256 = "sha256-duqAq8TooAqKt24gyurwva8y7YPZ7/58DzjZRGgy598=";
  };

  vendorSha256 = "sha256-6bVpPDZ1Eilc8nDnPbvmdqzxACij67a7+FxUz8M+ctk=";

  doCheck = true;

  ldflags = ["-s" "-w"];

  meta = with lib; {
    homepage = "https://github.com/ava-labs/avalanche-cli";
    description = "Avalanche CLI is a command line tool that gives developers access to everything Avalanche";
    license = licenses.bsd3;
    maintainers = with maintainers; [aldoborrero];
  };
}
