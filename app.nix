{
  self,
  lib,
  system,
}: {
  hometrustd = {
    type = "app";
    program = lib.getExe self.packages.${system}.hometrustd;
    meta.description = "HomeTrust Daemon executable";
  };

  default = self.apps.${system}.hometrustd;
}
