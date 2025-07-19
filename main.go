package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/joho/godotenv"
)

// AuctionItem represents an auction house item
type AuctionItem struct {
	ID          int    `json:"id"`
	HouseID     int    `json:"house_id"`
	ItemGUID    int    `json:"item_guid"`
	ItemOwner   int    `json:"item_owner"`
	BuyoutPrice int    `json:"buyout_price"`
	Time        int    `json:"time"`
	BuyGUID     int    `json:"buy_guid"`
	LastBid     int    `json:"last_bid"`
	StartBid    int    `json:"start_bid"`
	Deposit     int    `json:"deposit"`
	ItemEntry   int    `json:"item_entry"`
	ItemName    string `json:"item_name"`
	OwnerName   string `json:"owner_name"`
	Count       int    `json:"count"`
	Quality     int    `json:"quality"`
	ItemLevel   int    `json:"item_level"`
	TimeLeft    string `json:"time_left"`
}

// AuctionHouseStats represents auction house statistics
type AuctionHouseStats struct {
	TotalItems   int `json:"total_items"`
	TotalValue   int `json:"total_value"`
	ActiveBids   int `json:"active_bids"`
	UniqueOwners int `json:"unique_owners"`
	UniqueItems  int `json:"unique_items"`
}

var db *sql.DB

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using system environment variables")
	}

	// Database connection
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true&loc=Local",
		getEnv("DB_USER", "root"),
		getEnv("DB_PASSWORD", ""),
		getEnv("DB_HOST", "localhost"),
		getEnv("DB_PORT", "3306"),
		getEnv("DB_NAME", "acore_characters"),
	)

	var err error
	db, err = sql.Open("mysql", dsn)
	if err != nil {
		log.Fatal("Error opening database:", err)
	}
	defer db.Close()

	// Test database connection
	if err := db.Ping(); err != nil {
		log.Fatal("Error connecting to database:", err)
	}

	// Set connection pool settings
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(25)
	db.SetConnMaxLifetime(5 * time.Minute)

	log.Println("Connected to database successfully")

	// Create router using Go's built-in ServeMux
	mux := http.NewServeMux()

	// Register routes
	mux.HandleFunc("GET /", handleHome)
	mux.HandleFunc("GET /api/auctions", handleGetAuctions)
	mux.HandleFunc("GET /api/stats", handleGetStats)
	mux.HandleFunc("GET /api/search", handleSearch)
	mux.HandleFunc("GET /api/sellers", handleGetSellers)

	// Start server
	port := getEnv("PORT", "8080")
	log.Printf("Server starting on port %s", port)
	log.Fatal(http.ListenAndServe(":"+port, mux))
}

func handleHome(w http.ResponseWriter, r *http.Request) {
	tmpl := template.Must(template.New("index").Parse(htmlTemplate))
	tmpl.Execute(w, nil)
}

func handleGetAuctions(w http.ResponseWriter, r *http.Request) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	limit := 50
	offset := (page - 1) * limit

	query := `
		SELECT 
			ah.id, ah.houseid, ah.itemguid, ah.itemowner, ah.buyoutprice,
			ah.time, ah.buyguid, ah.lastbid, ah.startbid, ah.deposit,
			ii.itemEntry, ii.count,
			COALESCE(c.name, 'Unknown') as owner_name,
			COALESCE(it.name, 'Unknown Item') as item_name,
			COALESCE(it.Quality, 0) as quality,
			COALESCE(it.ItemLevel, 0) as item_level
		FROM auctionhouse ah
		LEFT JOIN item_instance ii ON ah.itemguid = ii.guid
		LEFT JOIN characters c ON ah.itemowner = c.guid
		LEFT JOIN acore_world.item_template it ON ii.itemEntry = it.entry
		WHERE ah.time > UNIX_TIMESTAMP()
		ORDER BY ah.time ASC
		LIMIT ? OFFSET ?
	`

	rows, err := db.Query(query, limit, offset)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var auctions []AuctionItem
	for rows.Next() {
		var auction AuctionItem
		var timeLeft int
		err := rows.Scan(
			&auction.ID, &auction.HouseID, &auction.ItemGUID, &auction.ItemOwner,
			&auction.BuyoutPrice, &auction.Time, &auction.BuyGUID, &auction.LastBid,
			&auction.StartBid, &auction.Deposit, &auction.ItemEntry, &auction.Count,
			&auction.OwnerName, &auction.ItemName, &auction.Quality, &auction.ItemLevel,
		)
		if err != nil {
			log.Printf("Error scanning auction: %v", err)
			continue
		}

		// Calculate time left
		timeLeft = auction.Time - int(time.Now().Unix())
		if timeLeft > 0 {
			auction.TimeLeft = formatTimeLeft(timeLeft)
		} else {
			auction.TimeLeft = "Expired"
		}

		auctions = append(auctions, auction)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"auctions": auctions,
		"page":     page,
		"limit":    limit,
	})
}

