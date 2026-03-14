package cloud

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"google.golang.org/api/cloudidentity/v1"
	"google.golang.org/api/option"
)

type GCPMetadata struct {
	CustomerID string `json:"customer_id"`
}

// GCPProvider implements the Provider interface for Google Cloud Platform.
type GCPProvider struct {
	client     *cloudidentity.Service
	customerID string
}

// NewGCPProvider initializes a new Google Cloud provider client.
func NewGCPProvider(credentialsJSON string, metadataJSON string) (Provider, error) {
	if credentialsJSON == "" {
		return nil, errors.New("GCP credentials JSON cannot be empty")
	}

	var meta GCPMetadata
	if metadataJSON != "" {
		_ = json.Unmarshal([]byte(metadataJSON), &meta)
	}

	client, err := cloudidentity.NewService(context.Background(), option.WithCredentialsJSON([]byte(credentialsJSON)))
	if err != nil {
		return nil, fmt.Errorf("failed to initialize GCP client: %w", err)
	}

	return &GCPProvider{
		client:     client,
		customerID: meta.CustomerID,
	}, nil
}

func (p *GCPProvider) getGroupName(ctx context.Context, idOrEmail string) (string, error) {
	if strings.HasPrefix(idOrEmail, "groups/") {
		return idOrEmail, nil
	}

	resp, err := p.client.Groups.Lookup().GroupKeyId(idOrEmail).Context(ctx).Do()
	if err != nil {
		return "", fmt.Errorf("failed to lookup GCP group %s: %w", idOrEmail, err)
	}
	return resp.Name, nil
}

func (p *GCPProvider) ResolveUser(ctx context.Context, email string) (string, error) {
	// For GCP Cloud Identity, we can directly use the email as an EntityKey.
	return email, nil
}

func (p *GCPProvider) GrantAccess(ctx context.Context, req AccessRequest) (*AccessResult, error) {
	groupName, err := p.getGroupName(ctx, req.TargetGroupID)
	if err != nil {
		return nil, err
	}

	membership := &cloudidentity.Membership{
		PreferredMemberKey: &cloudidentity.EntityKey{
			Id: req.UserEmail,
		},
		Roles: []*cloudidentity.MembershipRole{
			{
				Name: "MEMBER",
			},
		},
	}

	_, err = p.client.Groups.Memberships.Create(groupName, membership).Context(ctx).Do()
	if err != nil {
		if strings.Contains(err.Error(), "409") || strings.Contains(err.Error(), "already exists") {
			return &AccessResult{ConsoleURL: "https://console.cloud.google.com"}, nil
		}
		return nil, fmt.Errorf("failed to add user to GCP group %s: %w", req.TargetGroupID, err)
	}

	return &AccessResult{ConsoleURL: "https://console.cloud.google.com"}, nil
}

func (p *GCPProvider) RevokeAccess(ctx context.Context, req AccessRequest) error {
	groupName, err := p.getGroupName(ctx, req.TargetGroupID)
	if err != nil {
		if strings.Contains(err.Error(), "404") || strings.Contains(err.Error(), "Not Found") {
			return nil
		}
		return err
	}

	lookupResp, err := p.client.Groups.Memberships.Lookup(groupName).MemberKeyId(req.UserEmail).Context(ctx).Do()
	if err != nil {
		if strings.Contains(err.Error(), "404") || strings.Contains(err.Error(), "Not Found") {
			return nil
		}
		return fmt.Errorf("failed to lookup GCP membership for %s: %w", req.UserEmail, err)
	}

	_, err = p.client.Groups.Memberships.Delete(lookupResp.Name).Context(ctx).Do()
	if err != nil {
		return fmt.Errorf("failed to remove user from GCP group %s: %w", req.TargetGroupID, err)
	}

	return nil
}

func (p *GCPProvider) TestConnection(ctx context.Context) error {
	_, err := p.client.Groups.Lookup().GroupKeyId("test-connection-dummy@example.com").Context(ctx).Do()
	if err != nil {
		if strings.Contains(err.Error(), "401") || strings.Contains(err.Error(), "invalid_grant") {
			return fmt.Errorf("GCP authentication failed: %w", err)
		}
	}
	return nil
}

func (p *GCPProvider) ListGroups(ctx context.Context) ([]CloudGroup, error) {
	// Stub: Returning empty list for GCP until fully integrated with Customer ID
	return []CloudGroup{}, nil
}
