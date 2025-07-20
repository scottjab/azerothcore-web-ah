{
  description = "AzerothCore Web Auction House Viewer";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, flake-utils }:
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = nixpkgs.legacyPackages.${system};
      in
      {
        packages = {
          azerothcore-web-ah = pkgs.buildGoModule {
            pname = "azerothcore-web-ah";
            version = "1.0.0";
            src = ./.;
            vendorHash = "sha256-kA5ITxwaDC3wTlfKpJYXHq5L3mnv+sYAOihBqQBVAXI=";
            doCheck = false;
            meta = with pkgs.lib; {
              description = "Web-based auction house viewer for AzerothCore servers";
              homepage = "https://github.com/scottjab/azerothcore-web-ah";
              license = licenses.mit;
              platforms = platforms.unix;
            };
          };
          default = self.packages.${system}.azerothcore-web-ah;
        };

        apps = {
          azerothcore-web-ah = flake-utils.lib.mkApp {
            drv = self.packages.${system}.azerothcore-web-ah;
          };
          default = self.apps.${system}.azerothcore-web-ah;
        };

        nixosModules = {
          azerothcore-web-ah = { config, lib, pkgs, ... }:
            with lib;
            let
              cfg = config.services.azerothcore-web-ah;
              settingsFormat = pkgs.formats.json { };
            in
            {
              options.services.azerothcore-web-ah = {
                enable = mkEnableOption "AzerothCore Web Auction House service";

                package = mkOption {
                  type = types.package;
                  default = self.packages.${system}.azerothcore-web-ah;
                  description = "The azerothcore-web-ah package to use.";
                };

                user = mkOption {
                  type = types.str;
                  default = "azerothcore-web-ah";
                  description = "User account under which the service runs.";
                };

                group = mkOption {
                  type = types.str;
                  default = "azerothcore-web-ah";
                  description = "Group under which the service runs.";
                };

                port = mkOption {
                  type = types.port;
                  default = 8080;
                  description = "Port on which the service listens.";
                };

                database = {
                  host = mkOption {
                    type = types.str;
                    default = "localhost";
                    description = "Database host address.";
                  };

                  port = mkOption {
                    type = types.port;
                    default = 3306;
                    description = "Database port.";
                  };

                  user = mkOption {
                    type = types.str;
                    default = "root";
                    description = "Database username.";
                  };

                  password = mkOption {
                    type = types.str;
                    default = "";
                    description = "Database password.";
                  };

                  name = mkOption {
                    type = types.str;
                    default = "acore_characters";
                    description = "Database name.";
                  };
                };

                environmentFile = mkOption {
                  type = types.nullOr types.path;
                  default = null;
                  description = "Path to environment file with additional configuration.";
                };

                openFirewall = mkOption {
                  type = types.bool;
                  default = false;
                  description = "Whether to open the service port in the firewall.";
                };
              };

              config = mkIf cfg.enable {
                users.users = mkIf (cfg.user == "azerothcore-web-ah") {
                  azerothcore-web-ah = {
                    isSystemUser = true;
                    group = cfg.group;
                    description = "AzerothCore Web Auction House service user";
                    home = "/var/lib/azerothcore-web-ah";
                    createHome = true;
                  };
                };

                users.groups = mkIf (cfg.group == "azerothcore-web-ah") {
                  azerothcore-web-ah = { };
                };

                systemd.services.azerothcore-web-ah = {
                  description = "AzerothCore Web Auction House Viewer";
                  wantedBy = [ "multi-user.target" ];
                  after = [ "network.target" ];
                  serviceConfig = {
                    Type = "simple";
                    User = cfg.user;
                    Group = cfg.group;
                    ExecStart = "${cfg.package}/bin/azerothcore-web-ah";
                    Restart = "always";
                    RestartSec = "10";
                    WorkingDirectory = "/var/lib/azerothcore-web-ah";
                    Environment = [
                      "PORT=${toString cfg.port}"
                      "DB_HOST=${cfg.database.host}"
                      "DB_PORT=${toString cfg.database.port}"
                      "DB_USER=${cfg.database.user}"
                      "DB_NAME=${cfg.database.name}"
                    ] ++ (lib.optional (cfg.database.password != "") "DB_PASSWORD=${cfg.database.password}")
                      ++ (lib.optional (cfg.environmentFile != null) "ENV_FILE=${cfg.environmentFile}");
                    EnvironmentFile = lib.optional (cfg.environmentFile != null) cfg.environmentFile;
                    # Security settings
                    NoNewPrivileges = true;
                    PrivateTmp = true;
                    ProtectSystem = "strict";
                    ProtectHome = true;
                    ReadWritePaths = [ "/var/lib/azerothcore-web-ah" ];
                    # Resource limits
                    LimitNOFILE = 65536;
                  };
                };

                networking.firewall = mkIf cfg.openFirewall {
                  allowedTCPPorts = [ cfg.port ];
                };

                # Create a default configuration file
                environment.etc."azerothcore-web-ah/config.json" = mkIf (cfg.environmentFile == null) {
                  text = builtins.toJSON {
                    database = {
                      host = cfg.database.host;
                      port = cfg.database.port;
                      user = cfg.database.user;
                      password = cfg.database.password;
                      name = cfg.database.name;
                    };
                    server = {
                      port = cfg.port;
                    };
                  };
                };
              };
            };
        };

        # Development shell
        devShells.default = pkgs.mkShell {
          buildInputs = with pkgs; [
            go
            gcc
            mysql80
            # Optional: Add other development tools
            # air # for hot reloading
            # delve # for debugging
          ];
          shellHook = ''
            echo "AzerothCore Web AH Development Environment"
            echo "Available commands:"
            echo "  go run main.go - Run the application"
            echo "  go test ./... - Run tests"
            echo "  go mod tidy - Clean up dependencies"
          '';
        };
      }
    ) // {
      # NixOS module
      nixosModule = self.nixosModules.${builtins.currentSystem}.azerothcore-web-ah;
    };
} 