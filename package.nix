{pkgs ? import <nixpkgs> {}}:
pkgs.buildGoModule {
  pname = "hometrustd";
  version = "0.1.0";

  src = ./.;

  vendorHash = "sha256-PTGksFXzBg5tdzpktqhdem/jNTHCKaxNhkvFrOi/0ag=";

  meta = {
    description = "HomeTrust Daemon";
    homepage = "https://github.com/thomas-btst/hometrustd";
    mainProgram = "hometrustd";
  };
}
