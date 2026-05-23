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
          pname = "odyssey-tui";
          version = "1.0.0";
          src = ./.;
          vendorHash = "";
          nativeBuildInputs = buildPkgs;
          buildInputs = libPkgs;
          postInstall = ''
            mkdir -p $out/share/man/man1
            scdoc < odyssey-tui.1.scd | sed "s/1980-01-01/$(date '+%B %Y')/" > odyssey-tui.1
          '';
        };
        devShell = pkgs.mkShell {
          nativeBuildInputs = buildPkgs ++ [ pkgs.go ];
          buildInputs = libPkgs ++ devPkgs;
        };
      }
    );
}
