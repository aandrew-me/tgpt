package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/aandrew-me/tgpt/v2/src/client"
	"github.com/aandrew-me/tgpt/v2/src/search"
	http "github.com/bogdanfinn/fhttp"
)

// cohesivityMCPEndpoint is the hosted Cohesivity MCP server. No API key is needed.
var cohesivityMCPEndpoint = "https://cohesivity.ai/mcp"

var cohesivityResources = []string{
	"railway-hosting", "cloudflare-workers", "postgres", "redis", "object-storage", "vector-database",
	"social-login", "inbox", "realtime", "ai-gateway", "openai-api", "exa-api", "deepgram-api",
	"steel-browser", "openweather-api", "google-geocoding-api",
}

const (
	cohesivityCreatePrompt       = "\nCreate a Cohesivity backend? It is free and expires in 72 hours unless claimed."
	cohesivityCreateCancelled    = "Backend creation cancelled by user."
	cohesivityProvisionCancelled = "Provisioning cancelled by user."
	cohesivityCreateNote         = "Save credentials_file.content from this result verbatim to .cohesivity in the project root " +
		"with write_file, then run `chmod 600 .cohesivity` and add .cohesivity to .gitignore. " +
		"Do not show the keys in chat. Tell the user what was created and that it expires in 72 hours unless they claim it."
)

// cohesivityResourceList accepts the resources argument as a JSON array or as a
// single string, optionally comma separated, since smaller models often send
// "postgres" or "postgres, object-storage" instead of ["postgres"].
func cohesivityResourceList(value any) []any {
	var out []any
	add := func(s string) {
		if s = strings.TrimSpace(s); s != "" {
			out = append(out, s)
		}
	}
	switch v := value.(type) {
	case []any:
		for _, item := range v {
			if s, ok := item.(string); ok {
				add(s)
			}
		}
	case string:
		for _, s := range strings.Split(v, ",") {
			add(s)
		}
	}
	return out
}

// cohesivityProvisionPrompt names the resources and tenant a provision call would change.
func cohesivityProvisionPrompt(args map[string]any) string {
	var names []string
	for _, r := range cohesivityResourceList(args["resources"]) {
		names = append(names, r.(string))
	}
	tenantID, _ := args["tenant_id"].(string)
	return fmt.Sprintf("\nAdd `%s` to Cohesivity backend `%s` ?", strings.Join(names, ", "), tenantID)
}

// cohesivityTenantParams returns the tenant parameters shared by cohesivity_provision and cohesivity_status.
func cohesivityTenantParams() map[string]any {
	return map[string]any{
		"tenant_id":          map[string]any{"type": "string", "description": "tenant_id from .cohesivity or cohesivity_create_tenant"},
		"coh_management_key": map[string]any{"type": "string", "description": "coh_management_key from .cohesivity or cohesivity_create_tenant"},
	}
}

// callCohesivityMCP calls a tool on the hosted Cohesivity MCP server and returns its text content.
func callCohesivityMCP(ctx context.Context, toolName string, arguments map[string]interface{}) (string, error) {
	payload := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "tools/call",
		"params": map[string]interface{}{
			"name":      toolName,
			"arguments": arguments,
		},
	}

	reqBody, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("failed to marshal JSON-RPC request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", cohesivityMCPEndpoint, bytes.NewBuffer(reqBody))
	if err != nil {
		return "", fmt.Errorf("failed to build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	req.Header.Set("User-Agent", "tgpt-cli")

	httpClient, err := client.NewClient(60)
	if err != nil {
		return "", fmt.Errorf("failed to create HTTP client: %w", err)
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("server error HTTP %d: %s", resp.StatusCode, string(bodyBytes))
	}

	// The reply is plain JSON or an event stream that may carry progress
	// notifications first, so use the first message that has a result or error.
	var mcpResp search.MCPResponse
	for _, line := range strings.Split(string(bodyBytes), "\n") {
		line = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(line), "data:"))
		if json.Unmarshal([]byte(line), &mcpResp) == nil && (mcpResp.Result != nil || mcpResp.Error != nil) {
			break
		}
	}

	if mcpResp.Error != nil {
		return "", fmt.Errorf("MCP RPC Error [%d]: %s", mcpResp.Error.Code, mcpResp.Error.Message)
	}
	if mcpResp.Result == nil || len(mcpResp.Result.Content) == 0 {
		return "", fmt.Errorf("empty response from Cohesivity MCP")
	}
	if mcpResp.Result.IsError {
		return "", fmt.Errorf("MCP Tool execution error: %s", mcpResp.Result.Content[0].Text)
	}

	var builder strings.Builder
	for _, item := range mcpResp.Result.Content {
		if item.Type == "text" {
			builder.WriteString(item.Text)
			builder.WriteString("\n\n")
		}
	}
	return strings.TrimSpace(builder.String()), nil
}
