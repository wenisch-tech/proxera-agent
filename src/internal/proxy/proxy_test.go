package proxy

import (
	"context"
	"encoding/base64"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/wenisch-tech/proxera-agent/internal/protocol"
)

func TestHandleForwardsRequestAndEncodesResponse(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/local/path", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Connection") != "" {
			t.Fatalf("hop-by-hop header should not be forwarded")
		}
		body, _ := io.ReadAll(r.Body)
		if string(body) != "hello" {
			t.Fatalf("unexpected request body: %q", string(body))
		}
		w.Header().Set("Content-Type", "text/plain")
		_, _ = w.Write([]byte("ok"))
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	hostPort := strings.TrimPrefix(server.URL, "http://")
	parts := strings.Split(hostPort, ":")
	if len(parts) != 2 {
		t.Fatalf("unexpected hostPort: %s", hostPort)
	}
	port, err := strconv.Atoi(parts[1])
	if err != nil {
		t.Fatalf("invalid port: %v", err)
	}

	client := New(5 * time.Second)
	resp, err := client.Handle(context.Background(), protocol.RequestPayload{
		Method:      http.MethodPost,
		Path:        "/api/local/path",
		StripPrefix: "/api",
		LocalHost:   parts[0],
		LocalPort:   port,
		Headers: map[string][]string{
			"Connection": {"keep-alive"},
		},
		Body: base64.StdEncoding.EncodeToString([]byte("hello")),
	})
	if err != nil {
		t.Fatalf("unexpected handle error: %v", err)
	}
	if resp.Status != 200 {
		t.Fatalf("unexpected status: %d", resp.Status)
	}
	decoded, err := base64.StdEncoding.DecodeString(resp.Body)
	if err != nil {
		t.Fatalf("invalid encoded response body: %v", err)
	}
	if string(decoded) != "ok" {
		t.Fatalf("unexpected response body: %q", string(decoded))
	}
}

func TestHandlePreservesPublicHostWhenEnabled(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/bucket", func(w http.ResponseWriter, r *http.Request) {
		if r.Host != "s3.intranet.wenisch.tech" {
			t.Fatalf("unexpected request host: %q", r.Host)
		}
		w.WriteHeader(http.StatusNoContent)
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	hostPort := strings.TrimPrefix(server.URL, "http://")
	parts := strings.Split(hostPort, ":")
	if len(parts) != 2 {
		t.Fatalf("unexpected hostPort: %s", hostPort)
	}
	port, err := strconv.Atoi(parts[1])
	if err != nil {
		t.Fatalf("invalid port: %v", err)
	}

	client := New(5 * time.Second)
	resp, err := client.Handle(context.Background(), protocol.RequestPayload{
		Method:             http.MethodGet,
		Path:               "/bucket",
		LocalHost:          parts[0],
		LocalPort:          port,
		PreserveHostHeader: true,
		Headers: map[string][]string{
			"host": {"s3.intranet.wenisch.tech"},
		},
	})
	if err != nil {
		t.Fatalf("unexpected handle error: %v", err)
	}
	if resp.Status != http.StatusNoContent {
		t.Fatalf("unexpected status: %d", resp.Status)
	}
}

func TestHandleKeepsLocalHostWhenPreservationDisabled(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/bucket", func(w http.ResponseWriter, r *http.Request) {
		if r.Host == "s3.intranet.wenisch.tech" {
			t.Fatalf("request host should not be preserved when disabled")
		}
		w.WriteHeader(http.StatusNoContent)
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	hostPort := strings.TrimPrefix(server.URL, "http://")
	parts := strings.Split(hostPort, ":")
	if len(parts) != 2 {
		t.Fatalf("unexpected hostPort: %s", hostPort)
	}
	port, err := strconv.Atoi(parts[1])
	if err != nil {
		t.Fatalf("invalid port: %v", err)
	}

	client := New(5 * time.Second)
	resp, err := client.Handle(context.Background(), protocol.RequestPayload{
		Method:    http.MethodGet,
		Path:      "/bucket",
		LocalHost: parts[0],
		LocalPort: port,
		Headers: map[string][]string{
			"host": {"s3.intranet.wenisch.tech"},
		},
	})
	if err != nil {
		t.Fatalf("unexpected handle error: %v", err)
	}
	if resp.Status != http.StatusNoContent {
		t.Fatalf("unexpected status: %d", resp.Status)
	}
}
