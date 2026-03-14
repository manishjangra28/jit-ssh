package cloud

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
)

// AzureCredentials represents the sensitive client secrets needed to authenticate with Entra ID.
type AzureCredentials struct {
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
}

// AzureMetadata represents non-sensitive configuration for the Azure integration.
type AzureMetadata struct {
	TenantID string `json:"tenant_id"`
}

// AzureAdapter implements the Provider interface for Microsoft Azure (Entra ID).
// It uses standard REST calls to Microsoft Graph API authenticated via azidentity.
type AzureAdapter struct {
	credential *azidentity.ClientSecretCredential
	tenantID   string
	httpClient *http.Client
}

// NewAzureProvider initializes a new Azure cloud provider instance.
func NewAzureProvider(credentialsJSON string, metadataJSON string) (Provider, error) {
	var creds AzureCredentials
	if err := json.Unmarshal([]byte(credentialsJSON), &creds); err != nil {
		return nil, fmt.Errorf("failed to parse azure credentials: %w", err)
	}

	var meta AzureMetadata
	if err := json.Unmarshal([]byte(metadataJSON), &meta); err != nil {
		return nil, fmt.Errorf("failed to parse azure metadata: %w", err)
	}

	if creds.ClientID == "" || creds.ClientSecret == "" || meta.TenantID == "" {
		return nil, errors.New("missing required azure configuration fields (client_id, client_secret, tenant_id)")
	}

	// Create an Azure Identity Client Secret Credential
	cred, err := azidentity.NewClientSecretCredential(meta.TenantID, creds.ClientID, creds.ClientSecret, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize azure credential: %w", err)
	}

	return &AzureAdapter{
		credential: cred,
		tenantID:   meta.TenantID,
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}, nil
}

// getGraphToken retrieves a fresh OAuth2 token scoped for Microsoft Graph API.
func (a *AzureAdapter) getGraphToken(ctx context.Context) (string, error) {
	opts := policy.TokenRequestOptions{
		Scopes: []string{"https://graph.microsoft.com/.default"},
	}
	token, err := a.credential.GetToken(ctx, opts)
	if err != nil {
		return "", fmt.Errorf("failed to acquire azure token: %w", err)
	}
	return token.Token, nil
}

// ResolveUser queries Microsoft Graph to find a user by their User Principal Name (UPN) or email.
func (a *AzureAdapter) ResolveUser(ctx context.Context, email string) (string, error) {
	token, err := a.getGraphToken(ctx)
	if err != nil {
		return "", err
	}

	// Graph API endpoint for user lookup
	url := fmt.Sprintf("https://graph.microsoft.com/v1.0/users/%s", email)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/json")

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("graph api request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return "", fmt.Errorf("user %s not found in Entra ID", email)
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("failed to resolve user (status %d): %s", resp.StatusCode, string(body))
	}

	var result struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to decode graph response: %w", err)
	}

	if result.ID == "" {
		return "", errors.New("graph response did not contain an object ID")
	}

	return result.ID, nil
}

// GrantAccess adds the user to the specified Azure AD (Entra ID) Group.
func (a *AzureAdapter) GrantAccess(ctx context.Context, req AccessRequest) (*AccessResult, error) {
	userID, err := a.ResolveUser(ctx, req.UserEmail)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve user before granting access: %w", err)
	}

	token, err := a.getGraphToken(ctx)
	if err != nil {
		return nil, err
	}

	url := fmt.Sprintf("https://graph.microsoft.com/v1.0/groups/%s/members/$ref", req.TargetGroupID)

	// MS Graph expects the OData reference for the user object
	payload := map[string]string{
		"@odata.id": fmt.Sprintf("https://graph.microsoft.com/v1.0/directoryObjects/%s", userID),
	}
	bodyData, _ := json.Marshal(payload)

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(bodyData))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Authorization", "Bearer "+token)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := a.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("graph api add member failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNoContent {
		return nil, nil // Success
	}

	if resp.StatusCode == http.StatusBadRequest {
		// Might already be a member, let's check the error body
		bodyBytes, _ := io.ReadAll(resp.Body)
		if bytes.Contains(bodyBytes, []byte("One or more added object references already exist")) {
			return nil, nil // Idempotent success
		}
		return nil, fmt.Errorf("failed to add group member (status %d): %s", resp.StatusCode, string(bodyBytes))
	}

	body, _ := io.ReadAll(resp.Body)
	return nil, fmt.Errorf("failed to grant access (status %d): %s", resp.StatusCode, string(body))
}

// RevokeAccess removes the user from the specified Azure AD (Entra ID) Group.
func (a *AzureAdapter) RevokeAccess(ctx context.Context, req AccessRequest) error {
	userID, err := a.ResolveUser(ctx, req.UserEmail)
	if err != nil {
		return fmt.Errorf("failed to resolve user before revoking access: %w", err)
	}

	token, err := a.getGraphToken(ctx)
	if err != nil {
		return err
	}

	url := fmt.Sprintf("https://graph.microsoft.com/v1.0/groups/%s/members/%s/$ref", req.TargetGroupID, userID)

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodDelete, url, nil)
	if err != nil {
		return err
	}
	httpReq.Header.Set("Authorization", "Bearer "+token)

	resp, err := a.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("graph api remove member failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNoContent || resp.StatusCode == http.StatusNotFound {
		return nil // Success or user already removed (idempotent)
	}

	body, _ := io.ReadAll(resp.Body)
	return fmt.Errorf("failed to revoke access (status %d): %s", resp.StatusCode, string(body))
}

// TestConnection attempts a lightweight Graph API call to ensure credentials are valid.
func (a *AzureAdapter) TestConnection(ctx context.Context) error {
	token, err := a.getGraphToken(ctx)
	if err != nil {
		return fmt.Errorf("failed to authenticate with Azure: %w", err)
	}

	// Just request the tenant details or a single user to verify Graph API access
	url := "https://graph.microsoft.com/v1.0/users?$top=1&$select=id"

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("test connection request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("azure test connection failed (status %d): %s", resp.StatusCode, string(body))
	}

	return nil
}

func (a *AzureAdapter) ListGroups(ctx context.Context) ([]CloudGroup, error) {
	// Stub: Returning empty list for Azure until fully integrated
	return []CloudGroup{}, nil
}
