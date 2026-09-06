{
  lib,
  config,
  pkgs,
  htdPkgs,
  ...
}: let
  inherit (lib) mkEnableOption mkPackageOption mkOption mkIf types getExe;
  cfg = config.services.hometrustd;
  toYAML = pkgs.formats.yaml {};
in {
  options.services.hometrustd = {
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
    systemd.user.services.hometrustd = {
      Unit = {
        Description = "HomeTrust Daemon";
        Documentation = "https://github.com/thomas-btst/hometrustd";
        After = ["network.target"];
      };

      Service = {
        Type = "simple";
        ExecStart = getExe cfg.package;
        Restart = "on-failure";
        RestartSec = "5s";
      };

      Install = {
        WantedBy = ["default.target"];
      };
    };

    home.packages = [cfg.package];

    xdg.configFile."hometrust/config.yml".source = mkIf (cfg.settings != null) (toYAML.generate "hometrust.yml" cfg.settings);
  };
}
