package ingest

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/mikey/faultline/internal/event"
)

func TestServeIngest(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var got event.Event
	done := make(chan struct{})
	errCh := make(chan error, 1)
	ready := make(chan string, 1)
	go func() {
		_, err := Serve(ctx, "127.0.0.1:0", Options{
			Name: "browser",
			Emit: func(ev event.Event) {
				got = ev
				close(done)
			},
			Ready: func(addr string) {
				ready <- addr
			},
		})
		errCh <- err
	}()

	var bound string
	select {
	case bound = <-ready:
	case <-time.After(2 * time.Second):
		t.Fatal("server did not start")
	}

	body, _ := json.Marshal(Payload{Type: "TypeError", Message: "boom", File: "/src/a.js", Line: 3})
	req, err := http.NewRequest(http.MethodPost, "http://"+bound+"/ingest", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", "http://localhost:5173")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusAccepted {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("status %d body %s", resp.StatusCode, b)
	}
	if resp.Header.Get("Access-Control-Allow-Origin") != "*" {
		t.Fatal("missing CORS")
	}

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("event not emitted")
	}
	if got.Type != "TypeError" || got.Message != "boom" {
		t.Fatalf("event = %+v", got)
	}

	optReq, _ := http.NewRequest(http.MethodOptions, "http://"+bound+"/ingest", nil)
	optResp, err := http.DefaultClient.Do(optReq)
	if err != nil {
		t.Fatal(err)
	}
	optResp.Body.Close()
	if optResp.StatusCode != http.StatusNoContent {
		t.Fatalf("options status %d", optResp.StatusCode)
	}

	cancel()
	select {
	case <-errCh:
	case <-time.After(2 * time.Second):
		t.Fatal("server did not exit")
	}
}

func TestServeDisabled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	addr, err := Serve(ctx, Disabled, Options{})
	if err != nil || addr != "" {
		t.Fatalf("addr=%q err=%v", addr, err)
	}
}
