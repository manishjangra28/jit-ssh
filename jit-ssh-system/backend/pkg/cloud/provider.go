package cloud

import "context"

// AccessRequest represents the data needed by a cloud provider
// to grant or revoke a user's access to a specific resource group.
type AccessRequest struct {
	TargetGroupID     string
	TargetGroupName   string
	UserEmail         string
	GeneratePassword  bool
	GenerateAccessKey bool
}

// AccessResult contains temporary credentials if generated during provisioning.
type AccessResult struct {
	ConsoleURL      string `json:"console_url,omitempty"`
	Username        string `json:"username,omitempty"`
	Password        string `json:"password,omitempty"`
	AccessKeyID     string `json:"access_key_id,omitempty"`
	SecretAccessKey string `json:"secret_access_key,omitempty"`
}

// CloudGroup represents a group fetched directly from the cloud provider
type CloudGroup struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// Provider defines the standard interface that all cloud integrations
// (AWS, GCP, Azure) must implement to support Just-In-Time access.
type Provider interface {
	// ResolveUser checks if a user exists in the cloud directory (e.g., AWS Identity Store).
	// If the user does not exist, implementations may optionally create the user.
	// Returns the cloud-specific User ID or an error.
	ResolveUser(ctx context.Context, email string) (string, error)

	// GrantAccess adds the specified user to the target group or role defined in the request.
	// It may return an AccessResult containing temporary passwords or access keys if requested.
	GrantAccess(ctx context.Context, req AccessRequest) (*AccessResult, error)

	// RevokeAccess removes the specified user from the target group or role.
	RevokeAccess(ctx context.Context, req AccessRequest) error

	// TestConnection verifies that the provided credentials and configuration are valid.
	TestConnection(ctx context.Context) error

	// ListGroups fetches the available groups from the cloud provider so users can select them.
	ListGroups(ctx context.Context) ([]CloudGroup, error)
}

// Config represents the decrypted configuration needed to initialize a Provider.
type Config struct {
	IntegrationID string
	ProviderType  string // aws, gcp, azure
	Metadata      string // JSON string containing non-secret config (e.g., region, tenant_id)
	Credentials   []byte // The decrypted raw credentials (e.g., AWS Secret Key, GCP JSON)
}
