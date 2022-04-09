package api

import "context"

type ServerApi interface {
	Listen(address string) error
	Serve() error
	Shutdown(ctx context.Context) error
}
