# Example NixOS configuration for AzerothCore Web AH
# Add this to your configuration.nix or use as a separate module

{ config, lib, pkgs, ... }:

{
  # Import the flake module
  imports = [
    (import (builtins.getFlake "github:scottjab/azerothcore-web-ah")).nixosModule
  ];

  # Enable the service
  services.azerothcore-web-ah = {
    enable = true;
    
    # Service configuration
    port = 8080;
    user = "azerothcore-web-ah";
    group = "azerothcore-web-ah";
    
    # Database configuration
    database = {
      host = "localhost";
      port = 3306;
      user = "root";
      password = "your_secure_password_here";
      name = "acore_characters";
    };
    
    # Optional: Open firewall port
    openFirewall = true;
    
    # Optional: Use environment file instead of inline configuration
    # environmentFile = "/etc/azerothcore-web-ah/secrets.env";
  };

  # Optional: Additional system configuration
  # networking.firewall = {
  #   allowedTCPPorts = [ 8080 ];
  # };

  # Optional: Create environment file for secrets
  # environment.etc."azerothcore-web-ah/secrets.env" = {
  #   text = ''
  #     DB_HOST=localhost
  #     DB_PORT=3306
  #     DB_USER=root
  #     DB_PASSWORD=your_secure_password_here
  #     DB_NAME=acore_characters
  #     PORT=8080
  #   '';
  #   mode = "0600";
  # };
} 