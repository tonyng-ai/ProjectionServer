package actor

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/asynkron/protoactor-go/actor"
	"go.uber.org/zap"

	"mssql-postgres-sync/internal/config"
	syncpkg "mssql-postgres-sync/internal/sync"
)

// Messages
type SyncTableMessage struct {
	TableConfig config.TableConfig
}

type ScheduleSyncMessage struct{}

type SyncResultMessage struct {
	TableName string
	Success   bool
	Error     error
	Duration  time.Duration
}

// SyncActor handles table synchronization with scheduling
type SyncActor struct {
	syncEngine   *syncpkg.SyncEngine
	tableConfig  config.TableConfig
	defaults     config.DefaultConfig
	logger       *zap.Logger
	actorSystem  *actor.ActorSystem
	cancelFunc   context.CancelFunc
	timerMu      sync.Mutex
	nextSchedule *time.Timer
}

// NewSyncActor creates a new sync actor
func NewSyncActor(syncEngine *syncpkg.SyncEngine, tableConfig config.TableConfig, defaults config.DefaultConfig, logger *zap.Logger, actorSystem *actor.ActorSystem) actor.Actor {
	return &SyncActor{
		syncEngine:  syncEngine,
		tableConfig: tableConfig,
		defaults:    defaults,
		logger:      logger,
		actorSystem: actorSystem,
	}
}

// Receive handles incoming messages
func (a *SyncActor) Receive(ctx actor.Context) {
	switch ctx.Message().(type) {
	case *actor.Started:
		a.logger.Info("SyncActor started",
			zap.String("source_table", a.tableConfig.SourceTable),
			zap.String("target_table", a.tableConfig.TargetTable),
		)

		// Start scheduled sync if enabled
		if a.tableConfig.GetProtoActorTrigger(a.defaults) {
			ctx.Send(ctx.Self(), &ScheduleSyncMessage{})
		}

	case *ScheduleSyncMessage:
		a.performSync(ctx)
		// Schedule next sync
		if a.tableConfig.GetProtoActorTrigger(a.defaults) {
			a.scheduleNextSync(ctx)
		}

	case *SyncTableMessage:
		// Manual trigger
		a.performSync(ctx)

	case *actor.Stopping:
		a.logger.Info("SyncActor stopping",
			zap.String("source_table", a.tableConfig.SourceTable),
		)
		if a.cancelFunc != nil {
			a.cancelFunc()
			a.cancelFunc = nil
		}
		a.stopSchedule()

	case *actor.Stopped:
		a.logger.Info("SyncActor stopped",
			zap.String("source_table", a.tableConfig.SourceTable),
		)
	}
}

// performSync executes the synchronization
func (a *SyncActor) performSync(ctx actor.Context) {
	startTime := time.Now()

	a.logger.Info("Performing sync",
		zap.String("source_table", a.tableConfig.SourceTable),
		zap.String("target_table", a.tableConfig.TargetTable),
	)

	// Create context with timeout
	syncCtx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	a.cancelFunc = cancel
	defer func() {
		cancel()
		a.cancelFunc = nil
	}()

	// Perform sync
	err := a.syncEngine.SyncTable(syncCtx, a.tableConfig)
	duration := time.Since(startTime)

	result := &SyncResultMessage{
		TableName: a.tableConfig.TargetTable,
		Success:   err == nil,
		Error:     err,
		Duration:  duration,
	}

	if err != nil {
		a.logger.Error("Sync failed",
			zap.String("table", a.tableConfig.TargetTable),
			zap.Error(err),
			zap.Duration("duration", duration),
		)
	} else {
		a.logger.Info("Sync completed successfully",
			zap.String("table", a.tableConfig.TargetTable),
			zap.Duration("duration", duration),
		)
	}

	// Send result to parent (coordinator)
	if ctx.Parent() != nil {
		ctx.Send(ctx.Parent(), result)
	}
}

// scheduleNextSync schedules the next sync operation
func (a *SyncActor) scheduleNextSync(ctx actor.Context) {
	refreshRate := time.Duration(a.tableConfig.GetRefreshRate(a.defaults)) * time.Second
	if refreshRate <= 0 {
		refreshRate = time.Second
	}

	a.logger.Info("Scheduling next sync",
		zap.String("table", a.tableConfig.TargetTable),
		zap.Duration("refresh_rate", refreshRate),
	)

	pid := ctx.Self()

	a.timerMu.Lock()
	if a.nextSchedule != nil {
		a.nextSchedule.Stop()
	}
	a.nextSchedule = time.AfterFunc(refreshRate, func() {
		if a.actorSystem != nil {
			a.actorSystem.Root.Send(pid, &ScheduleSyncMessage{})
		}
	})
	a.timerMu.Unlock()
}

