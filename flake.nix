{
  description = "Kyma - a terminal-based presentation tool";

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
        pkgs = nixpkgs.legacyPackages.${system};
        go = pkgs.go_1_26;
        buildGoModule = pkgs.buildGoModule.override { inherit go; };
        version = self.shortRev or self.dirtyShortRev or "dev";
      in
      {
        packages = {
          kyma = buildGoModule {
            pname = "kyma";
            inherit version;
            src = ./.;

            vendorHash = "sha256-tSasKfa3aB8n0VS9w79d4rHo+3qN9lPDw4sKpCgiwSo=";

            env.CGO_ENABLED = 0;

            ldflags = [
              "-s"
              "-w"
              "-X github.com/museslabs/kyma/cmd.version=${version}"
            ];

            meta = {
              description = "A terminal-based presentation tool";
              homepage = "https://github.com/museslabs/kyma";
              license = pkgs.lib.licenses.gpl3Only;
              mainProgram = "kyma";
            };
          };
          default = self.packages.${system}.kyma;
        };

        apps.default = flake-utils.lib.mkApp { drv = self.packages.${system}.kyma; };

        devShells.default = pkgs.mkShell {
          packages = [
            go
            pkgs.gopls
            pkgs.gotools
            pkgs.goreleaser
          ];
        };

        formatter = pkgs.nixfmt-rfc-style;
      }
    );
}
