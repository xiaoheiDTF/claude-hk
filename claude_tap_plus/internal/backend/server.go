package backend

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/liaohch3/claude-tap/claude_tap_plus/internal/backend/api"
	"github.com/liaohch3/claude-tap/claude_tap_plus/internal/backend/service"
	"github.com/liaohch3/claude-tap/claude_tap_plus/internal/backend/store"
)

type Server struct {
	cfg   Config
	store *store.SQLiteStore
}

func NewServer(cfg Config) (*Server, error) {
	s, err := store.NewSQLiteStore(cfg.DBPath)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}
	return &Server{cfg: cfg, store: s}, nil
}

func (s *Server) Start() error {
	issueSvc := service.NewIssueService(s.store.Issues())

	router := api.NewRouter(api.Handlers{
		Issue: api.NewIssueHandler(issueSvc),
	})

	srv := &http.Server{
		Addr:    s.cfg.Addr(),
		Handler: router,
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-quit
		log.Println("shutting down backend server...")
		srv.Shutdown(context.Background())
	}()

	log.Printf("backend server listening on %s", s.cfg.Addr())
	if err := srv.ListenAndServe(); err != http.ErrServerClosed {
		return err
	}
	return nil
}

func (s *Server) Close() error {
	return s.store.Close()
}
