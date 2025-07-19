# WoW Auction House Web Viewer

A modern web application for viewing real-time auction house data from your AzerothCore World of Warcraft private server. Built with Go and featuring a beautiful, responsive web interface.

## Features

- 🏪 **Real-time Auction Data**: View live auction house listings from your server
- 📊 **Statistics Dashboard**: See total items, value, active bids, and unique sellers
- 🔍 **Search Functionality**: Search by item name or seller name
- 🎨 **Quality-based Coloring**: Items are colored according to their quality (Poor, Common, Uncommon, Rare, Epic, Legendary)
- 💰 **Gold Formatting**: Prices displayed in proper WoW gold format (g/s/c)
- ⏰ **Time Remaining**: Shows time left for each auction
- 📱 **Responsive Design**: Works on desktop and mobile devices
- 🔄 **Auto-refresh**: Data automatically updates every 30 seconds
- 🚀 **Lightweight**: Uses only Go standard library for HTTP routing (no external dependencies)

## Prerequisites

- Go 1.22 or later (uses Go's built-in http.ServeMux)
- MySQL database with AzerothCore data
- Access to the following databases:
  - `acore_characters` (for auction house data)
  - `acore_world` (for item templates)

## Installation

1. **Clone the repository**:
   ```bash
   git clone https://github.com/scottjab/azerothcore-web-ah.git
   cd azerothcore-web-ah
   ```

2. **Install dependencies**:
   ```bash
   go mod tidy
   ```

3. **Configure database connection**:
   ```bash
   cp env.example .env
   ```
   
   Edit `.env` with your database credentials:
   ```env
   DB_HOST=localhost
   DB_PORT=3306
   DB_USER=your_username
   DB_PASSWORD=your_password
   DB_NAME=acore_characters
   PORT=8080
   ```

4. **Build and run**:
   ```bash
   go run main.go
   ```

   Or build a binary:
   ```bash
   go build -o wow-ah-viewer main.go
   ./wow-ah-viewer
   ```

## Usage

1. Open your web browser and navigate to `http://localhost:8080`
2. The application will automatically load auction house data
3. Use the search bar to find specific items or sellers
4. Click "Refresh" to manually update the data
5. Navigate through pages using the pagination controls

## Database Schema

The application connects to your AzerothCore database and queries the following tables:

- `auctionhouse` - Contains auction listings
- `item_instance` - Item data for auctioned items
- `characters` - Character names for sellers
- `item_template` - Item template data (name, quality, level)

## API Endpoints

- `GET /` - Main web interface
- `GET /api/auctions?page=N` - Get paginated auction data
- `GET /api/stats` - Get auction house statistics
- `GET /api/search?q=term` - Search auctions by item name or seller

## Configuration

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `DB_HOST` | `localhost` | MySQL server hostname |
| `DB_PORT` | `3306` | MySQL server port |
| `DB_USER` | `root` | MySQL username |
| `DB_PASSWORD` | `` | MySQL password |
| `DB_NAME` | `acore_characters` | Database name |
| `PORT` | `8080` | Web server port |

### Database Permissions

Ensure your MySQL user has the following permissions:
- `SELECT` on `acore_characters.auctionhouse`
- `SELECT` on `acore_characters.item_instance`
- `SELECT` on `acore_characters.characters`
- `SELECT` on `acore_world.item_template`

## Building for Production

To create a standalone binary with embedded assets:

```bash
go build -ldflags="-s -w" -o wow-ah-viewer main.go
```

This creates a single executable file that includes all HTML, CSS, and JavaScript.

## Docker Support

Create a `Dockerfile`:

```dockerfile
FROM golang:1.24-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o wow-ah-viewer main.go

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/wow-ah-viewer .
EXPOSE 8080
CMD ["./wow-ah-viewer"]
```

Build and run:
```bash
docker build -t wow-ah-viewer .
docker run -p 8080:8080 --env-file .env wow-ah-viewer
```

## Troubleshooting

### Database Connection Issues

1. Verify your database credentials in `.env`
2. Ensure the MySQL server is running
3. Check that your user has proper permissions
4. Verify the database names are correct

### No Data Displayed

1. Check if there are active auctions in your database
2. Verify the `item_template` table exists in `acore_world`
3. Check server logs for SQL errors

### Performance Issues

1. Ensure proper indexes exist on auctionhouse table
2. Consider increasing database connection pool settings
3. Monitor server resources during peak usage

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Test thoroughly
5. Submit a pull request

## License

This project is licensed under the MIT License - see the LICENSE file for details.

## Support

For issues and questions:
1. Check the troubleshooting section
2. Review the database schema requirements
3. Open an issue on GitHub

---

**Note**: This application is designed for AzerothCore private servers. Make sure you have proper permissions to access the database and comply with your server's terms of service. 