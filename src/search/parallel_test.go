package search

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// Synthetic MCP peer: exercise the supported SDK and actual outgoing headers.
func TestParallelMCP(t *testing.T) {
	for _, failure := range []string{"", "rpc", "tool", "http"} {
		t.Run("failure="+failure, func(t *testing.T) {
			var methods []string
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.UserAgent() != parallelUserAgent {
					t.Errorf("User-Agent = %q", r.UserAgent())
				}
				if r.Header.Get("Authorization") != "" {
					t.Error("anonymous search sent credentials")
				}
				if r.Method == "DELETE" {
					methods = append(methods, "DELETE")
					w.WriteHeader(200)
					return
				}
				var request struct {
					ID     any    `json:"id"`
					Method string `json:"method"`
					Params struct {
						Arguments map[string]any `json:"arguments"`
					} `json:"params"`
				}
				if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
					t.Error(err)
					return
				}
				methods = append(methods, request.Method)
				w.Header().Set("Content-Type", "Application/JSON")
				reply := map[string]any{"jsonrpc": "2.0", "id": request.ID}
				switch request.Method {
				case "initialize":
					w.Header().Set("Mcp-Session-Id", "synthetic-session")
					reply["result"] = map[string]any{"protocolVersion": "2025-03-26", "capabilities": map[string]any{}, "serverInfo": map[string]any{"name": "synthetic", "version": "1"}}
				case "notifications/initialized":
					w.WriteHeader(202)
					return
				case "tools/list":
					reply["result"] = map[string]any{"tools": []any{map[string]any{"name": "web_search", "inputSchema": map[string]any{"type": "object"}}}}
				case "tools/call":
					args := request.Params.Arguments
					queries, _ := args["search_queries"].([]any)
					if len(queries) != 1 || queries[0] != "site:example.org Go release notes" || len(args) != 2 {
						t.Errorf("arguments = %#v", args)
					}
					if failure == "http" {
						w.WriteHeader(429)
						fmt.Fprint(w, "rate limited")
						return
					}
					if failure == "rpc" {
						reply["error"] = map[string]any{"code": -32603, "message": "synthetic failure"}
					} else if failure == "tool" {
						reply["result"] = map[string]any{"isError": true, "content": []any{map[string]any{"type": "text", "text": "synthetic failure"}}}
					} else {
						// Flush a complete result while holding the connection open. Unicode
						// separators are legal inside JSON and must survive SSE framing.
						w.Header().Set("Content-Type", "Text/Event-Stream")
						text := `{"results":[{"title":"Go","url":"https://example.org/go","excerpts":["alpha\u0085beta\u2028gamma\u2029delta"]},{"url":"https://example.org/extra","excerpts":["extra"]}]}`
						reply["result"] = map[string]any{"content": []any{map[string]any{"type": "text", "text": text}}}
						data, _ := json.Marshal(reply)
						fmt.Fprintf(w, "event: message\r\ndata: {\"jsonrpc\":\"2.0\",\"method\":\"notifications/message\"}\r\n\r\nevent: message\r\ndata: %s\r\n\r\n", data)
						w.(http.Flusher).Flush()
						select {
						case <-r.Context().Done():
						case <-time.After(5 * time.Second):
							t.Error("client did not close completed SSE response")
						}
						return
					}
				default:
					t.Errorf("unexpected method %s", request.Method)
				}
				_ = json.NewEncoder(w).Encode(reply)
			}))
			defer server.Close()
			hc := server.Client()
			hc.Transport = parallelTransport{base: hc.Transport}
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			result, err := performParallelSearch(ctx, SearchParams{Query: "Go release notes", SiteFilter: "example.org", NumResults: 1}, false, server.URL, hc)
			if failure != "" {
				if err == nil || (!strings.Contains(err.Error(), "synthetic failure") && !strings.Contains(err.Error(), "rate limited")) {
					t.Fatalf("expected provider error, got %v", err)
				}
			} else if err != nil || !strings.Contains(result, "alpha\u0085beta\u2028gamma\u2029delta") || strings.Contains(result, "/extra") {
				t.Fatalf("result=%q, error=%v", result, err)
			}
			if got := strings.Join(methods, ","); got != "initialize,notifications/initialized,tools/list,tools/call,DELETE" {
				t.Fatalf("requests: %s", got)
			}
		})
	}
}

