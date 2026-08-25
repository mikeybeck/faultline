package ingest

import (
	"context"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/mikey/faultline/internal/event"
	"github.com/mikey/faultline/internal/source"
)

const maxBody = 256 << 10

// Options configure the browser ingest HTTP server.
type Options struct {
	Name       string
	ProjectDir string
	Emit       func(event.Event)
	OnStatus   func(source.Status)
	Ready      func(addr string)
}

// Serve listens on addr until ctx is cancelled. Addr Disabled is a no-op.
// The bound address is returned (useful when addr ends in :0).
func Serve(ctx context.Context, addr string, opt Options) (string, error) {
	addr = NormalizeAddr(addr)
	if addr == "" || strings.EqualFold(addr, Disabled) {
		return "", nil
	}
	if opt.Name == "" {
		opt.Name = DefaultName
	}

	ln, err := net.Listen("tcp", addr)
	if err != nil {
		report(opt, source.Status{
			Name:    opt.Name,
			Type:    "browser",
			Path:    addr,
			State:   source.StateError,
			Message: err.Error(),
			Updated: time.Now(),
		})
		return "", err
	}
	bound := ln.Addr().String()
	if opt.Ready != nil {
		opt.Ready(bound)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/ingest", func(w http.ResponseWriter, r *http.Request) {
		cors(w)
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		r.Body = http.MaxBytesReader(w, r.Body, maxBody)
		defer r.Body.Close()
		var p Payload
		if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
			http.Error(w, "invalid json", http.StatusBadRequest)
			return
		}
		if opt.Emit != nil {
			opt.Emit(ToEvent(opt.Name, opt.ProjectDir, p))
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		_, _ = io.WriteString(w, `{"ok":true}`)
	})
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		cors(w)
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"ok":true}`)
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		cors(w)
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = io.WriteString(w, "Faultline browser ingest\nPOST /ingest\n")
	})

	srv := &http.Server{
		Handler:           mux,
		ReadHeaderTimeout: 2 * time.Second,
		BaseContext:       func(net.Listener) context.Context { return ctx },
	}

	report(opt, source.Status{
		Name:    opt.Name,
		Type:    "browser",
		Path:    bound,
		State:   source.StateOK,
		Message: "extension → " + bound,
		Updated: time.Now(),
	})

	go func() {
		<-ctx.Done()
		shCtx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
		defer cancel()
		_ = srv.Shutdown(shCtx)
	}()

	err = srv.Serve(ln)
	if err != nil && err != http.ErrServerClosed {
		report(opt, source.Status{
			Name:    opt.Name,
			Type:    "browser",
			Path:    bound,
			State:   source.StateError,
			Message: err.Error(),
			Updated: time.Now(),
		})
		return bound, err
	}
	return bound, nil
}

func cors(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	w.Header().Set("Access-Control-Max-Age", "600")
}

func report(opt Options, s source.Status) {
	if opt.OnStatus != nil {
		opt.OnStatus(s)
	}
}