func handleGetStats(w http.ResponseWriter, r *http.Request) {
	query := `
		SELECT 
			COUNT(*) as total_items,
			SUM(ah.buyoutprice) as total_value,
			COUNT(DISTINCT ah.itemowner) as unique_owners,
			COUNT(DISTINCT ii.itemEntry) as unique_items
		FROM auctionhouse ah
		LEFT JOIN item_instance ii ON ah.itemguid = ii.guid
		WHERE ah.time > UNIX_TIMESTAMP()
	`

	var stats AuctionHouseStats
	err := db.QueryRow(query).Scan(
		&stats.TotalItems,
		&stats.TotalValue,
		&stats.UniqueOwners,
		&stats.UniqueItems,
	)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Count active bids (items with bids)
	bidQuery := `SELECT COUNT(*) FROM auctionhouse WHERE lastbid > 0 AND time > UNIX_TIMESTAMP()`
	err = db.QueryRow(bidQuery).Scan(&stats.ActiveBids)
	if err != nil {
		log.Printf("Error counting active bids: %v", err)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

func handleSearch(w http.ResponseWriter, r *http.Request) {
	searchTerm := r.URL.Query().Get("q")
	if searchTerm == "" {
		http.Error(w, "Search term required", http.StatusBadRequest)
		return
	}

	query := `
		SELECT 
			ah.id, ah.houseid, ah.itemguid, ah.itemowner, ah.buyoutprice,
			ah.time, ah.buyguid, ah.lastbid, ah.startbid, ah.deposit,
			ii.itemEntry, ii.count,
			COALESCE(c.name, 'Unknown') as owner_name,
			COALESCE(it.name, 'Unknown Item') as item_name,
			COALESCE(it.Quality, 0) as quality,
			COALESCE(it.ItemLevel, 0) as item_level
		FROM auctionhouse ah
		LEFT JOIN item_instance ii ON ah.itemguid = ii.guid
		LEFT JOIN characters c ON ah.itemowner = c.guid
		LEFT JOIN acore_world.item_template it ON ii.itemEntry = it.entry
		WHERE ah.time > UNIX_TIMESTAMP()
		AND (it.name LIKE ? OR c.name LIKE ?)
		ORDER BY ah.time ASC
		LIMIT 100
	`

	searchPattern := "%" + searchTerm + "%"
	rows, err := db.Query(query, searchPattern, searchPattern)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var auctions []AuctionItem
	for rows.Next() {
		var auction AuctionItem
		var timeLeft int
		err := rows.Scan(
			&auction.ID, &auction.HouseID, &auction.ItemGUID, &auction.ItemOwner,
			&auction.BuyoutPrice, &auction.Time, &auction.BuyGUID, &auction.LastBid,
			&auction.StartBid, &auction.Deposit, &auction.ItemEntry, &auction.Count,
			&auction.OwnerName, &auction.ItemName, &auction.Quality, &auction.ItemLevel,
		)
		if err != nil {
			log.Printf("Error scanning auction: %v", err)
			continue
		}

		// Calculate time left
		timeLeft = auction.Time - int(time.Now().Unix())
		if timeLeft > 0 {
			auction.TimeLeft = formatTimeLeft(timeLeft)
		} else {
			auction.TimeLeft = "Expired"
		}

		auctions = append(auctions, auction)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"auctions": auctions,
		"search":   searchTerm,
	})
}

func handleGetSellers(w http.ResponseWriter, r *http.Request) {
	query := `
		SELECT 
			c.name as seller_name,
			COUNT(ah.id) as total_auctions,
			SUM(ah.buyoutprice) as total_value,
			COUNT(DISTINCT ii.itemEntry) as unique_items
		FROM auctionhouse ah
		LEFT JOIN characters c ON ah.itemowner = c.guid
		LEFT JOIN item_instance ii ON ah.itemguid = ii.guid
		WHERE ah.time > UNIX_TIMESTAMP()
		AND c.name IS NOT NULL
		GROUP BY ah.itemowner, c.name
		ORDER BY total_auctions DESC, total_value DESC
	`

	rows, err := db.Query(query)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	type Seller struct {
		Name          string `json:"name"`
		TotalAuctions int    `json:"total_auctions"`
		TotalValue    int    `json:"total_value"`
		UniqueItems   int    `json:"unique_items"`
	}

	var sellers []Seller
	for rows.Next() {
		var seller Seller
		err := rows.Scan(
			&seller.Name,
			&seller.TotalAuctions,
			&seller.TotalValue,
			&seller.UniqueItems,
		)
		if err != nil {
			log.Printf("Error scanning seller: %v", err)
			continue
		}
		sellers = append(sellers, seller)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"sellers": sellers,
	})
}

