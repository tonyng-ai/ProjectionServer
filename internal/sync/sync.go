package sync

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"go.uber.org/zap"

	"mssql-postgres-sync/internal/config"
	"mssql-postgres-sync/internal/database"
)

// SyncEngine handles the synchronization logic
type SyncEngine struct {
	DB     *database.DatabaseManager
	Config *config.Config
	Logger *zap.Logger
}

// NewSyncEngine creates a new sync engine
func NewSyncEngine(db *database.DatabaseManager, cfg *config.Config, logger *zap.Logger) *SyncEngine {
	return &SyncEngine{
		DB:     db,
		Config: cfg,
		Logger: logger,
	}
}

// SyncTable synchronizes a single table from source to target
func (se *SyncEngine) SyncTable(ctx context.Context, tableConfig config.TableConfig) error {
	startTime := time.Now()
	logger := se.Logger.With(
		zap.String("source_table", tableConfig.SourceTable),
		zap.String("target_table", tableConfig.TargetTable),
	)

	logger.Info("Starting table sync")

	// Step 1: Get source table schema
	columns, err := se.getSourceColumns(tableConfig.SourceTable, tableConfig.Fields)
	if err != nil {
		return fmt.Errorf("failed to get source columns: %w", err)
	}

	logger.Info("Retrieved source columns", zap.Int("count", len(columns)))

	// Step 2: Create target table if it doesn't exist
	if se.Config.Defaults.CreateTargetTable {
		if err := se.createTargetTable(tableConfig.TargetTable, columns); err != nil {
			return fmt.Errorf("failed to create target table: %w", err)
		}
	}

	// Step 3: Fetch data from source
	data, err := se.fetchSourceData(tableConfig, columns)
	if err != nil {
		return fmt.Errorf("failed to fetch source data: %w", err)
	}

	logger.Info("Fetched source data", zap.Int("rows", len(data)))

	// Step 4: Sync data to target (truncate and insert for full sync)
	if err := se.syncToTarget(tableConfig.TargetTable, columns, data); err != nil {
		return fmt.Errorf("failed to sync to target: %w", err)
	}

	duration := time.Since(startTime)
	logger.Info("Table sync completed",
		zap.Duration("duration", duration),
		zap.Int("rows_synced", len(data)),
	)

	return nil
}

