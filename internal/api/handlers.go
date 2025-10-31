package api

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/asynkron/protoactor-go/actor"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	actorpkg "mssql-postgres-sync/internal/actor"
	"mssql-postgres-sync/internal/config"
	"mssql-postgres-sync/internal/database"
)

// APIHandler handles HTTP requests
type APIHandler struct {
	Config         *config.Config
	Logger         *zap.Logger
	CoordinatorPID *actor.PID
	ActorSystem    *actor.ActorSystem
	DBManager      *database.DatabaseManager
}

// NewAPIHandler creates a new API handler
func NewAPIHandler(cfg *config.Config, logger *zap.Logger, coordinatorPID *actor.PID, actorSystem *actor.ActorSystem, dbManager *database.DatabaseManager) *APIHandler {
	return &APIHandler{
		Config:         cfg,
		Logger:         logger,
		CoordinatorPID: coordinatorPID,
		ActorSystem:    actorSystem,
		DBManager:      dbManager,
	}
}

// SyncRequest represents a sync request
type SyncRequest struct {
	TableName string `json:"table_name,omitempty"`
	SyncAll   bool   `json:"sync_all,omitempty"`
}

// SyncResponse represents a sync response
type SyncResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// TableStatus represents table sync status
type TableStatus struct {
	SourceTable       string    `json:"source_table"`
	TargetTable       string    `json:"target_table"`
	RefreshRate       int       `json:"refresh_rate"`
	ProtoActorEnabled bool      `json:"proto_actor_enabled"`
	WebAPIEnabled     bool      `json:"web_api_enabled"`
	LastSync          time.Time `json:"last_sync,omitempty"`
}

// StatusResponse represents the status response
type StatusResponse struct {
	Status string        `json:"status"`
	Tables []TableStatus `json:"tables"`
}

// TriggerSync triggers a sync operation
func (h *APIHandler) TriggerSync(c *gin.Context) {
	var req SyncRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, SyncResponse{
			Success: false,
			Message: "Invalid request: " + err.Error(),
		})
		return
	}

	h.Logger.Info("Received sync trigger request",
		zap.String("table_name", req.TableName),
		zap.Bool("sync_all", req.SyncAll),
	)

	if req.SyncAll {
		// Trigger all tables
		h.ActorSystem.Root.Send(h.CoordinatorPID, &actorpkg.TriggerAllSyncMessage{})

		c.JSON(http.StatusOK, SyncResponse{
			Success: true,
			Message: "Sync triggered for all tables",
		})
		return
	}

	if req.TableName == "" {
		c.JSON(http.StatusBadRequest, SyncResponse{
			Success: false,
			Message: "Either table_name or sync_all must be specified",
		})
		return
	}

	// Find table config
	var tableConfig *config.TableConfig
	for _, tc := range h.Config.Tables {
		if tc.TargetTable == req.TableName {
			tableConfig = &tc
			break
		}
	}

	if tableConfig == nil {
		c.JSON(http.StatusNotFound, SyncResponse{
			Success: false,
			Message: "Table not found: " + req.TableName,
		})
		return
	}

	// Check if WebAPI trigger is enabled
	if !tableConfig.GetWebAPITrigger(h.Config.Defaults) {
		c.JSON(http.StatusForbidden, SyncResponse{
			Success: false,
			Message: "WebAPI trigger is disabled for this table",
		})
		return
	}

	// Trigger sync
	h.ActorSystem.Root.Send(h.CoordinatorPID, &actorpkg.TriggerSyncMessage{
		TableName:   req.TableName,
		TableConfig: *tableConfig,
	})

	c.JSON(http.StatusOK, SyncResponse{
		Success: true,
		Message: "Sync triggered for table: " + req.TableName,
	})
}

// GetStatus returns the current status
func (h *APIHandler) GetStatus(c *gin.Context) {
	var tables []TableStatus

	for _, tc := range h.Config.Tables {
		tables = append(tables, TableStatus{
			SourceTable:       tc.SourceTable,
			TargetTable:       tc.TargetTable,
			RefreshRate:       tc.GetRefreshRate(h.Config.Defaults),
			ProtoActorEnabled: tc.GetProtoActorTrigger(h.Config.Defaults),
			WebAPIEnabled:     tc.GetWebAPITrigger(h.Config.Defaults),
		})
	}

	c.JSON(http.StatusOK, StatusResponse{
		Status: "running",
		Tables: tables,
	})
}

// ListProjections returns all configured projections for the UI
func (h *APIHandler) ListProjections(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"projections": h.Config.Projections,
	})
}

