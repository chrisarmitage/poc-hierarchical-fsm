package webserver

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"sync"
)

//go:embed static/*
var staticFiles embed.FS

type Server struct {
	clients    map[chan map[string]string]bool
	clientsMux sync.RWMutex
	broadcaster chan map[string]string
}

func NewServer() *Server {
	s := &Server{
		clients:    make(map[chan map[string]string]bool),
		broadcaster: make(chan map[string]string, 100),
	}

	// Start broadcaster goroutine
	go s.broadcastLoop()

	return s
}

func (s *Server) broadcastLoop() {
	for update := range s.broadcaster {
		s.clientsMux.RLock()
		for client := range s.clients {
			select {
			case client <- update:
			default:
				// Client buffer full, skip
			}
		}
		s.clientsMux.RUnlock()
	}
}

func (s *Server) GetBroadcastChannel() chan<- map[string]string {
	return s.broadcaster
}

func (s *Server) Start(addr string) error {
	mux := http.NewServeMux()

	// Serve static files from embedded FS
	staticFS, err := fs.Sub(staticFiles, "static")
	if err != nil {
		return err
	}
	mux.Handle("/", http.FileServer(http.FS(staticFS)))

	// SSE endpoint
	mux.HandleFunc("/events", s.handleSSE)

	log.Printf("Web server starting on %s", addr)
	return http.ListenAndServe(addr, mux)
}

func (s *Server) handleSSE(w http.ResponseWriter, r *http.Request) {
	// Set headers for SSE
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	// Create client channel
	clientChan := make(chan map[string]string, 10)

	// Register client
	s.clientsMux.Lock()
	s.clients[clientChan] = true
	s.clientsMux.Unlock()

	// Send connected message
	msg := map[string]string{
		"type":    "connected",
		"message": "Connected to FSM visualizer",
	}
	data, _ := json.Marshal(msg)
	fmt.Fprintf(w, "data: %s\n\n", data)
	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}

	// Stream updates
	ctx := r.Context()
	for {
		select {
		case <-ctx.Done():
			// Client disconnected
			s.clientsMux.Lock()
			delete(s.clients, clientChan)
			s.clientsMux.Unlock()
			close(clientChan)
			return
		case update := <-clientChan:
			data, err := json.Marshal(update)
			if err != nil {
				continue
			}
			fmt.Fprintf(w, "data: %s\n\n", data)
			if f, ok := w.(http.Flusher); ok {
				f.Flush()
			}
		}
	}
}

func (s *Server) BroadcastUpdate(update map[string]string) {
	s.broadcaster <- update
}