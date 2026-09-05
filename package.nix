{pkgs ? import <nixpkgs> {}}:
pkgs.buildGoModule {
  pname = "hometrustd";
  version = "0.1.0";

  src = ./.;

  vendorHash = "sha256-ABt6Hv+uJEfDqJ7/kBiSLgWPwuTh5pViJl+2inyqsyg=";

  meta = {
    description = "HomeTrust Daemon";
    homepage = "https://github.com/thomas-btst/hometrustd";
    mainProgram = "hometrustd";
  };
}
