package server

import (
	"bufio"
	"fmt"
	"log/slog"
	"net"
	"net/http"

	"github.com/EternalQ/gtnh-hub/internal/discord"
	"github.com/EternalQ/gtnh-hub/internal/game"
	"github.com/EternalQ/gtnh-hub/internal/hub"
	"github.com/gorilla/mux"
)

type Server struct {
	r    *mux.Router
	hub  *hub.Hub
	ds   *discord.Discord
	game *game.Instance
}

func NewServer(ds *discord.Discord, hub *hub.Hub) (*Server, error) {
	s := &Server{
		hub:  hub,
		r:    mux.NewRouter(),
		ds:   ds,
		game: game.NewInstance(),
	}

	if err := ds.Setup(s.dsConnect, s.dsHandler, s.dsDisconnect, s.onlineRefresher); err != nil {
		return nil, err
	}

	return s, nil
}

type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	h, ok := rw.ResponseWriter.(http.Hijacker)
	if !ok {
		return nil, nil, fmt.Errorf("ResponseWriter does not implement http.Hijacker")
	}
	return h.Hijack()
}

func mwLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// start := time.Now()

		wrappedWriter := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		next.ServeHTTP(wrappedWriter, r)

		slog.Info("incoming request",
			slog.String("method", r.Method),
			slog.String("path", r.URL.Path),
			slog.Int("status", wrappedWriter.statusCode),
			// slog.Duration("duration", time.Since(start)),
			// slog.String("ip", r.RemoteAddr),
		)
	})
}

func (s *Server) Routes() *mux.Router {
	s.r.Use(mwLogger)
	s.r.HandleFunc("/ws/gtnh", s.handleGtnh)

	return s.r
}
