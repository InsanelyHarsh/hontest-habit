package webserver

import (
	"fmt"
	"log/slog"
	"net/http"

	"github.com/spf13/viper"
)

type Server struct {
	mux *http.ServeMux
}

func (s *Server) InitWebServer() error {
	port := viper.GetInt("APP_PORT")

	mux := http.NewServeMux()
	s.mux = mux

	addr := fmt.Sprintf(":%d", port)
	slog.Info("webserver: starting", "addr", addr)
	if err := http.ListenAndServe(addr, s.mux); err != nil {
		return fmt.Errorf("webserver: listen and serve: %w", err)
	}
	return nil
}

func (s *Server) NewGroup(prefix string) Group {
	grpMux := http.NewServeMux()
	s.mux.Handle(prefix, grpMux)

	return Group{
		prefix: prefix,
		mux:    grpMux,
	}
}

type Group struct {
	prefix string
	mux    *http.ServeMux
}

func (g Group) GET(path string, h http.HandlerFunc) {
	g.mux.HandleFunc("GET "+path, h)
}

func (g Group) POST(path string, h http.HandlerFunc) {
	g.mux.HandleFunc("POST "+path, h)
}
