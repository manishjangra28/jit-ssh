package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

const (
	defaultBaseURL = "http://localhost:8080/api/v1"
)

type JITClient struct {
	BaseURL string
	UserID  string
}

func main() {
	baseURL := os.Getenv("JIT_API_URL")
	if baseURL == "" {
		baseURL = defaultBaseURL
	}

	userID := os.Getenv("JIT_USER_ID")
	if userID == "" {
		userID = "mcp-agent-default"
	}

	jitClient := &JITClient{
		BaseURL: baseURL,
		UserID:  userID,
	}

	s := server.NewMCPServer(
		"JIT Multi-Cloud Access Manager",
		"1.0.0",
	)

	s.AddTool(mcp.NewTool("list_ssh_servers",
		mcp.WithDescription("List all registered SSH servers available for access"),
	), jitClient.listSSHServers)

	s.AddTool(mcp.NewTool("request_ssh_access",
		mcp.WithDescription("Request temporary SSH access to a specific server"),
	), jitClient.requestSSHAccess)

	s.AddTool(mcp.NewTool("list_cloud_integrations",
		mcp.WithDescription("List available cloud environments (AWS, Azure, GCP)"),
	), jitClient.listCloudIntegrations)

	s.AddTool(mcp.NewTool("list_cloud_groups",
		mcp.WithDescription("List available IAM groups/roles for a specific cloud integration"),
	), jitClient.listCloudGroups)

	s.AddTool(mcp.NewTool("request_cloud_access",
		mcp.WithDescription("Request temporary membership in a cloud group"),
	), jitClient.requestCloudAccess)

	s.AddTool(mcp.NewTool("get_access_status",
		mcp.WithDescription("Check the status of a specific access request"),
	), jitClient.getAccessStatus)

	if err := server.ServeStdio(s); err != nil {
		log.Fatalf("MCP server error: %v", err)
	}
}

func (c *JITClient) listSSHServers(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	resp, err := http.Get(c.BaseURL + "/servers")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to contact JIT API: %v", err)), nil
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	return mcp.NewToolResultText(string(body)), nil
}

func (c *JITClient) requestSSHAccess(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	argData, _ := json.Marshal(request.Params.Arguments)
	var params struct {
		ServerID      string `json:"server_id"`
		DurationHours int    `json:"duration_hours"`
		Reason        string `json:"reason"`
		PubKey        string `json:"pub_key"`
	}
	if err := json.Unmarshal(argData, &params); err != nil {
		return nil, err
	}
	payload := map[string]interface{}{
		"user_id":   c.UserID,
		"server_id": params.ServerID,
		"duration":  fmt.Sprintf("%dh", params.DurationHours),
		"reason":    params.Reason,
		"pub_key":   params.PubKey,
	}
	data, _ := json.Marshal(payload)
	resp, err := http.Post(c.BaseURL+"/requests", "application/json", bytes.NewBuffer(data))
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("API error: %v", err)), nil
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	return mcp.NewToolResultText(fmt.Sprintf("Request submitted. Response: %s", string(body))), nil
}

func (c *JITClient) listCloudIntegrations(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	resp, err := http.Get(c.BaseURL + "/cloud-integrations")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("API error: %v", err)), nil
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	return mcp.NewToolResultText(string(body)), nil
}

func (c *JITClient) listCloudGroups(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	argData, _ := json.Marshal(request.Params.Arguments)
	var params struct {
		IntegrationID string `json:"integration_id"`
	}
	if err := json.Unmarshal(argData, &params); err != nil {
		return nil, err
	}
	url := fmt.Sprintf("%s/cloud-integrations/%s/groups", c.BaseURL, params.IntegrationID)
	resp, err := http.Get(url)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("API error: %v", err)), nil
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	return mcp.NewToolResultText(string(body)), nil
}

func (c *JITClient) requestCloudAccess(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	argData, _ := json.Marshal(request.Params.Arguments)
	var params struct {
		IntegrationID    string `json:"integration_id"`
		TargetGroupID    string `json:"target_group_id"`
		TargetGroupName  string `json:"target_group_name"`
		DurationHours    int    `json:"duration_hours"`
		Reason           string `json:"reason"`
		RequiresPassword bool   `json:"requires_password"`
		RequiresKeys     bool   `json:"requires_keys"`
	}
	if err := json.Unmarshal(argData, &params); err != nil {
		return nil, err
	}
	payload := map[string]interface{}{
		"user_id":           c.UserID,
		"integration_id":    params.IntegrationID,
		"target_group_id":   params.TargetGroupID,
		"target_group_name": params.TargetGroupName,
		"duration_hours":    params.DurationHours,
		"reason":            params.Reason,
		"requires_password": params.RequiresPassword,
		"requires_keys":     params.RequiresKeys,
	}
	data, _ := json.Marshal(payload)
	resp, err := http.Post(c.BaseURL+"/cloud-requests", "application/json", bytes.NewBuffer(data))
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("API error: %v", err)), nil
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	return mcp.NewToolResultText(fmt.Sprintf("Cloud access request submitted. Response: %s", string(body))), nil
}

func (c *JITClient) getAccessStatus(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	argData, _ := json.Marshal(request.Params.Arguments)
	var params struct {
		RequestID   string `json:"request_id"`
		RequestType string `json:"request_type"`
	}
	if err := json.Unmarshal(argData, &params); err != nil {
		return nil, err
	}
	endpoint := "/requests"
	if params.RequestType == "cloud" {
		endpoint = "/cloud-requests"
	}
	resp, err := http.Get(c.BaseURL + endpoint)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("API error: %v", err)), nil
	}
	defer resp.Body.Close()
	var allRequests []map[string]interface{}
	body, _ := io.ReadAll(resp.Body)
	if err := json.Unmarshal(body, &allRequests); err != nil {
		return mcp.NewToolResultError("Failed to parse API response"), nil
	}
	for _, r := range allRequests {
		if r["id"] == params.RequestID {
			reqJSON, _ := json.MarshalIndent(r, "", "  ")
			return mcp.NewToolResultText(string(reqJSON)), nil
		}
	}
	return mcp.NewToolResultError("Request ID not found"), nil
}