// getSourceColumns retrieves column information from source table
func (se *SyncEngine) getSourceColumns(tableName string, requestedFields []string) ([]ColumnInfo, error) {
	parts := strings.Split(tableName, ".")
	var schema, table string
	if len(parts) == 2 {
		schema = parts[0]
		table = parts[1]
	} else {
		schema = "dbo"
		table = tableName
	}

	query := `
		SELECT 
			COLUMN_NAME,
			DATA_TYPE,
			CHARACTER_MAXIMUM_LENGTH,
			NUMERIC_PRECISION,
			NUMERIC_SCALE,
			IS_NULLABLE
		FROM INFORMATION_SCHEMA.COLUMNS
		WHERE TABLE_SCHEMA = @p1 AND TABLE_NAME = @p2
		ORDER BY ORDINAL_POSITION
	`

	rows, err := se.DB.Source.Queryx(query, schema, table)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var columns []ColumnInfo
	for rows.Next() {
		var col ColumnInfo
		var charLen, numPrec, numScale sql.NullInt64
		var isNullable string

		err := rows.Scan(&col.Name, &col.DataType, &charLen, &numPrec, &numScale, &isNullable)
		if err != nil {
			return nil, err
		}

		col.Length = int(charLen.Int64)
		col.Precision = int(numPrec.Int64)
		col.Scale = int(numScale.Int64)
		col.Nullable = (isNullable == "YES")

		// Filter by requested fields if specified
		if len(requestedFields) > 0 {
			found := false
			for _, field := range requestedFields {
				if strings.EqualFold(field, col.Name) {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}

		columns = append(columns, col)
	}

	return columns, nil
}

// createTargetTable creates the target table if it doesn't exist
func (se *SyncEngine) createTargetTable(tableName string, columns []ColumnInfo) error {
	// Check if table exists
	parts := strings.Split(tableName, ".")
	var schema, table string
	if len(parts) == 2 {
		schema = parts[0]
		table = parts[1]
	} else {
		schema = "public"
		table = tableName
	}

	checkQuery := `
		SELECT EXISTS (
			SELECT FROM information_schema.tables 
			WHERE table_schema = $1 AND table_name = $2
		)
	`

	var exists bool
	err := se.DB.Target.QueryRow(checkQuery, schema, table).Scan(&exists)
	if err != nil {
		return err
	}

	if exists {
		se.Logger.Info("Target table already exists", zap.String("table", tableName))
		return nil
	}

	// Build CREATE TABLE statement
	var colDefs []string
	for _, col := range columns {
		pgType := mapMSSQLToPostgreSQL(col)
		nullable := ""
		if !col.Nullable {
			nullable = " NOT NULL"
		}
		colDefs = append(colDefs, fmt.Sprintf("\"%s\" %s%s", col.Name, pgType, nullable))
	}

	createQuery := fmt.Sprintf("CREATE TABLE %s (\n  %s\n)", tableName, strings.Join(colDefs, ",\n  "))

	se.Logger.Info("Creating target table", zap.String("query", createQuery))

	_, err = se.DB.Target.Exec(createQuery)
	if err != nil {
		return err
	}

	se.Logger.Info("Target table created successfully", zap.String("table", tableName))
	return nil
}

// fetchSourceData retrieves data from source table
func (se *SyncEngine) fetchSourceData(tableConfig config.TableConfig, columns []ColumnInfo) ([]map[string]interface{}, error) {
	// Build column list
	var columnNames []string
	for _, col := range columns {
		columnNames = append(columnNames, fmt.Sprintf("[%s]", col.Name))
	}

	// Build query
	query := fmt.Sprintf("SELECT %s FROM %s", strings.Join(columnNames, ", "), tableConfig.SourceTable)
	
	// Add filter if specified
	if tableConfig.Filter != "" {
		query += fmt.Sprintf(" WHERE %s", tableConfig.Filter)
	}

	se.Logger.Info("Fetching source data", zap.String("query", query))

	rows, err := se.DB.Source.Queryx(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []map[string]interface{}
	for rows.Next() {
		row := make(map[string]interface{})
		if err := rows.MapScan(row); err != nil {
			return nil, err
		}
		results = append(results, row)
	}

	return results, nil
}

// syncToTarget synchronizes data to target table
func (se *SyncEngine) syncToTarget(tableName string, columns []ColumnInfo, data []map[string]interface{}) error {
	if len(data) == 0 {
		se.Logger.Info("No data to sync", zap.String("table", tableName))
		return nil
	}

	// Start transaction
	tx, err := se.DB.Target.Beginx()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Truncate target table
	truncateQuery := fmt.Sprintf("TRUNCATE TABLE %s", tableName)
	se.Logger.Info("Truncating target table", zap.String("table", tableName))
	
	if _, err := tx.Exec(truncateQuery); err != nil {
		return err
	}

	// Build INSERT statement
	var columnNames []string
	for _, col := range columns {
		columnNames = append(columnNames, fmt.Sprintf("\"%s\"", col.Name))
	}

	placeholders := make([]string, len(columnNames))
	for i := range placeholders {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
	}

	insertQuery := fmt.Sprintf(
		"INSERT INTO %s (%s) VALUES (%s)",
		tableName,
		strings.Join(columnNames, ", "),
		strings.Join(placeholders, ", "),
	)

	// Prepare statement
	stmt, err := tx.Preparex(insertQuery)
	if err != nil {
		return err
	}
	defer stmt.Close()

	// Insert data
	for _, row := range data {
		values := make([]interface{}, len(columns))
		for i, col := range columns {
			values[i] = row[col.Name]
		}

		if _, err := stmt.Exec(values...); err != nil {
			se.Logger.Error("Failed to insert row", zap.Error(err), zap.Any("values", values))
			return err
		}
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return err
	}

	se.Logger.Info("Data synced successfully", 
		zap.String("table", tableName),
		zap.Int("rows", len(data)),
	)

	return nil
}

// ColumnInfo represents database column information
type ColumnInfo struct {
	Name      string
	DataType  string
	Length    int
	Precision int
	Scale     int
	Nullable  bool
}

// mapMSSQLToPostgreSQL maps MSSQL data types to PostgreSQL
func mapMSSQLToPostgreSQL(col ColumnInfo) string {
	switch strings.ToLower(col.DataType) {
	case "int":
		return "INTEGER"
	case "bigint":
		return "BIGINT"
	case "smallint":
		return "SMALLINT"
	case "tinyint":
		return "SMALLINT"
	case "bit":
		return "BOOLEAN"
	case "decimal", "numeric":
		if col.Precision > 0 {
			return fmt.Sprintf("NUMERIC(%d,%d)", col.Precision, col.Scale)
		}
		return "NUMERIC"
	case "money", "smallmoney":
		return "NUMERIC(19,4)"
	case "float":
		return "DOUBLE PRECISION"
	case "real":
		return "REAL"
	case "date":
		return "DATE"
	case "datetime", "datetime2", "smalldatetime":
		return "TIMESTAMP"
	case "time":
		return "TIME"
	case "char":
		if col.Length > 0 {
			return fmt.Sprintf("CHAR(%d)", col.Length)
		}
		return "CHAR(1)"
	case "varchar":
		if col.Length > 0 && col.Length <= 10485760 {
			return fmt.Sprintf("VARCHAR(%d)", col.Length)
		}
		return "TEXT"
	case "nchar":
		if col.Length > 0 {
			return fmt.Sprintf("CHAR(%d)", col.Length)
		}
		return "CHAR(1)"
	case "nvarchar":
		if col.Length > 0 && col.Length <= 10485760 {
			return fmt.Sprintf("VARCHAR(%d)", col.Length)
		}
		return "TEXT"
	case "text", "ntext":
		return "TEXT"
	case "uniqueidentifier":
		return "UUID"
	case "varbinary", "binary", "image":
		return "BYTEA"
	case "xml":
		return "XML"
	default:
		return "TEXT"
	}
}