// GetProjectionData returns data for a specific projection view with optional filters and sorting
func (h *APIHandler) GetProjectionData(c *gin.Context) {
	if h.DBManager == nil || h.DBManager.Target == nil {
		h.Logger.Error("Target database not configured for projections")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Target database connection is not available",
		})
		return
	}

	projectionID := c.Param("id")
	projection, ok := h.Config.GetProjectionByID(projectionID)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{
			"error": fmt.Sprintf("Projection not found: %s", projectionID),
		})
		return
	}

	selectClause, sortableColumns := buildSelectClause(projection)
	queryBuilder := strings.Builder{}
	queryBuilder.WriteString("SELECT ")
	queryBuilder.WriteString(selectClause)
	queryBuilder.WriteString(" FROM ")
	queryBuilder.WriteString(quoteQualifiedIdentifier(projection.TargetView))

	filtersMap := c.QueryMap("filters")
	var (
		whereClauses   []string
		queryArgs      []interface{}
		appliedFilters = make(map[string]interface{})
		parameterIndex = 1
	)

	for _, filterCfg := range projection.Filters {
		raw, exists := filtersMap[filterCfg.ID]
		if !exists {
			continue
		}

		raw = strings.TrimSpace(raw)
		if raw == "" {
			continue
		}

		columnIdentifier := quoteIdentifier(filterCfg.Column)
		switch strings.ToLower(filterCfg.Type) {
		case "select":
			values := splitAndClean(raw)
			if len(values) == 0 {
				continue
			}
			placeholders := make([]string, 0, len(values))
			for _, value := range values {
				queryArgs = append(queryArgs, value)
				placeholders = append(placeholders, fmt.Sprintf("$%d", parameterIndex))
				parameterIndex++
			}
			whereClauses = append(whereClauses, fmt.Sprintf("%s IN (%s)", columnIdentifier, strings.Join(placeholders, ", ")))
			appliedFilters[filterCfg.ID] = values
		case "number":
			value, err := strconv.ParseFloat(raw, 64)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{
					"error": fmt.Sprintf("Invalid numeric filter for %s", filterCfg.ID),
				})
				return
			}
			queryArgs = append(queryArgs, value)
			whereClauses = append(whereClauses, fmt.Sprintf("%s >= $%d", columnIdentifier, parameterIndex))
			parameterIndex++
			appliedFilters[filterCfg.ID] = value
		default:
			queryArgs = append(queryArgs, raw)
			whereClauses = append(whereClauses, fmt.Sprintf("%s = $%d", columnIdentifier, parameterIndex))
			parameterIndex++
			appliedFilters[filterCfg.ID] = raw
		}
	}

	if len(whereClauses) > 0 {
		queryBuilder.WriteString(" WHERE ")
		queryBuilder.WriteString(strings.Join(whereClauses, " AND "))
	}

	sortColumn := strings.TrimSpace(c.Query("sort"))
	sortDirection := strings.ToUpper(strings.TrimSpace(c.DefaultQuery("direction", "")))
	if sortDirection != "ASC" && sortDirection != "DESC" {
		sortDirection = ""
	}

	if sortColumn == "" && projection.DefaultSort != nil {
		sortColumn = projection.DefaultSort.Column
		sortDirection = strings.ToUpper(projection.DefaultSort.Direction)
	}

	if projection.DefaultSort != nil {
		sortableColumns[strings.ToLower(projection.DefaultSort.Column)] = true
	}

	if sortColumn != "" {
		if !sortableColumns[strings.ToLower(sortColumn)] {
			sortColumn = ""
		} else {
			if sortDirection != "DESC" {
				sortDirection = "ASC"
			}
			queryBuilder.WriteString(" ORDER BY ")
			queryBuilder.WriteString(quoteIdentifier(sortColumn))
			queryBuilder.WriteRune(' ')
			queryBuilder.WriteString(sortDirection)
		}
	}

	query := queryBuilder.String()
	h.Logger.Debug("Executing projection query",
		zap.String("projection_id", projection.ID),
		zap.String("query", query),
		zap.Any("args", queryArgs),
	)

	rows, err := h.DBManager.Target.Queryx(query, queryArgs...)
	if err != nil {
		h.Logger.Error("Failed to query projection data",
			zap.String("projection_id", projection.ID),
			zap.Error(err),
		)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to query projection data",
		})
		return
	}
	defer rows.Close()

	var (
		resultRows  []map[string]interface{}
		totalSums   = make(map[string]float64)
		totalCounts = make(map[string]int)
	)

	for rows.Next() {
		rowData := make(map[string]interface{})
		if err := rows.MapScan(rowData); err != nil {
			h.Logger.Error("Failed to scan projection row",
				zap.String("projection_id", projection.ID),
				zap.Error(err),
			)
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to parse projection row",
			})
			return
		}

		normalizedRow := make(map[string]interface{})
		lowerRow := make(map[string]interface{})
		lowerRaw := make(map[string]interface{})
		for col, val := range rowData {
			normalized := normalizeDBValue(val)
			normalizedRow[col] = normalized
			lowerKey := strings.ToLower(col)
			lowerRow[lowerKey] = normalized
			lowerRaw[lowerKey] = val
		}

		resultRows = append(resultRows, normalizedRow)

		for _, totalCfg := range projection.Totals {
			columnKey := strings.ToLower(totalCfg.Column)
			switch strings.ToLower(totalCfg.Format) {
			case "count":
				if lowerRow[columnKey] != nil {
					totalCounts[columnKey]++
				}
			default:
				if value, ok := lowerRaw[columnKey]; ok {
					if floatVal, converted := valueToFloat64(value); converted {
						totalSums[columnKey] += floatVal
						continue
					}
				}
				if value, ok := lowerRow[columnKey]; ok {
					if floatVal, converted := valueToFloat64(value); converted {
						totalSums[columnKey] += floatVal
					}
				}
			}
		}
	}

	if err := rows.Err(); err != nil {
		h.Logger.Error("Error iterating projection rows",
			zap.String("projection_id", projection.ID),
			zap.Error(err),
		)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Error reading projection rows",
		})
		return
	}

	totalsResponse := make(map[string]interface{})
	for _, totalCfg := range projection.Totals {
		columnKey := strings.ToLower(totalCfg.Column)
		switch strings.ToLower(totalCfg.Format) {
		case "count":
			totalsResponse[totalCfg.Column] = totalCounts[columnKey]
		default:
			totalsResponse[totalCfg.Column] = totalSums[columnKey]
		}
	}

	response := gin.H{
		"projection_id": projection.ID,
		"rows":          resultRows,
		"totals":        totalsResponse,
		"filters":       appliedFilters,
		"meta": gin.H{
			"sort_column":    sortColumn,
			"sort_direction": sortDirection,
			"row_count":      len(resultRows),
		},
	}

	c.JSON(http.StatusOK, response)
}

