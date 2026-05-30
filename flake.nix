{
  description = "Help CLI development environment";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, flake-utils }:
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = nixpkgs.legacyPackages.${system};

        wailsDeps = with pkgs; [
          pkg-config
          gobject-introspection
          gtk3
          webkitgtk_4_1
          libsoup_3
          gsettings-desktop-schemas
          glib
        ];

        wailsWrapped = pkgs.writeShellScriptBin "wails" ''
          exec ${pkgs.wails}/bin/wails "$@" -tags webkit2_41
        '';
      in
      {
        devShells.default = pkgs.mkShell {
          buildInputs = with pkgs; [
            go
            gopls
            gotools
            go-tools
            nodejs_22
            stdenv.cc.cc.lib
            wailsWrapped
          ] ++ pkgs.lib.optionals pkgs.stdenv.isLinux wailsDeps;

          shellHook = ''
            export PATH="$PWD/bin:$PATH"
            export LD_LIBRARY_PATH="${pkgs.stdenv.cc.cc.lib}/lib:$LD_LIBRARY_PATH"
            export XDG_DATA_DIRS="${pkgs.gsettings-desktop-schemas}/share/gsettings-schemas/${pkgs.gsettings-desktop-schemas.name}:${pkgs.gtk3}/share/gsettings-schemas/${pkgs.gtk3.name}''${XDG_DATA_DIRS:+:$XDG_DATA_DIRS}"
          '';
        };
      }
    );
}
