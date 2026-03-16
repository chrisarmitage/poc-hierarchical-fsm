package webserver

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"sync"

	"github.com/chrisarmitage/poc-hierarchical-fsm/internal/events"
)

//go:embed static/*
var staticFiles embed.FS

type Server struct {
	clients     map[chan StateUpdate]bool
	clientsMux  sync.RWMutex
	broadcaster chan StateUpdate
	eventsChan  chan events.Event
}

type StateUpdate struct {
	Type      string `json:"type"`             // "state" for FSM listener, "payload" for comms to device
	System    string `json:"system,omitempty"` // e.g. "devicefsm", "SetProtectedValueTask"
	Timestamp string `json:"timestamp"`
	State     string `json:"state,omitempty"`
	Payload   string `json:"payload,omitempty"` // comms for device
}

func NewServer(eventsChan chan events.Event) *Server {
	s := &Server{
		clients:     make(map[chan StateUpdate]bool),
		broadcaster: make(chan StateUpdate, 100),
		eventsChan:  eventsChan,
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

func (s *Server) GetBroadcastChannel() chan<- StateUpdate {
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

	// Uplink endpoint
	mux.HandleFunc("/uplink", s.handleUplink)

	log.Printf("Web server starting on %s", addr)
	return http.ListenAndServe(addr, mux)
}

func (s *Server) handleUplink(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var payload struct {
		Device  string `json:"device"`
		Message string `json:"message"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	log.Printf("Received uplink from device %s: %s", payload.Device, payload.Message)

	switch payload.Message {
	case "StartConfigAck":
		go func() {
			s.eventsChan <- events.DeviceAck{}
		}()
	case "EndConfigAck":
		go func() {
			s.eventsChan <- events.EndConfigAck{}
		}()
	case "SetSleepPeriodAck":
		go func() {
			s.eventsChan <- events.DeviceAck{AckCode: "SLEEP_PERIOD"}
		}()
	case "ValueUnlockAck":
		go func() {
			s.eventsChan <- events.DeviceAck{
				AckCode: "VALUE_UNLOCK",
			}
		}()
	case "ValueSetAck":
		go func() {
			s.eventsChan <- events.DeviceAck{
				AckCode: "SET_PROTECTED_VALUE",
			}
		}()
	case "ValueLockAck":
		go func() {
			s.eventsChan <- events.DeviceAck{
				AckCode: "VALUE_LOCK",
			}
		}()
	}
}

func (s *Server) handleSSE(w http.ResponseWriter, r *http.Request) {
	// Set headers for SSE
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	// Create client channel
	clientChan := make(chan StateUpdate, 10)

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

func (s *Server) BroadcastUpdate(update StateUpdate) {
	s.broadcaster <- update
}
