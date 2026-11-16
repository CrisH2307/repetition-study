// Command api serves the HTTP API. All wiring lives in internal/app.
package main

import (
	"context"
	"errors"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/CrisH2307/repetition-study/internal/app"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	srv, closeDB, err := app.Build(ctx, app.Config{
		DatabaseURL:  os.Getenv("DATABASE_URL"),
		StudentModel: os.Getenv("STUDENT_MODEL"),
		CheckerModel: os.Getenv("CHECKER_MODEL"),
		Log:          slog.Default(),
	})
	if err != nil {
		log.Fatal(err)
	}
	defer closeDB()
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	srv.Addr = ":" + port
	if dir := os.Getenv("WEB_DIR"); dir != "" {
		srv.Handler = withWeb(srv.Handler, dir)
	}
	go func() {
		<-ctx.Done()
		sctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		srv.Shutdown(sctx)
	}()
	log.Printf("listening on %s", srv.Addr)
	if err := srv.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
		log.Fatal(err)
	}
}
