# MSSQL to PostgreSQL Sync Service

A robust, real-time synchronization service that replicates data from Microsoft SQL Server to PostgreSQL using Proto.Actor for concurrent processing, Gin for REST API, Zap for logging, and SQLx for database operations.

## 🚀 Features

- **One-Way Sync**: Seamless data synchronization from MSSQL (source) to PostgreSQL (target)
- **YAML Configuration**: Easy-to-manage master configuration file
- **Proto.Actor Integration**: Concurrent table synchronization with actor-based scheduling
- **Web API Triggers**: Manual sync operations via RESTful API
- **Automatic Table Creation**: Creates target tables if they don't exist
- **Field Filtering**: Sync only specific fields from source tables
- **Query Filtering**: Apply WHERE clauses to source queries
- **Configurable Refresh Rates**: Set different sync intervals per table
- **React.js Dashboard**: Beautiful, modern web interface with sync controls
- **Structured Logging**: Comprehensive logging with Zap

## 📋 Prerequisites

- Go 1.21 or higher
- Node.js 16+ and npm (for frontend)
- Microsoft SQL Server (source database)
- PostgreSQL (target database)
- Access credentials for both databases

## 🛠️ Installation

### 1. Clone the Repository

```bash
git clone <repository-url>
cd mssql-postgres-sync
```

### 2. Install Go Dependencies

```bash
go mod download
```

### 3. Install Frontend Dependencies

```bash
cd frontend
npm install
cd ..
```

## ⚙️ Configuration

### 1. Configure sync-config.yaml

Edit `config/sync-config.yaml` to set up your database connections and sync tables:

```yaml
# Source Database (MSSQL)
source:
  type: mssql
  host: localhost
  port: 1433
  database: SourceDB
  username: sa
  password: YourPassword123!

# Target Database (PostgreSQL)
target:
  type: postgresql
  host: localhost
  port: 5432
  database: targetdb
  username: postgres
  password: postgres
  sslmode: disable

# Default Settings
defaults:
  refresh_rate: 360  # seconds
  proto_actor_trigger: true
  webapi_trigger: true
  create_target_table: true

# Table Configurations
tables:
  - source_table: dbo.Users
    target_table: public.users
    sync_action: full
    refresh_rate: 360
    proto_actor_trigger: true
    webapi_trigger: true
    # fields: []  # Optional: specific fields only
    # filter: ""  # Optional: WHERE clause

# Projection UI Configuration
projections:
  - id: users-overview
    title: "Users Overview"
    description: "Summary of users with subscription and activity status"
    target_view: public.users
    sync_table: public.users
    header_color: "#1f2937"
    header_text_color: "#f9fafb"
    default_sort:
      column: LastModified
      direction: desc
    group_by:
      - Status
    fields:
      - column: UserID
        label: User ID
        type: number
        sortable: true
      - column: ProductName
        label: Primary Subscription
        type: text
        sortable: true
      - column: LastModified
        label: Last Seen
        type: datetime
        sortable: true
      - column: Status
        label: Status
        type: badge
        sortable: false
    filters:
      - id: status
        column: Status
        label: Status
        type: select
        options:
          - label: Active
            value: Active
          - label: Inactive
            value: Inactive
          - label: Suspended
            value: Suspended
    totals:
      - column: UserID
        label: Total Users
        format: count

```

### Configuration Options

#### Table Configuration Attributes:

- **source_table**: Source table name (with schema, e.g., `dbo.Users`)
- **target_table**: Target table name (with schema, e.g., `public.users`)
- **sync_action**: Sync type (`full`, `incremental`, `custom`)
- **refresh_rate**: Sync interval in seconds (default: 360)
- **proto_actor_trigger**: Enable automatic scheduled sync (default: true)
- **webapi_trigger**: Enable manual API trigger (default: true)
- **fields**: Array of specific fields to sync (empty = all fields)
- **filter**: SQL WHERE clause for source query (e.g., `IsActive = 1`)

## 🚀 Running the Service

### Option 1: Run Backend and Frontend Separately (Development)

**Terminal 1 - Backend:**
```bash
go run cmd/syncservice/main.go -config config/sync-config.yaml
```

**Terminal 2 - Frontend:**
```bash
cd frontend
npm start
```

Access the dashboard at: `http://localhost:3000`

### Option 2: Build and Run (Production)

**Build Frontend:**
```bash
cd frontend
npm run build
cd ..
```

**Build and Run Backend:**
```bash
go build -o syncservice cmd/syncservice/main.go
./syncservice -config config/sync-config.yaml
```

Access the dashboard at: `http://localhost:8080`

## 🌐 API Endpoints

### GET /api/health
Health check endpoint

**Response:**
```json
{
  "status": "healthy",
  "service": "mssql-postgres-sync",
  "time": "2024-01-01T12:00:00Z"
}
```

### GET /api/status
Get sync status for all tables

**Response:**
```json
{
  "status": "running",
  "tables": [
    {
      "source_table": "dbo.Users",
      "target_table": "public.users",
      "refresh_rate": 360,
      "proto_actor_enabled": true,
      "web_api_enabled": true
    }
  ]
}
```

### POST /api/sync
Trigger manual sync operation

