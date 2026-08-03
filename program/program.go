package main

import (
	"clipboard/api"
	"clipboard/server"
	"clipboard/storage"
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/kardianos/service"
	"github.com/mlctrez/servicego"
)

type program struct {
	servicego.Defaults
	dbPath  string
	storage api.StorageApi
	address string
	srv     api.ServerApi
	logger  service.Logger
}

func (p *program) Start(_ service.Service) error {
	p.logger = p.Log()

	if strings.TrimSpace(os.Getenv("EXTERNAL_HOST")) == "" {
		return fmt.Errorf("EXTERNAL_HOST environment variable is required (e.g. clipboard.mlctrez.com)")
	}
	clipToken := strings.TrimSpace(os.Getenv("CLIP_TOKEN"))
	if clipToken == "" {
		return fmt.Errorf("CLIP_TOKEN environment variable is required")
	}

	if err := os.MkdirAll(filepath.Dir(p.dbPath), 0755); err != nil {
		return err
	}

	p.storage = storage.New()
	if err := p.storage.Open(p.dbPath, time.Second*5); err != nil {
		return err
	}

	p.srv = server.New(p.storage, p.logger, clipToken)
	if err := p.srv.Listen(p.address); err != nil {
		return err
	}

	go func() {
		err := p.srv.Serve()
		if err != http.ErrServerClosed {
			p.logger.Warning("unexpected error on http server exit", err)
		}
	}()

	return nil
}

func (p *program) Stop(_ service.Service) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if p.srv != nil {
		if err := p.srv.Shutdown(ctx); err != nil {
			_ = p.logger.Warning("error shutting down server", err)
		}
	}

	if p.storage != nil {
		if err := p.storage.Close(); err != nil {
			_ = p.logger.Warning("error closing storage", err)
			return err
		}
	}
	return nil
}

func main() {
	servicego.Run(&program{dbPath: "clipboard.storage", address: ":8777"})
}
