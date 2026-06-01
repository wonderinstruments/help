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

        runtimeLibs = with pkgs; [
          onnxruntime
          gtk3
          webkitgtk_4_1
          libsoup_3
          glib
          gsettings-desktop-schemas
          gst_all_1.gstreamer
          gst_all_1.gst-plugins-base
          gst_all_1.gst-plugins-good
        ];

        wrapHelp = name: bin: pkgs.stdenv.mkDerivation {
          pname = name;
          version = "0.1.0";
          src = ./bin;
          nativeBuildInputs = [ pkgs.makeWrapper ];
          dontBuild = true;
          installPhase = ''
            mkdir -p $out/bin
            cp ${bin} $out/bin/${name}
            chmod +x $out/bin/${name}
            wrapProgram $out/bin/${name} \
              --prefix LD_LIBRARY_PATH : "${pkgs.lib.makeLibraryPath runtimeLibs}" \
              --prefix XDG_DATA_DIRS : "${pkgs.gsettings-desktop-schemas}/share/gsettings-schemas/${pkgs.gsettings-desktop-schemas.name}:${pkgs.gtk3}/share/gsettings-schemas/${pkgs.gtk3.name}" \
              --prefix GST_PLUGIN_PATH : "${pkgs.gst_all_1.gst-plugins-base}/lib/gstreamer-1.0:${pkgs.gst_all_1.gst-plugins-good}/lib/gstreamer-1.0"
          '';
        };
      in
      {
        packages = {
          help-cli = wrapHelp "help-cli" "help-cli";
          help = wrapHelp "help" "help-ui";
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
