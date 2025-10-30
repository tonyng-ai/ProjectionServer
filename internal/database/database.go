package database

import (
	"fmt"

	_ "github.com/denisenkom/go-mssqldb"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"go.uber.org/zap"

	"mssql-postgres-sync/internal/config"
)

// DatabaseManager manages database connections
type DatabaseManager struct {
	Source *sqlx.DB
	Target *sqlx.DB
	Logger *zap.Logger
}

// NewDatabaseManager creates a new database manager
func NewDatabaseManager(cfg *config.Config, logger *zap.Logger) (*DatabaseManager, error) {
	// Connect to source database (MSSQL)
	sourceConn := cfg.Source.GetConnectionString()
	logger.Info("Connecting to source database", zap.String("type", cfg.Source.Type))
	
	sourceDB, err := sqlx.Connect("sqlserver", sourceConn)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to source database: %w", err)
	}

	// Test source connection
	if err := sourceDB.Ping(); err != nil {
		sourceDB.Close()
		return nil, fmt.Errorf("failed to ping source database: %w", err)
	}

	logger.Info("Connected to source database successfully")

	// Connect to target database (PostgreSQL)
	targetConn := cfg.Target.GetConnectionString()
	logger.Info("Connecting to target database", zap.String("type", cfg.Target.Type))
	
	targetDB, err := sqlx.Connect("postgres", targetConn)
	if err != nil {
		sourceDB.Close()
		return nil, fmt.Errorf("failed to connect to target database: %w", err)
	}

	// Test target connection
	if err := targetDB.Ping(); err != nil {
		sourceDB.Close()
		targetDB.Close()
		return nil, fmt.Errorf("failed to ping target database: %w", err)
	}

	logger.Info("Connected to target database successfully")

	return &DatabaseManager{
		Source: sourceDB,
		Target: targetDB,
		Logger: logger,
	}, nil
}

// Close closes all database connections
func (dm *DatabaseManager) Close() error {
	var err error
	if dm.Source != nil {
		if e := dm.Source.Close(); e != nil {
			err = e
		}
	}
	if dm.Target != nil {
		if e := dm.Target.Close(); e != nil {
			err = e
		}
	}
	return err
}
