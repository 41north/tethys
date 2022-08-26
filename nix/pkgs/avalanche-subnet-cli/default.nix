{
  lib,
  buildGoModule,
  fetchFromGitHub,
  libusb1,
}:
# TODO: Solve compilation issue related to Zondax HID. See the following:
#   - https://gitlab.com/thorchain/thornode/-/issues/1250
#   - https://github.com/karalabe/hid/issues/27
#
# Alternative approach would be to use this forked version:
#   - https://github.com/dolmen-go/hid/commit/ae6e6ef1b126c15a70fcededd7d39499b17864f5
#
# But that implies updating several tooling related to Avalanche:
#   $ go mod why -m github.com/zondax/hid
#   # github.com/zondax/hid
#   github.com/ava-labs/subnet-cli/internal/key
#   github.com/ava-labs/avalanche-ledger-go
#   github.com/zondax/ledger-go
#   github.com/zondax/hid
#
buildGoModule rec {
  pname = "subnet-cli";
  #   version = "0.0.2";
  version = "206ed3a24a3db06a223ef0e6e064c0852a5789ca";

  src = fetchFromGitHub {
    # owner = "ava-labs";
    owner = "aldoborrero";
    repo = pname;
    rev = "${version}";
    # sha256 = "sha256-fBmHy4KHHM9KPkffc/2JvuPZ9QpIjx7xAWhAjTWdwRg=";
    sha256 = "sha256-XBWjcD68OXEaobEuodD/FxDMxUzmsU84r4Tr4eB+7Q0=";
  };

  #   vendorSha256 = "sha256-3wQaUOBrfjsp8Bcih8wCqNzGcI3F9bbxU0O0CuQdCGc=";
  vendorSha256 = "sha256-ykgWHP/G0+MX6xguEhbN7BGxaZd2XSh0Srt7B3aYN3c=";

  # E2E tests fails
  doCheck = false;

  buildInputs = [libusb1];

  ldflags = ["-s" "-w"];

  meta = with lib; {
    homepage = "https://github.com/ava-labs/subnet-cli";
    description = "A command-line interface to manage Avalanche Subnets";
    license = licenses.bsd3;
    maintainers = with maintainers; [aldoborrero];
    platforms = platforms.linux;
  };
}
