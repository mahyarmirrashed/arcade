{
  description = "Flake for github:mahyarmirrashed/arcade";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-25.11";
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
        pkgs = nixpkgs.legacyPackages.${system};
      in
      {
        packages.default = pkgs.buildGoModule {
          pname = "arcade";
          version = "0.1.0";
          src = ./.;
          subPackages = [ "cmd/arcade" ];
          vendorHash = "sha256-DO9P1W9DPc1dod0g5IGu2dfVlWZCWQblp95TyweXtCQ=";
        };

        devShells.default = pkgs.mkShell {
          packages = with pkgs; [
            go
          ];
        };
      }
    );
}
