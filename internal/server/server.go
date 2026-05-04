package server

import (
	"log/slog"
	"net/http"

	"github.com/EternalQ/gtnh-hub/internal/chat"
	"github.com/EternalQ/gtnh-hub/internal/discord"
	"github.com/gorilla/mux"
)

type Server struct {
	r    *mux.Router
	chat *chat.Hub
	ds   *discord.Discord
}

func NewServer(ds *discord.Discord, hub *chat.Hub) *Server {
	s := &Server{
		chat: hub,
		r:    mux.NewRouter(),
		ds:   ds,
	}

	ds.Setup(s.dsReady, s.dsHandler, s.dsDisconnect)

	return s
}

type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
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
	s.r.HandleFunc("/gtnh-chat", s.handleChat)

	return s.r
}
