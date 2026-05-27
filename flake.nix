{
  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };
  outputs =
    {
      self,
      nixpkgs,
      flake-utils,
    }:
    flake-utils.lib.eachDefaultSystem (
      system:
      let
        pkgs = import nixpkgs {
          inherit system;
          config.allowUnfree = true;
        };
        buildPkgs = with pkgs; [
          pkg-config
          scdoc
        ];
        libPkgs = with pkgs; [
        ];
        devPkgs = with pkgs; [
          just
          go
          golangci-lint
        ];
      in
      {
        packages.default = pkgs.buildGoModule {
          pname = "odyssey";
          version = "1.0.0";
          src = ./.;
          vendorHash = "sha256-PrDlkEswRpF32GEKHjPmgCL6B8kwC+e5p2YV0y0sAZc=";
          nativeBuildInputs = buildPkgs;
          buildInputs = libPkgs;
          postInstall = ''
            mv $out/bin/odyssey-tui $out/bin/odyssey
            mkdir -p $out/share/man/man1
            scdoc < odyssey.1.scd | sed "s/1980-01-01/$(date '+%B %Y')/" > $out/share/man/man1/odyssey.1
          '';
        };
        devShell = pkgs.mkShell {
          nativeBuildInputs = buildPkgs ++ [ pkgs.go ];
          buildInputs = libPkgs ++ devPkgs;
        };
      }
    );
}
