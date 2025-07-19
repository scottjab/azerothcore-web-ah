# AzerothCore Web AH - Nix Flake

This Nix flake provides a complete build and deployment solution for the AzerothCore Web Auction House Viewer.

## Features

- **Go Module Build**: Automatically builds the Go application with proper dependency management
- **Systemd Service**: Complete systemd service configuration with security hardening
- **NixOS Module**: Full NixOS integration with declarative configuration
- **Development Shell**: Development environment with all necessary tools
- **Firewall Integration**: Optional firewall port opening
- **Security**: Hardened service configuration with proper user isolation

## Quick Start

### Building the Package

```bash
# Build the package
nix build

# Run the application directly
nix run
```

### Development Environment

```bash
# Enter development shell
nix develop

# Available commands in the shell:
go run main.go    # Run the application
go test ./...     # Run tests
go mod tidy       # Clean up dependencies
```

## NixOS Configuration

### Basic Configuration

Add this to your `configuration.nix`:

```nix
{ config, lib, pkgs, ... }:

{
  imports = [
    (import (builtins.getFlake "github:scottjab/azerothcore-web-ah")).nixosModule
  ];

  services.azerothcore-web-ah = {
    enable = true;
    port = 8080;
    database = {
      host = "localhost";
      port = 3306;
      user = "root";
      password = "your_password_here";
      name = "acore_characters";
    };
    openFirewall = true; # Optional: opens port in firewall
  };
}
```

### Advanced Configuration

```nix
{ config, lib, pkgs, ... }:

{
  imports = [
    (import (builtins.getFlake "github:scottjab/azerothcore-web-ah")).nixosModule
  ];

  services.azerothcore-web-ah = {
    enable = true;
    
    # Service configuration
    user = "azerothcore-web-ah";
    group = "azerothcore-web-ah";
    port = 8080;
    
    # Database configuration
    database = {
      host = "db.example.com";
      port = 3306;
      user = "auction_user";
      password = "secure_password";
      name = "acore_characters";
    };
    
    # Optional: Use environment file
    environmentFile = "/etc/azerothcore-web-ah/secrets.env";
    
    # Optional: Open firewall port
    openFirewall = true;
  };
}
```

### Using Environment File

Create `/etc/azerothcore-web-ah/secrets.env`:

```bash
# Database Configuration
DB_HOST=localhost
DB_PORT=3306
DB_USER=root
DB_PASSWORD=your_password_here
DB_NAME=acore_characters

# Server Configuration
PORT=8080
```

Then reference it in your configuration:

```nix
services.azerothcore-web-ah = {
  enable = true;
  environmentFile = "/etc/azerothcore-web-ah/secrets.env";
  # Other options will be overridden by environment file
};
```

## Configuration Options

### Service Options

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `enable` | bool | false | Enable the service |
| `package` | package | built package | The package to use |
| `user` | string | "azerothcore-web-ah" | Service user |
| `group` | string | "azerothcore-web-ah" | Service group |
| `port` | port | 8080 | Service port |

### Database Options

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `database.host` | string | "localhost" | Database host |
| `database.port` | port | 3306 | Database port |
| `database.user` | string | "root" | Database user |
| `database.password` | string | - | Database password |
| `database.name` | string | "acore_characters" | Database name |

### Security Options

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `environmentFile` | path | null | Path to environment file |
| `openFirewall` | bool | false | Open port in firewall |

## Security Features

The systemd service includes several security hardening features:

- **User Isolation**: Runs as dedicated system user
- **No New Privileges**: Prevents privilege escalation
- **Private Temporary Directory**: Isolated temp space
- **System Protection**: Read-only system directories
- **Home Protection**: Read-only home directories
- **Resource Limits**: File descriptor limits
- **Restricted Paths**: Limited write access

## Service Management

```bash
# Start the service
sudo systemctl start azerothcore-web-ah

# Enable auto-start
sudo systemctl enable azerothcore-web-ah

# Check status
sudo systemctl status azerothcore-web-ah

# View logs
sudo journalctl -u azerothcore-web-ah -f

# Restart service
sudo systemctl restart azerothcore-web-ah
```

## Troubleshooting

### Common Issues

1. **Database Connection Failed**
   - Check database credentials in configuration
   - Ensure database server is running
   - Verify network connectivity

2. **Permission Denied**
   - Check service user permissions
   - Verify database user has proper access

3. **Port Already in Use**
   - Change the port in configuration
   - Check for other services using the port

### Debug Mode

To run with debug output:

```bash
# In development shell
DB_DEBUG=1 go run main.go

# Or modify the service
sudo systemctl edit azerothcore-web-ah
```

Add:
```ini
[Service]
Environment=DB_DEBUG=1
```

## Development

### Adding Dependencies

1. Update `go.mod` with new dependencies
2. Run `go mod tidy` in development shell
3. Rebuild the flake

### Modifying the Service

The service configuration is in the `nixosModules` section of the flake. Modify the `systemd.services.azerothcore-web-ah` configuration as needed.

### Testing Changes

```bash
# Test configuration
nixos-rebuild build --flake .#your-hostname

# Apply configuration
sudo nixos-rebuild switch --flake .#your-hostname
```

## License

This flake is provided under the same license as the main project. 