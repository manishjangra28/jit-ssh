package cloud

import (
	"errors"
	"fmt"

	"github.com/manishjangra/jit-ssh-system/backend/models"
	"github.com/manishjangra/jit-ssh-system/backend/pkg/crypto"
)

// NewProvider is a factory function that instantiates the correct cloud provider client
func NewProvider(integration *models.CloudIntegration) (Provider, error) {
	if integration == nil {
		return nil, errors.New("integration cannot be nil")
	}

	// 1. Get the Master Key to decrypt credentials
	key, err := crypto.GetMasterKey()
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve master encryption key: %w", err)
	}

	// 2. Decrypt the credentials
	decryptedCreds, err := crypto.DecryptString(string(integration.EncryptedCredentials), key)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt cloud credentials: %w", err)
	}

	// 3. Route to the correct provider adapter
	switch integration.Provider {
	case models.ProviderAWS:
		return NewAWSProvider(decryptedCreds, integration.Metadata)
	case models.CloudProviderType("aws-iam"):
		return NewAWSIAMProvider(decryptedCreds, integration.Metadata)
	case models.ProviderGCP:
		return NewGCPProvider(decryptedCreds, integration.Metadata)
	case models.ProviderAzure:
		return NewAzureProvider(decryptedCreds, integration.Metadata)
	default:
		return nil, fmt.Errorf("unsupported cloud provider: %s", integration.Provider)
	}
}
