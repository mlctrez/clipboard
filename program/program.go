package main

import (
	"clipboard/api"
	"clipboard/server"
	"clipboard/storage"
	"context"
	"net/http"
	"os"
	"path/filepath"
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

	if err := os.MkdirAll(filepath.Dir(p.dbPath), 0755); err != nil {
		return nil
	}

	p.storage = storage.New()
	if err := p.storage.Open(p.dbPath, time.Second*5); err != nil {
		return err
	}

	p.srv = server.New(p.storage, p.logger)
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

	if err := p.srv.Shutdown(ctx); err != nil {
		_ = p.logger.Warning("error shutting down server", err)
	}

	if err := p.storage.Close(); err != nil {
		_ = p.logger.Warning("error closing storage", err)
	}
	return p.storage.Close()
}

func main() {
	servicego.Run(&program{dbPath: "clipboard.storage", address: ":8777"})
}
