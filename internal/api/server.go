package api

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/AsynkronIT/protoactor-go/actor"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"mssql-postgres-sync/internal/config"
)

// Server represents the API server
type Server struct {
	Config      *config.Config
	Logger      *zap.Logger
	Handler     *APIHandler
	HTTPServer  *http.Server
	ActorSystem *actor.ActorSystem
}

// NewServer creates a new API server
func NewServer(cfg *config.Config, logger *zap.Logger, coordinatorPID *actor.PID, actorSystem *actor.ActorSystem) *Server {
	handler := NewAPIHandler(cfg, logger, coordinatorPID, actorSystem)

	return &Server{
		Config:      cfg,
		Logger:      logger,
		Handler:     handler,
		ActorSystem: actorSystem,
	}
}

// Start starts the API server
func (s *Server) Start() error {
	// Set Gin mode
	gin.SetMode(gin.ReleaseMode)

	// Create router
	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(s.loggerMiddleware())

	// CORS middleware
	if s.Config.API.EnableCORS {
		router.Use(cors.New(cors.Config{
			AllowOrigins:     []string{"*"},
			AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
			AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
			ExposeHeaders:    []string{"Content-Length"},
			AllowCredentials: true,
			MaxAge:           12 * time.Hour,
		}))
	}

	// Routes
	api := router.Group("/api")
	{
		api.GET("/health", s.Handler.HealthCheck)
		api.GET("/status", s.Handler.GetStatus)
		api.POST("/sync", s.Handler.TriggerSync)
	}

	// Serve static frontend files
	router.Static("/static", "./frontend/build/static")
	router.StaticFile("/", "./frontend/build/index.html")
	router.StaticFile("/favicon.ico", "./frontend/build/favicon.ico")
	router.NoRoute(func(c *gin.Context) {
		c.File("./frontend/build/index.html")
	})

	// Create HTTP server
	addr := fmt.Sprintf("%s:%d", s.Config.API.Host, s.Config.API.Port)
	s.HTTPServer = &http.Server{
		Addr:           addr,
		Handler:        router,
		ReadTimeout:    30 * time.Second,
		WriteTimeout:   30 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}

	s.Logger.Info("Starting API server", zap.String("address", addr))

	// Start server
	if err := s.HTTPServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("failed to start server: %w", err)
	}

	return nil
}

// Stop stops the API server
func (s *Server) Stop() error {
	if s.HTTPServer == nil {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	s.Logger.Info("Stopping API server")
	return s.HTTPServer.Shutdown(ctx)
}

// loggerMiddleware creates a Gin middleware for logging
func (s *Server) loggerMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		c.Next()

		end := time.Now()
		latency := end.Sub(start)

		s.Logger.Info("HTTP request",
			zap.String("method", c.Request.Method),
			zap.String("path", path),
			zap.String("query", query),
			zap.Int("status", c.Writer.Status()),
			zap.Duration("latency", latency),
			zap.String("ip", c.ClientIP()),
		)
	}
}
