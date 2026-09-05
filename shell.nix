{
  self,
  pkgs,
}: {
  default = pkgs.mkShell {
    buildInputs = with pkgs; [
      self.packages.${pkgs.stdenv.hostPlatform.system}.hometrustd

      # Go tools
      go
      gopls
      golangci-lint

      # Nix tools
      alejandra
      statix
      deadnix
    ];
  };
}