func TestParallelResults(t *testing.T) {
	if _, err := formatParallelResults([]byte(`not json`), 5); err == nil {
		t.Fatal("accepted malformed results")
	}
	out, err := formatParallelResults([]byte(`{"results":[],"warnings":[{"message":"limited"}]}`), 0)
	if err != nil || !strings.Contains(out, "No search results") || !strings.Contains(out, "limited") {
		t.Fatalf("%q %v", out, err)
	}
}

// A selected credentialed provider must still fail on missing credentials,
// rather than silently sending the query to the new anonymous provider.
func TestCredentialedSearchStillRequiresKeys(t *testing.T) {
	t.Setenv("SOFYA_API_KEY", "")
	t.Setenv("TGPT_GOOGLE_API_KEY", "")
	t.Setenv("TGPT_GOOGLE_SEARCH_ENGINE_ID", "")
	params := SearchParams{Query: "q", NumResults: 1}
	if _, err := PerformSofyaSearch(params, false); err == nil || !strings.Contains(err.Error(), "SOFYA_API_KEY") {
		t.Fatalf("Sofya missing-key error: %v", err)
	}
	if _, err := PerformSearchWithParams(params, false); err == nil || !strings.Contains(err.Error(), "TGPT_GOOGLE_API_KEY") {
		t.Fatalf("Google missing-key error: %v", err)
	}
}

// Synthetic HTTP peer, real TCP connections: cleanup must reach the wrapped
// transport even when it has no idle timeout, as in the project HTTP client.
func TestParallelTransportCleanup(t *testing.T) {
	idle := make(chan net.Conn, 4)
	closed := make(chan net.Conn, 4)
	started := make(chan struct{})
	release := make(chan struct{})
	server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/active" {
			close(started)
			select {
			case <-release:
			case <-r.Context().Done():
				return
			}
		}
		fmt.Fprint(w, "ok")
	}))
	server.Config.ConnState = func(conn net.Conn, state http.ConnState) {
		switch state {
		case http.StateIdle:
			idle <- conn
		case http.StateClosed:
			closed <- conn
		}
	}
	server.Start()
	defer server.Close()
	base := &http.Transport{}         // IdleConnTimeout deliberately unset.
	defer base.CloseIdleConnections() // Also safely release sockets if an assertion fails.
	hc := &http.Client{Transport: parallelTransport{base: base}, Timeout: 2 * time.Second}
	wait := func(events <-chan net.Conn, state string) net.Conn {
		t.Helper()
		select {
		case conn := <-events:
			return conn
		case <-time.After(time.Second):
			t.Fatalf("connection did not become %s", state)
			return nil
		}
	}
	request := func(path string) error {
		resp, err := hc.Get(server.URL + path)
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		body, err := io.ReadAll(resp.Body)
		if err == nil && string(body) != "ok" {
			return fmt.Errorf("body = %q", body)
		}
		return err
	}
	for i := 0; i < 3; i++ {
		if err := request("/"); err != nil {
			t.Fatal(err)
		}
		conn := wait(idle, "idle")
		hc.CloseIdleConnections()
		hc.CloseIdleConnections() // Repeated cleanup is safe and the client stays usable.
		if wait(closed, "closed") != conn {
			t.Fatal("closed the wrong connection")
		}
		t.Logf("call %d: observed idle socket close after client cleanup", i+1)
	}
	// Closing idle connections must not interrupt an in-flight request.
	done := make(chan error, 1)
	go func() { done <- request("/active") }()
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("active request did not start")
	}
	hc.CloseIdleConnections()
	close(release)
	if err := <-done; err != nil {
		t.Fatalf("cleanup interrupted active request: %v", err)
	}
	conn := wait(idle, "idle")
	hc.CloseIdleConnections()
	if wait(closed, "closed") != conn {
		t.Fatal("active request's idle connection was not closed")
	}
	t.Log("active request survived cleanup and its idle socket then closed")
	// RoundTrippers without optional cleanup support must remain safe to wrap.
	(&http.Client{Transport: parallelTransport{base: http.NewFileTransport(http.Dir(t.TempDir()))}}).CloseIdleConnections()
}
