package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/petechu/idempotent-webhook-relay/webhook/internal/config"
	"github.com/petechu/idempotent-webhook-relay/webhook/internal/handler"
	"github.com/petechu/idempotent-webhook-relay/webhook/internal/svc"
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

	conn, err := pgx.Connect(ctx, "postgres://localhost:5432/idempotent-webhook-relay")
	if err != nil {
		log.Fatalf("Unable to connect to database: %v\n", err)
	}
	defer conn.Close(ctx)

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