func (a *SyncActor) stopSchedule() {
	a.timerMu.Lock()
	if a.nextSchedule != nil {
		a.nextSchedule.Stop()
		a.nextSchedule = nil
	}
	a.timerMu.Unlock()
}

// CoordinatorActor coordinates all sync actors
type CoordinatorActor struct {
	syncEngine  *syncpkg.SyncEngine
	config      *config.Config
	logger      *zap.Logger
	syncActors  map[string]*actor.PID
	actorSystem *actor.ActorSystem
}

// NewCoordinatorActor creates a new coordinator actor
func NewCoordinatorActor(syncEngine *syncpkg.SyncEngine, cfg *config.Config, logger *zap.Logger, actorSystem *actor.ActorSystem) actor.Actor {
	return &CoordinatorActor{
		syncEngine:  syncEngine,
		config:      cfg,
		logger:      logger,
		syncActors:  make(map[string]*actor.PID),
		actorSystem: actorSystem,
	}
}

// Receive handles incoming messages
func (c *CoordinatorActor) Receive(ctx actor.Context) {
	switch msg := ctx.Message().(type) {
	case *actor.Started:
		c.logger.Info("CoordinatorActor started")
		c.startSyncActors(ctx)

	case *SyncResultMessage:
		// Log sync results
		if msg.Success {
			c.logger.Info("Table sync result: SUCCESS",
				zap.String("table", msg.TableName),
				zap.Duration("duration", msg.Duration),
			)
		} else {
			c.logger.Error("Table sync result: FAILED",
				zap.String("table", msg.TableName),
				zap.Error(msg.Error),
				zap.Duration("duration", msg.Duration),
			)
		}

	case *TriggerSyncMessage:
		// Manual trigger for specific table
		if pid, ok := c.syncActors[msg.TableName]; ok {
			ctx.Send(pid, &SyncTableMessage{TableConfig: msg.TableConfig})
			c.logger.Info("Triggered manual sync", zap.String("table", msg.TableName))
		} else {
			c.logger.Warn("Sync actor not found", zap.String("table", msg.TableName))
		}

	case *TriggerAllSyncMessage:
		// Trigger all tables
		c.logger.Info("Triggering sync for all tables")
		for tableName, pid := range c.syncActors {
			// Find config for this table
			for _, tc := range c.config.Tables {
				if tc.TargetTable == tableName {
					ctx.Send(pid, &SyncTableMessage{TableConfig: tc})
					break
				}
			}
		}

	case *actor.Stopping:
		c.logger.Info("CoordinatorActor stopping")

	case *actor.Stopped:
		c.logger.Info("CoordinatorActor stopped")
	}
}

// startSyncActors starts all sync actors based on configuration
func (c *CoordinatorActor) startSyncActors(ctx actor.Context) {
	for _, tableConfig := range c.config.Tables {
		actorName := fmt.Sprintf("sync-%s", sanitizeActorName(tableConfig.TargetTable))

		props := actor.PropsFromProducer(func() actor.Actor {
			return NewSyncActor(c.syncEngine, tableConfig, c.config.Defaults, c.logger, c.actorSystem)
		})

		pid, err := ctx.SpawnNamed(props, actorName)
		if err != nil {
			c.logger.Error("Failed to start sync actor",
				zap.String("actor", actorName),
				zap.Error(err),
			)
			continue
		}

		c.syncActors[tableConfig.TargetTable] = pid

		c.logger.Info("Started sync actor",
			zap.String("actor", actorName),
			zap.String("source_table", tableConfig.SourceTable),
			zap.String("target_table", tableConfig.TargetTable),
		)
	}
}

func sanitizeActorName(name string) string {
	sanitized := strings.ReplaceAll(name, " ", "-")
	sanitized = strings.ReplaceAll(sanitized, ".", "-")
	sanitized = strings.ReplaceAll(sanitized, "/", "-")
	sanitized = strings.ReplaceAll(sanitized, "\\", "-")
	return sanitized
}

// TriggerSyncMessage triggers sync for a specific table
type TriggerSyncMessage struct {
	TableName   string
	TableConfig config.TableConfig
}

// TriggerAllSyncMessage triggers sync for all tables
type TriggerAllSyncMessage struct{}