// HealthCheck returns health status
func (h *APIHandler) HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "healthy",
		"service": "mssql-postgres-sync",
		"time":    time.Now().Format(time.RFC3339),
	})
}

func buildSelectClause(projection *config.ProjectionConfig) (string, map[string]bool) {
	sortableColumns := make(map[string]bool)
	if projection == nil || len(projection.Fields) == 0 {
		return "*", sortableColumns
	}

	columns := make([]string, 0, len(projection.Fields))
	for _, field := range projection.Fields {
		columns = append(columns, quoteIdentifier(field.Column))
		sortable := field.Sortable == nil || (field.Sortable != nil && *field.Sortable)
		if sortable {
			sortableColumns[strings.ToLower(field.Column)] = true
		}
	}

	return strings.Join(columns, ", "), sortableColumns
}

func quoteIdentifier(identifier string) string {
	identifier = strings.TrimSpace(identifier)
	if identifier == "" {
		return ""
	}
	if identifier == "*" {
		return "*"
	}

	parts := strings.Split(identifier, ".")
	for i, part := range parts {
		cleaned := strings.ReplaceAll(part, "\"", "\"\"")
		parts[i] = fmt.Sprintf("\"%s\"", cleaned)
	}
	return strings.Join(parts, ".")
}

func quoteQualifiedIdentifier(identifier string) string {
	return quoteIdentifier(identifier)
}

func splitAndClean(raw string) []string {
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

func normalizeDBValue(value interface{}) interface{} {
	switch v := value.(type) {
	case nil:
		return nil
	case []byte:
		return string(v)
	case time.Time:
		return v.Format(time.RFC3339)
	default:
		return v
	}
}

func valueToFloat64(value interface{}) (float64, bool) {
	switch v := value.(type) {
	case nil:
		return 0, false
	case float64:
		return v, true
	case float32:
		return float64(v), true
	case int64:
		return float64(v), true
	case int32:
		return float64(v), true
	case int:
		return float64(v), true
	case uint64:
		return float64(v), true
	case uint32:
		return float64(v), true
	case uint:
		return float64(v), true
	case string:
		f, err := strconv.ParseFloat(strings.TrimSpace(v), 64)
		if err == nil {
			return f, true
		}
	case []byte:
		f, err := strconv.ParseFloat(strings.TrimSpace(string(v)), 64)
		if err == nil {
			return f, true
		}
	default:
		str := strings.TrimSpace(fmt.Sprint(v))
		if str == "" {
			return 0, false
		}
		f, err := strconv.ParseFloat(str, 64)
		if err == nil {
			return f, true
		}
	}
	return 0, false
}