**Request Body (sync specific table):**
```json
{
  "table_name": "public.users"
}
```

**Request Body (sync all tables):**
```json
{
  "sync_all": true
}
```

**Response:**
```json
{
  "success": true,
  "message": "Sync triggered for table: public.users"
}
```

## 📊 Architecture

```
┌─────────────────┐
│   React.js UI   │
│   (Frontend)    │
└────────┬────────┘
         │ HTTP/REST
         ▼
┌─────────────────┐
│   Gin Server    │
│   (API Layer)   │
└────────┬────────┘
         │
         ▼
┌─────────────────┐       ┌──────────────┐
│  Proto.Actor    │◄─────►│ Sync Engine  │
│  (Coordinator)  │       │              │
└────────┬────────┘       └──────┬───────┘
         │                       │
         ▼                       ▼
┌─────────────────┐       ┌──────────────┐
│   Sync Actors   │       │   SQLx DB    │
│  (Per Table)    │       │   Manager    │
└─────────────────┘       └──────┬───────┘
                                 │
                    ┌────────────┴────────────┐
                    ▼                         ▼
              ┌──────────┐            ┌──────────────┐
              │  MSSQL   │            │ PostgreSQL   │
              │ (Source) │            │  (Target)    │
              └──────────┘            └──────────────┘
```

## 🔧 How It Works

1. **Configuration Loading**: Service reads YAML config on startup
2. **Database Connections**: Establishes connections to MSSQL and PostgreSQL
3. **Actor System**: Creates Proto.Actor coordinator and per-table sync actors
4. **Scheduled Sync**: Each actor runs on its configured refresh interval
5. **Manual Triggers**: REST API allows on-demand sync operations
6. **Table Creation**: Automatically creates target tables with proper schema mapping
7. **Data Transfer**: Fetches from source, transforms, and loads to target
8. **Logging**: Comprehensive logging of all operations

## 🎨 Frontend Features

- **Real-time Status**: View all configured tables and their sync status
- **Manual Triggers**: Click to sync individual tables or all tables at once
- **Visual Feedback**: Loading states, success/error indicators
- **Responsive Design**: Works on desktop and mobile devices
- **Modern UI**: Beautiful gradient design with smooth animations

## 📝 Data Type Mapping

The service automatically maps MSSQL data types to PostgreSQL:

| MSSQL Type | PostgreSQL Type |
|------------|----------------|
| INT | INTEGER |
| BIGINT | BIGINT |
| BIT | BOOLEAN |
| DECIMAL/NUMERIC | NUMERIC |
| FLOAT | DOUBLE PRECISION |
| DATETIME | TIMESTAMP |
| VARCHAR | VARCHAR |
| NVARCHAR | VARCHAR |
| TEXT | TEXT |
| UNIQUEIDENTIFIER | UUID |
| VARBINARY | BYTEA |

## 🔒 Security Considerations

- Store sensitive credentials in environment variables
- Use SSL/TLS for database connections in production
- Implement authentication for the API endpoints
- Run with least-privilege database accounts
- Use connection pooling limits

## 🐛 Troubleshooting

### Connection Issues

```bash
# Test MSSQL connection
sqlcmd -S localhost -U sa -P YourPassword123! -Q "SELECT @@VERSION"

# Test PostgreSQL connection
psql -h localhost -U postgres -d targetdb -c "SELECT version();"
```

### Check Logs

The service uses structured logging with Zap. All operations are logged with:
- Timestamp
- Log level
- Source/target tables
- Duration
- Error details (if any)

### Common Issues

1. **"Failed to connect to database"**: Check connection strings and credentials
2. **"Table not found"**: Verify source table exists and schema is correct
3. **"Permission denied"**: Ensure database users have proper permissions
4. **"Type conversion error"**: Check for unsupported data types

## 📦 Project Structure

```
.
├── cmd/
│   └── syncservice/
│       └── main.go           # Application entry point
├── internal/
│   ├── config/
│   │   └── config.go         # Configuration parser
│   ├── database/
│   │   └── database.go       # Database connections
│   ├── sync/
│   │   └── sync.go           # Sync engine logic
│   ├── actor/
│   │   └── sync_actor.go     # Proto.Actor implementation
│   └── api/
│       ├── server.go         # Gin server
│       └── handlers.go       # API handlers
├── config/
│   └── sync-config.yaml      # Master configuration
├── frontend/
│   ├── public/
│   ├── src/
│   │   ├── App.js           # React main component
│   │   ├── App.css          # Styles
│   │   └── index.js         # React entry point
│   └── package.json
├── go.mod
├── go.sum
└── README.md
```

## 🤝 Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## 📄 License

This project is licensed under the MIT License.

## 🙏 Acknowledgments

- [Proto.Actor](https://proto.actor/) - Actor framework
- [Gin](https://github.com/gin-gonic/gin) - Web framework
- [Zap](https://github.com/uber-go/zap) - Logging
- [SQLx](https://github.com/jmoiron/sqlx) - SQL extensions
- [React.js](https://reactjs.org/) - Frontend framework

## 📞 Support

For issues, questions, or contributions, please open an issue on GitHub.

---

**Made with ❤️ using Go, Proto.Actor, and React.js**
