{
  lib,
  buildGoModule,
  fetchFromGitHub,
}:
buildGoModule rec {
  pname = "avalanchego";
  version = "1.7.13";

  src = fetchFromGitHub {
    owner = "ava-labs";
    repo = pname;
    rev = "v${version}";
    sha256 = "sha256-Rjg+dHd6MxHy+ZQcPIx18VCzIvxqP3crvU9nF6DG8rs=";
  };

  vendorSha256 = "sha256-aaWCj3bucb81P3GjspOiFgUOoL2bygy2lmwmnG/W2a8=";

  doCheck = true;

  ldflags = [
    "-s"
    "-w"
    "-X github.com/ava-labs/avalanchego/version.GitCommit=${src.rev}"
  ];

  postInstall = ''
    # Build directory should have this structure:
    # See: https://github.com/ava-labs/avalanchego/blob/acd07505cd701dbd3832ca7aa301865fc0737839/config/config.go#L84
    #
    #  build
    #  ├── avalanchego (the binary from compiling the app directory)
    #  └── plugins
    #       └── evm
    mkdir -p $out/build/plugins

    # Store bin inside build/ folder
    mv $out/bin/main $out/build/${pname}

    # For now, we symlink, but the idea is to give support to different plugins
    ln -s $out/build/${pname} $out/bin/${pname}
  '';

  subPackages = ["main/main.go"];

  meta = with lib; {
    homepage = "https://www.avax.network/";
    description = "Go implementation of an Avalanche node";
    license = licenses.bsd3;
    maintainers = with maintainers; [aldoborrero];
  };
}
