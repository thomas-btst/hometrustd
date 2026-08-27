{
  description = "HomeTrust Daemon";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
  };

  outputs = {
    self,
    nixpkgs,
  }: let
    systems = ["x86_64-linux" "aarch64-linux"];
    eachSystem = nixpkgs.lib.genAttrs systems;
  in {
    packages = eachSystem (
      system: let
        pkgs = nixpkgs.legacyPackages.${system};
      in rec {
        hometrustd = pkgs.callPackage ./package.nix {};
        default = hometrustd;
      }
    );

    apps = eachSystem (
      system: rec {
        hometrustd = {
          type = "app";
          program = nixpkgs.lib.getExe self.packages.${system}.hometrustd;
          meta.description = "HomeTrust Daemon executable";
        };
        default = hometrustd;
      }
    );

    devShells = eachSystem (
      system: let
        pkgs = nixpkgs.legacyPackages.${system};
      in {
        default = pkgs.mkShell {
          buildInputs = with pkgs; [
            go
            gopls
            golangci-lint
          ];
        };
      }
    );
  };
}
