{
  lib,
  config,
  pkgs,
  htdPkgs,
  ...
}: let
  inherit (lib) mkEnableOption mkPackageOption mkOption mkIf types;
  cfg = config.programs.hometrustd;
  toYAML = pkgs.formats.yaml {};
in {
  options.programs.hometrustd = {
    enable = mkEnableOption "HomeTrust Daemon";

    package = mkPackageOption htdPkgs "HomeTrust Daemon" {
      default = "hometrustd";
    };

    settings = mkOption {
      type = types.nullOr (types.attrsOf types.anything);
      description = "HomeTrust Daemon settings";
      example = {
        trusted_networks = {
          bssids = [
            {"00:11:22:33:44:55" = "Home";}
          ];
        };
      };
      default = null;
    };
  };

  config = mkIf cfg.enable {
    home.packages = [cfg.package];

    xdg.configFile."hometrust/config.yml".source = mkIf (cfg.settings != null) (toYAML.generate "hometrust.yml" cfg.settings);
  };
}
