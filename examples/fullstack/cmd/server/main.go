// cmd/server — TaskFlow REST API server.
//
// Usage:
//
//	go run ./cmd/server \
//	    -addr :8080 \
//	    -db-host 127.0.0.1 \
//	    -db-port 3306 \
//	    -db-name taskflow \
//	    -db-user root \
//	    -db-pass "" \
//	    -jwt-secret "change-me-in-production"
package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/achiket/taskflow/internal/api"
	"github.com/achiket/taskflow/internal/auth"
	"github.com/achiket/taskflow/internal/db"
	"github.com/achiket/taskflow/internal/repository"
	"github.com/achiket/taskflow/internal/service"
)

func main() {
	cfg := db.DefaultConfig()

	addr := flag.String("addr", ":8080", "listen address")
	dbHost := flag.String("db-host", cfg.Host, "MySQL host")
	dbPort := flag.Int("db-port", cfg.Port, "MySQL port")
	dbName := flag.String("db-name", cfg.Name, "MySQL database name")
	dbUser := flag.String("db-user", cfg.User, "MySQL user")
	dbPass := flag.String("db-pass", cfg.Password, "MySQL password")
	jwtSecret := flag.String("jwt-secret", "super-secret-key", "JWT signing secret")
	flag.Parse()

	// ── Database ─────────────────────────────────────────────────────────────

	database := db.MustOpen(db.Config{
		Host: *dbHost, Port: *dbPort,
		User: *dbUser, Password: *dbPass, Name: *dbName,
	})
	defer database.Close()

	// ── Migrations ───────────────────────────────────────────────────────────

	if err := database.Migrate("schema.sql"); err != nil {
		log.Fatalf("migration failed: %v", err)
	}

	// ── Repositories ──────────────────────────────────────────────────────────

	repos := &service.Repos{
		User:      repository.NewUserRepo(database),
		Workspace: repository.NewWorkspaceRepo(database),
		Project:   repository.NewProjectRepo(database),
		Task:      repository.NewTaskRepo(database),
		Comment:   repository.NewCommentRepo(database),
		Activity:  repository.NewActivityRepo(database),
		Label:     repository.NewLabelRepo(database),
	}

	// ── Services ──────────────────────────────────────────────────────────────

	authSvc := auth.NewService(database, *jwtSecret)
	wsSvc := service.NewWorkspaceService(repos)
	projSvc := service.NewProjectService(repos)
	taskSvc := service.NewTaskService(repos)
	cmtSvc := service.NewCommentService(repos)
	dashSvc := service.NewDashboardService(repos)

	// ── HTTP Server ───────────────────────────────────────────────────────────

	apiServer := api.NewServer(authSvc, wsSvc, projSvc, taskSvc, cmtSvc, dashSvc)
	httpServer := apiServer.HTTPServer(*addr)

	go func() {
		log.Printf("[server] listening on %s", *addr)
		if err := httpServer.ListenAndServe(); err != nil {
			log.Printf("[server] stopped: %v", err)
		}
	}()

	// ── Graceful shutdown ─────────────────────────────────────────────────────

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("[server] shutting down…")
}
