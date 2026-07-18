package webserver

import (
	"encoding/json"
	goerrors "errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/spf13/viper"

	"github.com/insanelyharsh/hontest-habit/internal/common/errors"
	"github.com/insanelyharsh/hontest-habit/internal/webserver/middlewares"
)

type Controller interface {
	Routes(grp Group)
}

type Server struct {
	mux *http.ServeMux
}

// NewServer creates a Server with its mux ready, so routes can be
// registered (via NewGroup) before InitWebServer starts serving.
func NewServer() *Server {
	return &Server{mux: http.NewServeMux()}
}

func (s *Server) InitWebServer() error {
	port := viper.GetInt("APP_PORT")

	addr := fmt.Sprintf(":%d", port)
	slog.Info("webserver: starting", "addr", addr)
	if err := http.ListenAndServe(addr, middlewares.TraceID(s.mux)); err != nil {
		return fmt.Errorf("webserver: listen and serve: %w", err)
	}
	return nil
}

// Register mounts ctrl's routes under prefix, applying mw (in order) to
// the group before calling ctrl.Routes.
func (s *Server) Register(prefix string, ctrl Controller, mw ...func(http.Handler) http.Handler) {
	group := s.NewGroup(prefix)
	for _, m := range mw {
		group = group.Use(m)
	}
	ctrl.Routes(group)
}

func (s *Server) NewGroup(prefix string) Group {
	grpMux := http.NewServeMux()
	// prefix is a subtree pattern (e.g. "/auth/"), so net/http dispatches
	// the *original* path ("/auth/signup") to grpMux — strip it back down
	// to what the group's own patterns ("/signup") expect.
	s.mux.Handle(prefix, http.StripPrefix(strings.TrimSuffix(prefix, "/"), grpMux))

	return Group{
		prefix: prefix,
		mux:    grpMux,
	}
}

type Group struct {
	prefix      string
	mux         *http.ServeMux
	middlewares []func(http.Handler) http.Handler
}

// Use returns a copy of g with mw appended to its middleware chain, applied
// (outermost-first, in Use call order) to every route registered on the
// returned Group. g itself is unaffected, so a Group can be reused to
// register some routes protected and others not:
//
//	public := server.NewGroup("/blocklist/")
//	protected := public.Use(middlewares.Authenticate(jwtCfg))
//	protected.POST("/entries", ...) // requires auth
func (g Group) Use(mw func(http.Handler) http.Handler) Group {
	g.middlewares = append(append([]func(http.Handler) http.Handler{}, g.middlewares...), mw)
	return g
}

func (g Group) wrapChain(h HandlerFunc) http.HandlerFunc {
	var handler http.Handler = http.HandlerFunc(Wrap(h))
	for i := len(g.middlewares) - 1; i >= 0; i-- {
		handler = g.middlewares[i](handler)
	}
	return handler.ServeHTTP
}

func (g Group) GET(path string, h HandlerFunc) {
	g.mux.HandleFunc("GET "+path, g.wrapChain(h))
}

func (g Group) POST(path string, h HandlerFunc) {
	g.mux.HandleFunc("POST "+path, g.wrapChain(h))
}

func (g Group) DELETE(path string, h HandlerFunc) {
	g.mux.HandleFunc("DELETE "+path, g.wrapChain(h))
}

// HandlerFunc is like http.HandlerFunc but returns an error, letting
// handlers report failures instead of writing them directly.
type HandlerFunc func(w http.ResponseWriter, r *http.Request) error

// Wrap adapts a HandlerFunc into an http.HandlerFunc: on error, it maps the
// error to an HTTP status via errors.StatusCode and writes a JSON body
// containing only the client-safe message for a categorized *errors.HError
// (never the raw underlying cause, which may contain internal details),
// falling back to a generic message for uncategorized errors.
func Wrap(h HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := h(w, r)
		if err == nil {
			return
		}

		message := "internal server error"
		if he, ok := goerrors.AsType[*errors.HError](err); ok {
			message = he.Message
		}
		slog.Error("webserver: handler error", "path", r.URL.Path, "error", err)
		WriteJSON(w, errors.StatusCode(err), map[string]string{"error": message})
	}
}

// WriteJSON writes body as a JSON response with the given status code.
func WriteJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

// DecodeJSON decodes r's JSON body into dst, returning an errors.BadRequest
// on failure.
func DecodeJSON(r *http.Request, dst any) error {
	if err := json.NewDecoder(r.Body).Decode(dst); err != nil {
		return errors.BadRequest("invalid request body", err)
	}
	return nil
}
