{

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    utils.url = "github:numtide/flake-utils";
  };

  outputs =
    {
      self,
      nixpkgs,
      utils,
    }:
    utils.lib.eachSystem [ "x86_64-linux" ] (
      system:
      let
        pkgs = import nixpkgs {
          inherit system;
        };
      in
      {
        packages.default =
          (pkgs.buildGoModule.override {
            go = pkgs.go_1_27;
          })
            {
              pname = "bender";
              version = "0.2.1";

              src = ./.;

              # ./cmd/bender contains package main
              subPackages = [
                "cmd/bender"
              ];

              vendorHash = "sha256-h9FkFi3SgF448kF2LlDD86EvhpfupEAmTqG7XWB7lNg=";

              nativeBuildInputs = [
                pkgs.makeWrapper
                pkgs.copyDesktopItems
              ];

              buildInputs = [
                pkgs.gtk4
                pkgs.webkitgtk_6_0
                pkgs.glib
                pkgs.glib-networking
                pkgs.gnutls
                pkgs.libsoup_3
              ];

              desktopItems = [
                (pkgs.makeDesktopItem {
                  name = "bender";
                  desktopName = "Bender";
                  exec = "bender";
                  icon = "bender";
                  comment = "Bender";
                  categories = [ "Utility" ];
                })
              ];

              postInstall = ''
                # Linux desktop icons
                install -Dm644 ${./cmd/bender/winres/icon16.png} \
                  $out/share/icons/hicolor/16x16/apps/bender.png

                install -Dm644 ${./cmd/bender/winres/icon32.png} \
                  $out/share/icons/hicolor/32x32/apps/bender.png

                install -Dm644 ${./cmd/bender/winres/icon48.png} \
                  $out/share/icons/hicolor/48x48/apps/bender.png

                wrapProgram $out/bin/bender \
                  --prefix LD_LIBRARY_PATH : ${
                    pkgs.lib.makeLibraryPath [
                      pkgs.gtk4
                      pkgs.webkitgtk_6_0
                      pkgs.glib
                      pkgs.glib-networking
                      pkgs.gnutls
                      pkgs.libsoup_3
                    ]
                  } \
                  --set GIO_EXTRA_MODULES \
                    "${pkgs.glib-networking}/lib/gio/modules" \
                  --set SSL_CERT_FILE \
                    "${pkgs.cacert}/etc/ssl/certs/ca-bundle.crt"
              '';
            };
      }
    );
}
