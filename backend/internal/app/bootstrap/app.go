package bootstrap

import (
	"context"
	"fmt"
	"net/http"

	"github.com/OpenNSW/nsw/internal/auth"
	"github.com/OpenNSW/nsw/internal/config"
	"github.com/OpenNSW/nsw/internal/database"
	"github.com/OpenNSW/nsw/internal/form"
	"github.com/OpenNSW/nsw/internal/middleware"
	taskManager "github.com/OpenNSW/nsw/internal/task/manager"
	"github.com/OpenNSW/nsw/internal/uploads"
	workflowmanager "github.com/OpenNSW/nsw/internal/workflow/manager"
	"github.com/OpenNSW/nsw/internal/workflow/router"
	"github.com/OpenNSW/nsw/internal/workflow/service"
)

// App contains initialized HTTP server and cleanup hooks.
type App struct {
	Server *http.Server
	close  func() error
}

// Close releases resources initialized during bootstrap.
func (a *App) Close() error {
	if a == nil || a.close == nil {
		return nil
	}
	return a.close()
}

// Build initializes dependencies and returns a fully wired application server.
func Build(ctx context.Context, cfg *config.Config) (*App, error) {
	db, err := database.New(&cfg.Database)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	if err := database.HealthCheck(db); err != nil {
		_ = database.Close(db)
		return nil, fmt.Errorf("database health check failed: %w", err)
	}

	formService := form.NewFormService(db)
	tm, err := taskManager.NewTaskManager(db, cfg, formService)
	if err != nil {
		_ = database.Close(db)
		return nil, fmt.Errorf("failed to create task manager: %w", err)
	}

	nodeService := service.NewWorkflowNodeService(db)
	templateService := service.NewTemplateService(db)
	wm := workflowmanager.NewManager(db, nodeService, templateService)

	chaService := service.NewCHAService(db)
	hsCodeService := service.NewHSCodeService(db)
	consignmentService := service.NewConsignmentService(db, templateService, wm)
	preConsignmentService := service.NewPreConsignmentService(db, templateService, wm)

	if err := WireManagers(wm, tm); err != nil {
		_ = database.Close(db)
		return nil, fmt.Errorf("failed to wire managers: %w", err)
	}

	hsCodeRouter := router.NewHSCodeRouter(hsCodeService)
	chaRouter := router.NewCHARouter(chaService)
	consignmentRouter := router.NewConsignmentRouter(consignmentService, chaService)
	preConsignmentRouter := router.NewPreConsignmentRouter(preConsignmentService)

	storageDriver, err := uploads.NewStorageFromConfig(ctx, cfg.Storage)
	if err != nil {
		_ = database.Close(db)
		return nil, fmt.Errorf("failed to initialize storage: %w", err)
	}
	uploadService := uploads.NewUploadService(storageDriver)
	uploadHandler := uploads.NewHTTPHandler(uploadService)

	authManager, err := auth.NewManager(db, cfg.Auth)
	if err != nil {
		_ = database.Close(db)
		return nil, fmt.Errorf("failed to create auth manager: %w", err)
	}

	if err := authManager.Health(); err != nil {
		_ = authManager.Close()
		_ = database.Close(db)
		return nil, fmt.Errorf("auth system health check failed: %w", err)
	}

	tmHandler := taskManager.NewHTTPHandler(tm)

	// 1. Initialize Protected Router
	// We remove the /api/v1 prefix from these definitions because
	// we will mount this entire mux under that prefix later.
	protectedMux := http.NewServeMux()
	protectedMux.HandleFunc("POST /tasks", tmHandler.HandleExecuteTask)
	protectedMux.HandleFunc("GET /tasks/{id}", tmHandler.HandleGetTask)
	protectedMux.HandleFunc("GET /hscodes", hsCodeRouter.HandleGetAllHSCodes)
	protectedMux.HandleFunc("GET /chas", chaRouter.HandleGetCHAs)
	protectedMux.HandleFunc("POST /consignments", consignmentRouter.HandleCreateConsignment)
	protectedMux.HandleFunc("GET /consignments/{id}", consignmentRouter.HandleGetConsignmentByID)
	protectedMux.HandleFunc("PUT /consignments/{id}", consignmentRouter.HandleInitializeConsignment)
	protectedMux.HandleFunc("GET /consignments", consignmentRouter.HandleGetConsignments)
	protectedMux.HandleFunc("POST /pre-consignments", preConsignmentRouter.HandleCreatePreConsignment)
	protectedMux.HandleFunc("GET /pre-consignments/{preConsignmentId}", preConsignmentRouter.HandleGetPreConsignmentByID)
	protectedMux.HandleFunc("GET /pre-consignments", preConsignmentRouter.HandleGetTraderPreConsignments)
	protectedMux.HandleFunc("POST /uploads", uploadHandler.Upload)
	protectedMux.HandleFunc("GET /uploads/{key}/content", uploadHandler.DownloadContent)
	protectedMux.HandleFunc("GET /uploads/{key}", uploadHandler.Download)
	protectedMux.HandleFunc("DELETE /uploads/{key}", uploadHandler.Delete)

	// 2. Initialize Main (Public) Router
	mainMux := http.NewServeMux()

	// Public Health Check
	mainMux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		if err := database.HealthCheck(db); err != nil {
			http.Error(w, "Service Unavailable", http.StatusServiceUnavailable)
			return
		}
		if err := authManager.Health(); err != nil {
			http.Error(w, "Service Unavailable", http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok","service":"nsw-backend"}`))
	})

	// 3. Mount Protected Mux with Auth Middleware
	// The trailing slash in "/api/v1/" acts as a prefix match
	mainMux.Handle("/api/v1/", http.StripPrefix("/api/v1", authManager.Middleware()(protectedMux)))

	// 4. Apply Global Middlewares (CORS)
	handler := middleware.CORS(&cfg.CORS)(mainMux)

	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.Server.Port),
		Handler: handler,
	}

	closeFn := func() error {
		authErr := authManager.Close()
		dbErr := database.Close(db)
		if authErr != nil {
			if dbErr != nil {
				return fmt.Errorf("failed to close auth manager: %v; failed to close database: %v", authErr, dbErr)
			}
			return fmt.Errorf("failed to close auth manager: %w", authErr)
		}
		if dbErr != nil {
			return fmt.Errorf("failed to close database: %w", dbErr)
		}
		return nil
	}

	return &App{Server: server, close: closeFn}, nil
}
