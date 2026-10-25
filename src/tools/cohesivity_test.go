package tools

import (
	"context"
	"encoding/json"
	stdhttp "net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// startCohesivityServer points the Cohesivity tools at a local MCP server and
// records the arguments of the last tools/call it received.
func startCohesivityServer(t *testing.T, reply string) *map[string]any {
	var got map[string]any
	server := httptest.NewServer(stdhttp.HandlerFunc(func(w stdhttp.ResponseWriter, r *stdhttp.Request) {
		var req struct {
			Method string `json:"method"`
			Params struct {
				Name      string         `json:"name"`
				Arguments map[string]any `json:"arguments"`
			} `json:"params"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		got = req.Params.Arguments
		got["_tool"] = req.Method + " " + req.Params.Name
		got["_ua"] = r.UserAgent()
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte(reply))
	}))
	t.Cleanup(server.Close)
	orig := cohesivityMCPEndpoint
	cohesivityMCPEndpoint = server.URL
	t.Cleanup(func() { cohesivityMCPEndpoint = orig })
	return &got
}

func TestCohesivityCreateTenant(t *testing.T) {
	got := startCohesivityServer(t, "event: message\ndata: {\"jsonrpc\":\"2.0\",\"method\":\"notifications/progress\",\"params\":{}}\n\n"+
		"event: message\ndata: {\"jsonrpc\":\"2.0\",\"id\":1,\"result\":{\"content\":[{\"type\":\"text\",\"text\":\"tenant_id: test-tenant\"}]}}\n\n")

	r := NewRegistry()
	r.RegisterBuiltinTools("cohesivity_create_tenant")
	ctx := context.WithValue(context.Background(), AutoExecKey, true)
	res, err := r.Execute(ctx, "cohesivity_create_tenant", `{}`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(res, "tenant_id: test-tenant") || !strings.Contains(res, ".cohesivity") {
		t.Fatalf("expected tenant result and save instructions, got %q", res)
	}
	if (*got)["_tool"] != "tools/call create_tenant" || (*got)["confirmed"] != true || (*got)["attribution"] != "gh-tgpt" || (*got)["_ua"] != "tgpt-cli" {
		t.Fatalf("unexpected create_tenant call: %#v", *got)
	}
}

func TestCohesivityProvision(t *testing.T) {
	got := startCohesivityServer(t, `{"jsonrpc":"2.0","id":1,"result":{"content":[{"type":"text","text":"postgres active"}]}}`)

	r := NewRegistry()
	r.RegisterBuiltinTools("cohesivity_provision")
	ctx := context.WithValue(context.Background(), ConfirmedKey, true)
	res, err := r.Execute(ctx, "cohesivity_provision", `{"tenant_id": "t1", "coh_management_key": "coh_man_x", "resources": ["postgres"]}`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res != "postgres active" {
		t.Fatalf("expected %q, got %q", "postgres active", res)
	}
	resources, _ := (*got)["resources"].([]any)
	if (*got)["_tool"] != "tools/call provision_resource" || (*got)["tenant_id"] != "t1" || (*got)["confirmed"] != true || len(resources) != 1 || resources[0] != "postgres" {
		t.Fatalf("unexpected provision_resource call: %#v", *got)
	}
}

func TestCohesivityProvisionDeclined(t *testing.T) {
	got := startCohesivityServer(t, `{"jsonrpc":"2.0","id":1,"result":{"content":[{"type":"text","text":"postgres active"}]}}`)
	var prompt string
	orig := confirmAction
	confirmAction = func(p string) (bool, error) { prompt = p; return false, nil }
	t.Cleanup(func() { confirmAction = orig })

	args := `{"tenant_id": "t1", "coh_management_key": "coh_man_x", "resources": "postgres"}`
	proceed, msg, err := PreConfirm(context.Background(), "cohesivity_provision", args)
	if err != nil || proceed || msg != cohesivityProvisionCancelled {
		t.Fatalf("expected declined provision, got %v, %q, %v", proceed, msg, err)
	}
	if !strings.Contains(prompt, "`postgres`") || !strings.Contains(prompt, "`t1`") {
		t.Fatalf("prompt should name the resources and tenant, got %q", prompt)
	}

	r := NewRegistry()
	r.RegisterBuiltinTools("cohesivity_provision")
	res, err := r.Execute(context.Background(), "cohesivity_provision", args)
	if err != nil || res != cohesivityProvisionCancelled {
		t.Fatalf("expected cancelled result, got %q, %v", res, err)
	}
	if *got != nil {
		t.Fatalf("declined provision should not reach the server, got %#v", *got)
	}
}

func TestCohesivityStatusErrors(t *testing.T) {
	startCohesivityServer(t, `{"jsonrpc":"2.0","id":1,"result":{"isError":true,"content":[{"type":"text","text":"tenant not found"}]}}`)

	r := NewRegistry()
	r.RegisterBuiltinTools("cohesivity_status")
	if _, err := r.Execute(context.Background(), "cohesivity_status", `{"tenant_id": "t1"}`); err == nil {
		t.Fatal("expected error when coh_management_key is missing")
	}
	_, err := r.Execute(context.Background(), "cohesivity_status", `{"tenant_id": "t1", "coh_management_key": "coh_man_x"}`)
	if err == nil || !strings.Contains(err.Error(), "tenant not found") {
		t.Fatalf("expected tool error to be returned, got %v", err)
	}
}

func TestCohesivityToolsOptIn(t *testing.T) {
	cases := []struct {
		selected []string
		want     bool
	}{
		{nil, false},
		{[]string{"all"}, false},
		{[]string{"all", "cohesivity"}, true},
		{[]string{"cohesivity_status"}, true},
	}
	for _, c := range cases {
		r := NewRegistry()
		r.RegisterBuiltinTools(c.selected...)
		if r.Has("cohesivity_status") != c.want || r.Has("read_file") != (c.selected == nil || c.selected[0] == "all") {
			t.Errorf("unexpected tools registered for %v", c.selected)
		}
	}
	if list, ok := ParseToolList("all,cohesivity"); !ok || len(list) != 2 {
		t.Fatalf("expected all,cohesivity to parse, got %v, %v", list, ok)
	}
}

func TestCohesivityResourceList(t *testing.T) {
	cases := []struct {
		in   any
		want []any
	}{
		{[]any{"postgres", "object-storage"}, []any{"postgres", "object-storage"}},
		{"postgres", []any{"postgres"}},
		{"postgres, object-storage", []any{"postgres", "object-storage"}},
		{"", nil},
		{nil, nil},
		{42, nil},
	}
	for _, tc := range cases {
		got := cohesivityResourceList(tc.in)
		if len(got) != len(tc.want) {
			t.Fatalf("resources %v: got %v, want %v", tc.in, got, tc.want)
		}
		for i := range got {
			if got[i] != tc.want[i] {
				t.Fatalf("resources %v: got %v, want %v", tc.in, got, tc.want)
			}
		}
	}
}
