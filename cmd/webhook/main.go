package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/petechu/idempotent-webhook-relay/internal/config"
	"github.com/petechu/idempotent-webhook-relay/internal/db/migrations"
	"github.com/petechu/idempotent-webhook-relay/internal/handler"
	"github.com/petechu/idempotent-webhook-relay/internal/svc"
	"github.com/pressly/goose/v3"
)

func main() {
	var cfg *config.Config

	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Unable to load config: %v\n", err)
	}

	router := gin.New()

	router.Use(gin.Logger())
	router.Use(gin.Recovery())

	ctx := context.Background()

	conn, err := pgx.Connect(ctx, cfg.DatabaseURL())
	if err != nil {
		log.Fatalf("Unable to connect to database: %v\n", err)
	}
	defer func() {
		if err := conn.Close(ctx); err != nil {
			log.Printf("Error closing database connection: %v\n", err)
		}
	}()

	if err := migrate(ctx, cfg.DatabaseURL()); err != nil {
		log.Fatalf("Migration failed: %v\n", err)
	}

	svcCtx := svc.NewServiceContext(cfg, conn)
	handler.RegisterRoutes(router, svcCtx)

	server := http.Server{
		Addr:    ":3000",
		Handler: router,
	}

	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)

	fmt.Printf("Listening on port: %s\n", server.Addr)
	go func() {
		if err := server.ListenAndServe(); err != nil {
			log.Fatalf("Failed to start server: %v\n", err)
		}
	}()

	<-ch

	fmt.Println("Gracefully shutting down server...")
	ctx, cancel := context.WithDeadline(ctx, time.Now().Add(5*time.Second))
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Error closing server: %v\n", err)
	}
	fmt.Println("Server shut down successfully.")
}

func migrate(ctx context.Context, dbURL string) error {
	db, err := sql.Open("pgx", dbURL)
	if err != nil {
		log.Fatalf("Unable to open database connection: %v\n", err)
	}

	provider, err := goose.NewProvider(goose.DialectPostgres, db, migrations.Embed)
	if err != nil {
		return err
	}

	sources := provider.ListSources()
	for _, s := range sources {
		log.Printf("%-3s %-2v %v\n", s.Type, s.Version, filepath.Base(s.Path))
	}

	results, err := provider.Up(ctx)
	if err != nil {
		return err
	}
	for _, r := range results {
		log.Printf("%-3s %-2v done: %v\n", r.Source.Type, r.Source.Version, r.Duration)
	}

	return nil
}