func formatTimeLeft(seconds int) string {
	if seconds <= 0 {
		return "Expired"
	}

	days := seconds / 86400
	hours := (seconds % 86400) / 3600
	minutes := (seconds % 3600) / 60

	if days > 0 {
		return fmt.Sprintf("%dd %dh", days, hours)
	} else if hours > 0 {
		return fmt.Sprintf("%dh %dm", hours, minutes)
	} else {
		return fmt.Sprintf("%dm", minutes)
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// HTML template with embedded CSS and JavaScript
const htmlTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>WoW Auction House Viewer</title>
    <style>
        * {
            margin: 0;
            padding: 0;
            box-sizing: border-box;
        }

        body {
            font-family: 'Segoe UI', Tahoma, Geneva, Verdana, sans-serif;
            background: linear-gradient(135deg, #1e3c72 0%, #2a5298 100%);
            color: #333;
            min-height: 100vh;
        }

        .container {
            max-width: 1400px;
            margin: 0 auto;
            padding: 20px;
        }

        .header {
            text-align: center;
            margin-bottom: 30px;
            color: white;
        }

        .header h1 {
            font-size: 2.5rem;
            margin-bottom: 10px;
            text-shadow: 2px 2px 4px rgba(0,0,0,0.5);
        }

        .header p {
            font-size: 1.1rem;
            opacity: 0.9;
        }

        .stats-grid {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
            gap: 20px;
            margin-bottom: 30px;
        }

        .stat-card {
            background: rgba(255, 255, 255, 0.95);
            padding: 20px;
            border-radius: 10px;
            text-align: center;
            box-shadow: 0 4px 6px rgba(0,0,0,0.1);
            transition: transform 0.2s;
        }

        .stat-card:hover {
            transform: translateY(-2px);
        }

        .stat-number {
            font-size: 2rem;
            font-weight: bold;
            color: #2a5298;
            margin-bottom: 5px;
        }

        .stat-label {
            color: #666;
            font-size: 0.9rem;
        }

        .search-section {
            background: rgba(255, 255, 255, 0.95);
            padding: 20px;
            border-radius: 10px;
            margin-bottom: 20px;
            box-shadow: 0 4px 6px rgba(0,0,0,0.1);
        }

        .sellers-section {
            margin-bottom: 20px;
        }

        .search-form {
            display: flex;
            gap: 10px;
            align-items: center;
        }

        .search-input {
            flex: 1;
            padding: 12px;
            border: 2px solid #ddd;
            border-radius: 5px;
            font-size: 1rem;
        }

        .search-input:focus {
            outline: none;
            border-color: #2a5298;
        }

        .btn {
            padding: 12px 24px;
            background: #2a5298;
            color: white;
            border: none;
            border-radius: 5px;
            cursor: pointer;
            font-size: 1rem;
            transition: background 0.2s;
        }

        .btn:hover {
            background: #1e3c72;
        }

        .auctions-table {
            background: rgba(255, 255, 255, 0.95);
            border-radius: 10px;
            overflow: hidden;
            box-shadow: 0 4px 6px rgba(0,0,0,0.1);
        }

        .table-header {
            background: #2a5298;
            color: white;
            padding: 15px 20px;
            font-weight: bold;
        }

        .table-container {
            overflow-x: auto;
        }

        table {
            width: 100%;
            border-collapse: collapse;
        }

        th, td {
            padding: 12px 15px;
            text-align: left;
            border-bottom: 1px solid #ddd;
        }

        th {
            background: #f8f9fa;
            font-weight: 600;
            color: #333;
            cursor: pointer;
            user-select: none;
            position: relative;
        }

        th:hover {
            background: #e9ecef;
        }

        th.sortable::after {
            content: '↕';
            position: absolute;
            right: 8px;
            color: #999;
        }

        th.sort-asc::after {
            content: '↑';
            color: #2a5298;
        }

        th.sort-desc::after {
            content: '↓';
            color: #2a5298;
        }

        tr:hover {
            background: #f5f5f5;
        }

        .quality-0 { 
            color: #9d9d9d; 
            text-shadow: 1px 1px 2px rgba(0,0,0,0.3);
            font-weight: 500;
        }
        .quality-1 { color: #ffffff; }
        .quality-2 { color: #1eff00; }
        .quality-3 { color: #0070dd; }
        .quality-4 { color: #a335ee; }
        .quality-5 { color: #ff8000; }

        .item-link {
            text-decoration: none;
            color: inherit;
        }

        .item-link:hover {
            text-decoration: underline;
        }

        .price {
            font-weight: bold;
            color: #2a5298;
        }

        .time-left {
            font-size: 0.9rem;
            color: #666;
        }

        .loading {
            text-align: center;
            padding: 40px;
            color: #666;
        }

        .error {
            background: #ffebee;
            color: #c62828;
            padding: 15px;
            border-radius: 5px;
            margin: 10px 0;
        }

        .pagination {
            display: flex;
            justify-content: center;
            gap: 10px;
            margin-top: 20px;
        }

        .pagination button {
            padding: 8px 16px;
            border: 1px solid #ddd;
            background: white;
            cursor: pointer;
            border-radius: 3px;
        }

        .pagination button:hover {
            background: #f5f5f5;
        }

        .pagination button.active {
            background: #2a5298;
            color: white;
            border-color: #2a5298;
        }

        @media (max-width: 768px) {
            .container {
                padding: 10px;
            }
            
            .header h1 {
                font-size: 2rem;
            }
            
            .search-form {
                flex-direction: column;
            }
            
            .stats-grid {
                grid-template-columns: repeat(2, 1fr);
            }
        }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>⚔️ WoW Auction House Viewer</h1>
            <p>Real-time auction house data from your AzerothCore server</p>
        </div>

        <div class="stats-grid" id="statsGrid">
            <div class="stat-card">
                <div class="stat-number" id="totalItems">-</div>
                <div class="stat-label">Total Items</div>
            </div>
            <div class="stat-card">
                <div class="stat-number" id="totalValue">-</div>
                <div class="stat-label">Total Value (Gold)</div>
            </div>
            <div class="stat-card">
                <div class="stat-number" id="activeBids">-</div>
                <div class="stat-label">Active Bids</div>
            </div>
            <div class="stat-card">
                <div class="stat-number" id="uniqueOwners">-</div>
                <div class="stat-label">Unique Sellers</div>
            </div>
        </div>

        <div class="search-section">
            <form class="search-form" id="searchForm">
                <input type="text" class="search-input" id="searchInput" placeholder="Search by item name or seller...">
                <button type="submit" class="btn">Search</button>
                <button type="button" class="btn" onclick="loadAuctions()">Refresh</button>
                <button type="button" class="btn" onclick="toggleSellers()">Show Sellers</button>
            </form>
        </div>

        <div class="sellers-section" id="sellersSection" style="display: none;">
            <div class="auctions-table">
                <div class="table-header">
                    <h2>Active Sellers</h2>
                </div>
                <div class="table-container">
                    <table id="sellersTable">
                        <thead>
                            <tr>
                                <th class="sortable" data-sort="name">Seller Name</th>
                                <th class="sortable" data-sort="total_auctions">Total Auctions</th>
                                <th class="sortable" data-sort="total_value">Total Value</th>
                                <th class="sortable" data-sort="unique_items">Unique Items</th>
                            </tr>
                        </thead>
                        <tbody id="sellersBody">
                            <tr>
                                <td colspan="4" class="loading">Loading sellers...</td>
                            </tr>
                        </tbody>
                    </table>
                </div>
            </div>
        </div>

        <div class="auctions-table">
            <div class="table-header">
                <h2>Active Auctions</h2>
            </div>
            <div class="table-container">
                <table id="auctionsTable">
                    <thead>
                        <tr>
                            <th class="sortable" data-sort="item_name">Item</th>
                            <th class="sortable" data-sort="quality">Quality</th>
                            <th class="sortable" data-sort="item_level">Level</th>
                            <th class="sortable" data-sort="count">Count</th>
                            <th class="sortable" data-sort="owner_name">Seller</th>
                            <th class="sortable" data-sort="current_bid">Current Bid</th>
                            <th class="sortable" data-sort="buyout_price">Buyout</th>
                            <th class="sortable" data-sort="time_left">Time Left</th>
                        </tr>
                    </thead>
                    <tbody id="auctionsBody">
                        <tr>
                            <td colspan="8" class="loading">Loading auctions...</td>
                        </tr>
                    </tbody>
                </table>
            </div>
        </div>

        <div class="pagination" id="pagination"></div>
    </div>

    <script>
        let currentPage = 1;
        let currentSearch = '';
        let currentAuctions = [];
        let currentSellers = [];
        let sortColumn = '';
        let sortDirection = 'asc';
        let sellersSortColumn = '';
        let sellersSortDirection = 'asc';

        // Load initial data
        document.addEventListener('DOMContentLoaded', function() {
            loadStats();
            loadAuctions();
            
            // Auto-refresh every 30 seconds
            setInterval(() => {
                loadStats();
                loadAuctions();
            }, 30000);
        });

        // Search form handler
        document.getElementById('searchForm').addEventListener('submit', function(e) {
            e.preventDefault();
            currentSearch = document.getElementById('searchInput').value.trim();
            currentPage = 1;
            if (currentSearch) {
                searchAuctions();
            } else {
                loadAuctions();
            }
        });

        // Add click handlers for sortable columns
        document.addEventListener('DOMContentLoaded', function() {
            // Auction table sorting
            const auctionHeaders = document.querySelectorAll('#auctionsTable th.sortable');
            auctionHeaders.forEach(header => {
                header.addEventListener('click', function() {
                    const column = this.getAttribute('data-sort');
                    if (sortColumn === column) {
                        sortDirection = sortDirection === 'asc' ? 'desc' : 'asc';
                    } else {
                        sortColumn = column;
                        sortDirection = 'asc';
                    }
                    
                    // Update sort indicators
                    auctionHeaders.forEach(h => {
                        h.classList.remove('sort-asc', 'sort-desc');
                    });
                    this.classList.add(sortDirection === 'asc' ? 'sort-asc' : 'sort-desc');
                    
                    // Sort and display auctions
                    sortAuctions();
                });
            });

            // Sellers table sorting
            const sellersHeaders = document.querySelectorAll('#sellersTable th.sortable');
            sellersHeaders.forEach(header => {
                header.addEventListener('click', function() {
                    const column = this.getAttribute('data-sort');
                    if (sellersSortColumn === column) {
                        sellersSortDirection = sellersSortDirection === 'asc' ? 'desc' : 'asc';
                    } else {
                        sellersSortColumn = column;
                        sellersSortDirection = 'asc';
                    }
                    
                    // Update sort indicators
                    sellersHeaders.forEach(h => {
                        h.classList.remove('sort-asc', 'sort-desc');
                    });
                    this.classList.add(sellersSortDirection === 'asc' ? 'sort-asc' : 'sort-desc');
                    
                    // Sort and display sellers
                    sortSellers();
                });
            });
        });

        async function loadStats() {
            try {
                const response = await fetch('/api/stats');
                const stats = await response.json();
                
                document.getElementById('totalItems').textContent = stats.total_items.toLocaleString();
                document.getElementById('totalValue').textContent = formatGold(stats.total_value);
                document.getElementById('activeBids').textContent = stats.active_bids.toLocaleString();
                document.getElementById('uniqueOwners').textContent = stats.unique_owners.toLocaleString();
            } catch (error) {
                console.error('Error loading stats:', error);
            }
        }

        async function loadAuctions() {
            try {
                const response = await fetch('/api/auctions?page=' + currentPage);
                const data = await response.json();
                currentAuctions = data.auctions;
                sortAuctions();
                updatePagination(data.page, data.limit);
            } catch (error) {
                console.error('Error loading auctions:', error);
                document.getElementById('auctionsBody').innerHTML = 
                    '<tr><td colspan="8" class="error">Error loading auctions</td></tr>';
            }
        }

        async function searchAuctions() {
            try {
                const response = await fetch('/api/search?q=' + encodeURIComponent(currentSearch));
                const data = await response.json();
                currentAuctions = data.auctions;
                sortAuctions();
                document.getElementById('pagination').innerHTML = '';
            } catch (error) {
                console.error('Error searching auctions:', error);
                document.getElementById('auctionsBody').innerHTML = 
                    '<tr><td colspan="8" class="error">Error searching auctions</td></tr>';
            }
        }

        function sortAuctions() {
            if (!currentAuctions || currentAuctions.length === 0) {
                displayAuctions([]);
                return;
            }

            const sortedAuctions = [...currentAuctions].sort((a, b) => {
                let aVal, bVal;

                switch (sortColumn) {
                    case 'item_name':
                        aVal = a.item_name.toLowerCase();
                        bVal = b.item_name.toLowerCase();
                        break;
                    case 'quality':
                        aVal = a.quality;
                        bVal = b.quality;
                        break;
                    case 'item_level':
                        aVal = a.item_level;
                        bVal = b.item_level;
                        break;
                    case 'count':
                        aVal = a.count;
                        bVal = b.count;
                        break;
                    case 'owner_name':
                        aVal = a.owner_name.toLowerCase();
                        bVal = b.owner_name.toLowerCase();
                        break;
                    case 'current_bid':
                        aVal = a.last_bid || a.start_bid;
                        bVal = b.last_bid || b.start_bid;
                        break;
                    case 'buyout_price':
                        aVal = a.buyout_price;
                        bVal = b.buyout_price;
                        break;
                    case 'time_left':
                        // Convert time left to seconds for sorting
                        aVal = parseTimeLeftToSeconds(a.time_left);
                        bVal = parseTimeLeftToSeconds(b.time_left);
                        break;
                    default:
                        return 0;
                }

                if (aVal < bVal) return sortDirection === 'asc' ? -1 : 1;
                if (aVal > bVal) return sortDirection === 'asc' ? 1 : -1;
                return 0;
            });

            displayAuctions(sortedAuctions);
        }

        function parseTimeLeftToSeconds(timeLeft) {
            if (timeLeft === 'Expired') return -1;
            
            const parts = timeLeft.split(' ');
            let seconds = 0;
            
            for (let i = 0; i < parts.length; i += 2) {
                const value = parseInt(parts[i]);
                const unit = parts[i + 1];
                
                if (unit.includes('d')) seconds += value * 86400;
                else if (unit.includes('h')) seconds += value * 3600;
                else if (unit.includes('m')) seconds += value * 60;
            }
            
            return seconds;
        }

        function toggleSellers() {
            const sellersSection = document.getElementById('sellersSection');
            const button = event.target;
            
            if (sellersSection.style.display === 'none') {
                sellersSection.style.display = 'block';
                button.textContent = 'Hide Sellers';
                loadSellers();
            } else {
                sellersSection.style.display = 'none';
                button.textContent = 'Show Sellers';
            }
        }

        async function loadSellers() {
            try {
                const response = await fetch('/api/sellers');
                const data = await response.json();
                currentSellers = data.sellers;
                sortSellers();
            } catch (error) {
                console.error('Error loading sellers:', error);
                document.getElementById('sellersBody').innerHTML = 
                    '<tr><td colspan="4" class="error">Error loading sellers</td></tr>';
            }
        }

        function sortSellers() {
            if (!currentSellers || currentSellers.length === 0) {
                displaySellers([]);
                return;
            }

            const sortedSellers = [...currentSellers].sort((a, b) => {
                let aVal, bVal;

                switch (sellersSortColumn) {
                    case 'name':
                        aVal = a.name.toLowerCase();
                        bVal = b.name.toLowerCase();
                        break;
                    case 'total_auctions':
                        aVal = a.total_auctions;
                        bVal = b.total_auctions;
                        break;
                    case 'total_value':
                        aVal = a.total_value;
                        bVal = b.total_value;
                        break;
                    case 'unique_items':
                        aVal = a.unique_items;
                        bVal = b.unique_items;
                        break;
                    default:
                        return 0;
                }

                if (aVal < bVal) return sellersSortDirection === 'asc' ? -1 : 1;
                if (aVal > bVal) return sellersSortDirection === 'asc' ? 1 : -1;
                return 0;
            });

            displaySellers(sortedSellers);
        }

        function displaySellers(sellers) {
            const tbody = document.getElementById('sellersBody');
            
            if (sellers.length === 0) {
                tbody.innerHTML = '<tr><td colspan="4" class="loading">No sellers found</td></tr>';
                return;
            }

            tbody.innerHTML = sellers.map(function(seller) {
                return '<tr>' +
                    '<td>' + seller.name + '</td>' +
                    '<td>' + seller.total_auctions.toLocaleString() + '</td>' +
                    '<td class="price">' + formatGold(seller.total_value) + '</td>' +
                    '<td>' + seller.unique_items.toLocaleString() + '</td>' +
                    '</tr>';
            }).join('');
        }

        function displayAuctions(auctions) {
            const tbody = document.getElementById('auctionsBody');
            
            if (auctions.length === 0) {
                tbody.innerHTML = '<tr><td colspan="8" class="loading">No auctions found</td></tr>';
                return;
            }

            tbody.innerHTML = auctions.map(function(auction) {
                const wowheadUrl = 'https://www.wowhead.com/wotlk/item=' + auction.item_entry;
                return '<tr>' +
                    '<td><a href="' + wowheadUrl + '" target="_blank" class="item-link"><span class="quality-' + auction.quality + '">' + auction.item_name + '</span></a></td>' +
                    '<td><span class="quality-' + auction.quality + '">' + getQualityName(auction.quality) + '</span></td>' +
                    '<td>' + auction.item_level + '</td>' +
                    '<td>' + auction.count + '</td>' +
                    '<td>' + auction.owner_name + '</td>' +
                    '<td class="price">' + formatGold(auction.last_bid || auction.start_bid) + '</td>' +
                    '<td class="price">' + (auction.buyout_price > 0 ? formatGold(auction.buyout_price) : 'No Buyout') + '</td>' +
                    '<td class="time-left">' + auction.time_left + '</td>' +
                    '</tr>';
            }).join('');
        }

        function updatePagination(page, limit) {
            const pagination = document.getElementById('pagination');
            pagination.innerHTML = '';
            
            if (page > 1) {
                pagination.innerHTML += '<button onclick="changePage(' + (page - 1) + ')">Previous</button>';
            }
            
            pagination.innerHTML += '<button class="active">' + page + '</button>';
            pagination.innerHTML += '<button onclick="changePage(' + (page + 1) + ')">Next</button>';
        }

        function changePage(page) {
            currentPage = page;
            loadAuctions();
        }

        function formatGold(copper) {
            if (!copper) return '0c';
            
            const gold = Math.floor(copper / 10000);
            const silver = Math.floor((copper % 10000) / 100);
            const copperRemainder = copper % 100;
            
            let result = '';
            if (gold > 0) result += gold + 'g ';
            if (silver > 0) result += silver + 's ';
            if (copperRemainder > 0 || result === '') result += copperRemainder + 'c';
            
            return result.trim();
        }

        function getQualityName(quality) {
            const qualities = ['Poor', 'Common', 'Uncommon', 'Rare', 'Epic', 'Legendary'];
            return qualities[quality] || 'Unknown';
        }
    </script>
</body>
</html>`
