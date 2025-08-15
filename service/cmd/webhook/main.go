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
)

func main() {
	r := gin.New()

	r.GET("/healthz", func(c *gin.Context) {
		c.String(200, "OK")
	})

	server := http.Server{
		Addr:    ":3000",
		Handler: r,
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
	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(5*time.Second))
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Error closing server: %v\n", err)
	}
	fmt.Println("Server shut down successfully.")
}
