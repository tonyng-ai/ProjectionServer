package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"syscall"

	"github.com/asynkron/protoactor-go/actor"
	"go.uber.org/zap"

	actorpkg "mssql-postgres-sync/internal/actor"
	"mssql-postgres-sync/internal/api"
	"mssql-postgres-sync/internal/config"
	"mssql-postgres-sync/internal/database"
	syncpkg "mssql-postgres-sync/internal/sync"
)

func main() {
	configPath := flag.String("config", "config/sync-config.yaml", "path to configuration file")
	flag.Parse()

	logger, err := zap.NewProduction()
	if err != nil {
		panic(err)
	}
	defer logger.Sync()

	cfg, err := config.LoadConfig(*configPath)
	if err != nil {
		logger.Fatal("Failed to load configuration", zap.Error(err))
	}

	dbManager, err := database.NewDatabaseManager(cfg, logger)
	if err != nil {
		logger.Fatal("Failed to initialize database connections", zap.Error(err))
	}
	defer func() {
		if err := dbManager.Close(); err != nil {
			logger.Error("Failed to close database connections", zap.Error(err))
		}
	}()

	syncEngine := syncpkg.NewSyncEngine(dbManager, cfg, logger)

	actorSystem := actor.NewActorSystem()

	coordinatorProps := actor.PropsFromProducer(func() actor.Actor {
		return actorpkg.NewCoordinatorActor(syncEngine, cfg, logger, actorSystem)
	})
	coordinatorPID := actorSystem.Root.Spawn(coordinatorProps)

	apiServer := api.NewServer(cfg, logger, coordinatorPID, actorSystem, dbManager)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	serverErr := make(chan error, 1)
	go func() {
		if err := apiServer.Start(); err != nil {
			serverErr <- err
			return
		}
		serverErr <- nil
	}()

	var startErr error
	select {
	case <-ctx.Done():
		logger.Info("Shutdown signal received")
		if err := apiServer.Stop(); err != nil {
			logger.Error("Failed to stop API server", zap.Error(err))
		}
		startErr = <-serverErr
	case err := <-serverErr:
		startErr = err
	}

	if startErr != nil {
		logger.Error("API server exited with error", zap.Error(startErr))
	} else {
		logger.Info("API server exited")
	}

	stopFuture := actorSystem.Root.StopFuture(coordinatorPID)
	if stopFuture != nil {
		stopFuture.Wait()
	}

	logger.Info("Service shutdown complete")
}
