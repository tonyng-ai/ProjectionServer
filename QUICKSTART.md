# Quick Start Guide

## 🚀 Get Started in 5 Minutes

### Prerequisites Check
```bash
# Check Go version (need 1.21+)
go version

# Check Node.js version (need 16+)
node --version

# Check npm
npm --version
```

### Step 1: Setup Databases

**Option A: Use Docker Compose (Easiest)**
```bash
# Start MSSQL and PostgreSQL
docker-compose up -d mssql postgres

# Wait 30 seconds for databases to initialize
sleep 30
```

**Option B: Use Existing Databases**
- Update `config/sync-config.yaml` with your connection details

### Step 2: Create Sample Data in MSSQL

```sql
-- Connect to MSSQL
USE SourceDB;

-- Create sample table
CREATE TABLE dbo.Users (
    UserID INT PRIMARY KEY IDENTITY(1,1),
    Username NVARCHAR(50) NOT NULL,
    Email NVARCHAR(100) NOT NULL,
    CreatedAt DATETIME2 DEFAULT GETDATE(),
    IsActive BIT DEFAULT 1
);

-- Insert sample data
INSERT INTO dbo.Users (Username, Email, IsActive) VALUES
('john_doe', 'john@example.com', 1),
('jane_smith', 'jane@example.com', 1),
('bob_wilson', 'bob@example.com', 0);
```

### Step 3: Configure Sync

Edit `config/sync-config.yaml`:
```yaml
tables:
  - source_table: dbo.Users
    target_table: public.users
    sync_action: full
    refresh_rate: 60  # Sync every minute
    proto_actor_trigger: true
    webapi_trigger: true
```

### Step 4: Install Dependencies

```bash
# Backend dependencies
go mod download

# Frontend dependencies
cd frontend && npm install && cd ..
```

### Step 5: Run the Service

**Development Mode (Two terminals):**

Terminal 1:
```bash
go run cmd/syncservice/main.go
```

Terminal 2:
```bash
cd frontend && npm start
```

**Production Mode:**
```bash
# Build frontend
cd frontend && npm run build && cd ..

# Run backend
go run cmd/syncservice/main.go
```

### Step 6: Access the Dashboard

Open your browser: `http://localhost:8080` (production) or `http://localhost:3000` (development)

### Step 7: Test the Sync

1. Click "Sync All Tables" button in the dashboard
2. Watch the status change to "Syncing..." then "Success!"
3. Verify data in PostgreSQL:

```sql
-- Connect to PostgreSQL
\c targetdb

-- Check synced data
SELECT * FROM public.users;
```

## 🎯 What's Happening?

1. **Automatic Sync**: Tables sync automatically every 60 seconds (configurable)
2. **Manual Trigger**: Click "Sync Now" for immediate sync
3. **Table Creation**: Target tables are created automatically
4. **Type Mapping**: Data types are automatically converted
5. **Real-time Status**: Dashboard shows live sync status

## 🧪 Testing Different Scenarios

### Test 1: Field Filtering
```yaml
tables:
  - source_table: dbo.Users
    target_table: public.users_minimal
    sync_action: full
    fields:
      - UserID
      - Username
      - Email
```

### Test 2: Filtered Sync
```yaml
tables:
  - source_table: dbo.Users
    target_table: public.active_users
    sync_action: full
    filter: "IsActive = 1"
```

### Test 3: Different Refresh Rates
```yaml
tables:
  - source_table: dbo.Orders
    target_table: public.orders
    sync_action: full
    refresh_rate: 300  # 5 minutes
    
  - source_table: dbo.Products
    target_table: public.products
    sync_action: full
    refresh_rate: 3600  # 1 hour
```

## 📊 API Testing

### Check Status
```bash
curl http://localhost:8080/api/status
```

### Trigger Specific Table
```bash
curl -X POST http://localhost:8080/api/sync \
  -H "Content-Type: application/json" \
  -d '{"table_name": "public.users"}'
```

### Trigger All Tables
```bash
curl -X POST http://localhost:8080/api/sync \
  -H "Content-Type: application/json" \
  -d '{"sync_all": true}'
```

## 🐛 Troubleshooting

### Issue: Can't connect to MSSQL
```bash
# Check MSSQL is running
docker ps | grep mssql

# Test connection
sqlcmd -S localhost -U sa -P YourPassword123! -Q "SELECT @@VERSION"
```

### Issue: Can't connect to PostgreSQL
```bash
# Check PostgreSQL is running
docker ps | grep postgres

# Test connection
psql -h localhost -U postgres -d targetdb -c "SELECT version();"
```

### Issue: Frontend won't start
```bash
# Clear node modules and reinstall
cd frontend
rm -rf node_modules package-lock.json
npm install
npm start
```

## 🎓 Next Steps

1. Read the full [README.md](README.md) for detailed documentation
2. Customize your `sync-config.yaml` for your tables
3. Set up proper database credentials
4. Configure SSL/TLS for production
5. Add authentication to API endpoints
6. Set up monitoring and alerting

## 💡 Tips

- Start with small tables to test
- Use field filtering to reduce data transfer
- Set appropriate refresh rates based on your needs
- Monitor logs for any errors
- Use filters to sync only relevant data

## 🆘 Need Help?

- Check logs in the terminal for detailed error messages
- Verify database connections manually
- Ensure firewall allows database connections
- Check the GitHub issues page

Happy syncing! 🎉
