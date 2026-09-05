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
    homeModules.default = {pkgs, ...}: {
      imports = [
        ./module.nix
      ];

      _module.args = {
        htdPkgs = self.packages.${pkgs.stdenv.hostPlatform.system};
      };
    };

    packages = eachSystem (
      system: let
        pkgs = nixpkgs.legacyPackages.${system};
      in rec {
        hometrustd = pkgs.callPackage ./package.nix {};
        default = hometrustd;
      }
    );

    apps = eachSystem (system:
      import ./app.nix {
        inherit (nixpkgs) lib;
        inherit self system;
      });

    devShells = eachSystem (
      system:
        import ./shell.nix {
          inherit self;
          pkgs = nixpkgs.legacyPackages.${system};
        }
    );
  };
}
