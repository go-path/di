package lib

import (
	"log/slog"
	"net"
	"net/http"
	"os"

	"github.com/go-path/di"
)

type Server struct {
	Router http.Handler `inject:""`
}

func (s *Server) Initialize() {
	l, err := net.Listen("tcp", ":8081")
	if err != nil {
		slog.Error("error starting server", slog.Any("error", err))
	}

	slog.Info("server started", slog.String("addr", l.Addr().String()))

	if err := http.Serve(l, s.Router); err != nil {
		slog.Info("server closed", slog.Any("error", err))
	}

	os.Exit(1)
}

func init() {
	di.Injected[*Server](di.Startup(200))
}
