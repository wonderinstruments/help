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
          gst_all_1.gstreamer
          gst_all_1.gst-plugins-base
          gst_all_1.gst-plugins-good
        ];

        wailsWrapped = pkgs.writeShellScriptBin "wails" ''
          exec ${pkgs.wails}/bin/wails "$@" -tags webkit2_41
        '';

        help-cli = pkgs.buildGoModule {
          pname = "help-cli";
          version = "0.1.0";
          src = ./.;
          vendorHash = "sha256-GFAgl1rEh/SpMNaXXGNd3JRKzp7PGwlXeOnplTA84UI=";
          subPackages = [ "cmd/help" ];
          tags = [ "fts5" ];
          env.CGO_ENABLED = "1";
          nativeBuildInputs = [ pkgs.makeWrapper ];
          postInstall = ''
            mv $out/bin/help $out/bin/help-cli
            wrapProgram $out/bin/help-cli \
              --prefix LD_LIBRARY_PATH : "${pkgs.onnxruntime}/lib"
          '';
        };

        help-frontend = pkgs.buildNpmPackage {
          pname = "help-frontend";
          version = "0.1.0";
          src = ./ui/frontend;
          npmDepsHash = "sha256-Av70fcUrR7LOfmUjG88/FGHrEJDlhElYetnDcqtucnQ=";
          dontNpmBuild = true;
          buildPhase = ''
            node build.mjs
          '';
          installPhase = ''
            mkdir -p $out
            cp -r dist/* $out/
          '';
        };

        help-ui = pkgs.buildGoModule {
          pname = "help";
          version = "0.1.0";
          src = ./.;
          vendorHash = "sha256-GFAgl1rEh/SpMNaXXGNd3JRKzp7PGwlXeOnplTA84UI=";
          subPackages = [ "ui" ];
          tags = [ "fts5" "webkit2_41" ];
          env.CGO_ENABLED = "1";

          nativeBuildInputs = with pkgs; [ makeWrapper pkg-config ];
          buildInputs = with pkgs; [
            gtk3
            webkitgtk_4_1
            libsoup_3
            glib
          ];

          preBuild = ''
            mkdir -p ui/frontend/dist
            cp -r ${help-frontend}/* ui/frontend/dist/
          '';

          postInstall = ''
            mv $out/bin/ui $out/bin/help
            wrapProgram $out/bin/help \
              --prefix LD_LIBRARY_PATH : "${pkgs.onnxruntime}/lib"
          '';
        };
      in
      {
        packages = {
          help-cli = help-cli;
          help = help-ui;
        };

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
            export GST_PLUGIN_PATH="${pkgs.gst_all_1.gst-plugins-base}/lib/gstreamer-1.0:${pkgs.gst_all_1.gst-plugins-good}/lib/gstreamer-1.0''${GST_PLUGIN_PATH:+:$GST_PLUGIN_PATH}"
          '';
        };
      }
    );
}
