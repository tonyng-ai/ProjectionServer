package api

import (
	"net/http"
	"time"

	"github.com/asynkron/protoactor-go/actor"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	actorpkg "mssql-postgres-sync/internal/actor"
	"mssql-postgres-sync/internal/config"
)

// APIHandler handles HTTP requests
type APIHandler struct {
	Config         *config.Config
	Logger         *zap.Logger
	CoordinatorPID *actor.PID
	ActorSystem    *actor.ActorSystem
}

// NewAPIHandler creates a new API handler
func NewAPIHandler(cfg *config.Config, logger *zap.Logger, coordinatorPID *actor.PID, actorSystem *actor.ActorSystem) *APIHandler {
	return &APIHandler{
		Config:         cfg,
		Logger:         logger,
		CoordinatorPID: coordinatorPID,
		ActorSystem:    actorSystem,
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

// HealthCheck returns health status
func (h *APIHandler) HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "healthy",
		"service": "mssql-postgres-sync",
		"time":    time.Now().Format(time.RFC3339),
	})
}
