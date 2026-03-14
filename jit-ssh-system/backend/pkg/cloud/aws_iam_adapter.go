package cloud

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go-v2/service/iam/types"
)

// AWSIAMMetadata represents the non-secret configuration for AWS IAM (Legacy) integration.
type AWSIAMMetadata struct {
	Region    string `json:"region"`
	AccountID string `json:"account_id"`
}

// AWSIAMProvider implements the Provider interface for traditional AWS IAM.
// It creates IAM Users, adds them to an IAM Group, and can optionally generate
// console passwords and access keys on the fly.
type AWSIAMProvider struct {
	client    *iam.Client
	accountID string
}

func (p *AWSIAMProvider) ListGroups(ctx context.Context) ([]CloudGroup, error) {
	var groups []CloudGroup

	input := &iam.ListGroupsInput{}
	for {
		output, err := p.client.ListGroups(ctx, input)
		if err != nil {
			return nil, fmt.Errorf("failed to list IAM groups: %w", err)
		}

		for _, g := range output.Groups {
			if g.GroupName != nil {
				// For legacy IAM, the ID and Name we care about are both just the GroupName
				name := aws.ToString(g.GroupName)
				groups = append(groups, CloudGroup{
					ID:   name,
					Name: name,
				})
			}
		}

		if !output.IsTruncated {
			break
		}
		input.Marker = output.Marker
	}

	return groups, nil
}

// NewAWSIAMProvider creates and initializes a new AWS IAM Provider.
func NewAWSIAMProvider(credentialsJSON string, metadataJSON string) (Provider, error) {
	var creds AWSCredentials // Reusing the struct from aws_adapter.go
	if err := json.Unmarshal([]byte(credentialsJSON), &creds); err != nil {
		return nil, fmt.Errorf("failed to parse AWS IAM credentials JSON: %w", err)
	}

	var meta AWSIAMMetadata
	if err := json.Unmarshal([]byte(metadataJSON), &meta); err != nil {
		return nil, fmt.Errorf("failed to parse AWS IAM metadata JSON: %w", err)
	}

	if meta.Region == "" {
		meta.Region = "us-east-1" // IAM is global, but the SDK requires a region
	}

	cfg, err := config.LoadDefaultConfig(context.Background(),
		config.WithRegion(meta.Region),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(creds.AccessKeyID, creds.SecretAccessKey, "")),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS IAM config: %w", err)
	}

	client := iam.NewFromConfig(cfg)

	return &AWSIAMProvider{
		client:    client,
		accountID: meta.AccountID,
	}, nil
}

// generateIAMUsername converts an email address into a valid IAM username.
// It replaces characters that IAM does not allow.
func generateIAMUsername(email string) string {
	// IAM usernames can only contain alphanumeric characters and +=,.@_-
	// We use the full email address.
	return email
}

// generateRandomPassword creates a strong random password that meets typical AWS requirements.
func generateRandomPassword() string {
	const chars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!@#$%^&*"
	rand.Seed(time.Now().UnixNano())
	length := 16
	b := make([]byte, length)
	for i := range b {
		b[i] = chars[rand.Intn(len(chars))]
	}
	return string(b)
}

// ResolveUser checks if an IAM user exists. If not, it provisions one.
func (p *AWSIAMProvider) ResolveUser(ctx context.Context, email string) (string, error) {
	username := generateIAMUsername(email)

	input := &iam.GetUserInput{
		UserName: aws.String(username),
	}

	_, err := p.client.GetUser(ctx, input)
	if err != nil {
		var noSuchEntity *types.NoSuchEntityException
		if errors.As(err, &noSuchEntity) {
			// User does not exist, auto-provision them
			createInput := &iam.CreateUserInput{
				UserName: aws.String(username),
				Tags: []types.Tag{
					{Key: aws.String("ManagedBy"), Value: aws.String("JIT")},
					{Key: aws.String("Email"), Value: aws.String(email)},
				},
			}
			_, createErr := p.client.CreateUser(ctx, createInput)
			if createErr != nil {
				return "", fmt.Errorf("failed to auto-provision IAM user %s: %w", username, createErr)
			}
			return username, nil
		}
		return "", fmt.Errorf("failed to fetch IAM user %s: %w", username, err)
	}

	return username, nil
}

// GrantAccess adds the user to an IAM Group and optionally creates login credentials.
func (p *AWSIAMProvider) GrantAccess(ctx context.Context, req AccessRequest) (*AccessResult, error) {
	username, err := p.ResolveUser(ctx, req.UserEmail)
	if err != nil {
		return nil, err
	}

	// 1. Add to the requested IAM Group
	groupInput := &iam.AddUserToGroupInput{
		GroupName: aws.String(req.TargetGroupID),
		UserName:  aws.String(username),
	}

	_, err = p.client.AddUserToGroup(ctx, groupInput)
	if err != nil {
		return nil, fmt.Errorf("failed to add IAM user %s to group %s: %w", username, req.TargetGroupID, err)
	}

	result := &AccessResult{
		Username: username,
	}

	// 2. Generate Console Password if requested
	if req.GeneratePassword {
		password := generateRandomPassword()
		loginProfileInput := &iam.CreateLoginProfileInput{
			UserName:              aws.String(username),
			Password:              aws.String(password),
			PasswordResetRequired: false, // Or true if you want them to change it immediately
		}

		// First, check if they already have a login profile (e.g., from a previous request)
		_, err := p.client.GetLoginProfile(ctx, &iam.GetLoginProfileInput{UserName: aws.String(username)})
		if err == nil {
			// They have one, so we must Update it instead of Create
			updateProfileInput := &iam.UpdateLoginProfileInput{
				UserName: aws.String(username),
				Password: aws.String(password),
			}
			_, err = p.client.UpdateLoginProfile(ctx, updateProfileInput)
			if err != nil {
				return nil, fmt.Errorf("failed to update IAM console password for %s: %w", username, err)
			}
		} else {
			// They don't have one, create it
			_, err = p.client.CreateLoginProfile(ctx, loginProfileInput)
			if err != nil {
				return nil, fmt.Errorf("failed to create IAM console password for %s: %w", username, err)
			}
		}

		result.Password = password
		if p.accountID != "" {
			result.ConsoleURL = fmt.Sprintf("https://%s.signin.aws.amazon.com/console", p.accountID)
		}
	}

	// 3. Generate Programmatic Access Keys if requested
	if req.GenerateAccessKey {
		keyInput := &iam.CreateAccessKeyInput{
			UserName: aws.String(username),
		}

		keyOutput, err := p.client.CreateAccessKey(ctx, keyInput)
		if err != nil {
			return nil, fmt.Errorf("failed to generate IAM access keys for %s: %w", username, err)
		}

		result.AccessKeyID = aws.ToString(keyOutput.AccessKey.AccessKeyId)
		result.SecretAccessKey = aws.ToString(keyOutput.AccessKey.SecretAccessKey)
	}

	return result, nil
}

// RevokeAccess removes the user from the IAM Group, deletes their access keys, and deletes their login profile.
func (p *AWSIAMProvider) RevokeAccess(ctx context.Context, req AccessRequest) error {
	username := generateIAMUsername(req.UserEmail)

	// Check if user even exists before trying to revoke
	_, err := p.client.GetUser(ctx, &iam.GetUserInput{UserName: aws.String(username)})
	if err != nil {
		var noSuchEntity *types.NoSuchEntityException
		if errors.As(err, &noSuchEntity) {
			return nil // User is already gone
		}
		return err
	}

	// 1. Remove from the IAM Group
	removeGroupInput := &iam.RemoveUserFromGroupInput{
		GroupName: aws.String(req.TargetGroupID),
		UserName:  aws.String(username),
	}
	_, err = p.client.RemoveUserFromGroup(ctx, removeGroupInput)
	if err != nil {
		var noSuchEntity *types.NoSuchEntityException
		if !errors.As(err, &noSuchEntity) {
			return fmt.Errorf("failed to remove IAM user %s from group %s: %w", username, req.TargetGroupID, err)
		}
	}

	// 2. Delete Access Keys
	// We must list them first because we only have the username, not the Key ID
	listKeysInput := &iam.ListAccessKeysInput{
		UserName: aws.String(username),
	}
	listKeysOutput, err := p.client.ListAccessKeys(ctx, listKeysInput)
	if err == nil {
		for _, key := range listKeysOutput.AccessKeyMetadata {
			delKeyInput := &iam.DeleteAccessKeyInput{
				UserName:    aws.String(username),
				AccessKeyId: key.AccessKeyId,
			}
			_, _ = p.client.DeleteAccessKey(ctx, delKeyInput) // Ignore error on cleanup
		}
	}

	// 3. Delete Console Password (Login Profile)
	delProfileInput := &iam.DeleteLoginProfileInput{
		UserName: aws.String(username),
	}
	_, _ = p.client.DeleteLoginProfile(ctx, delProfileInput) // Ignore error if they don't have one

	return nil
}

// TestConnection attempts a lightweight call to verify the connection and credentials.
func (p *AWSIAMProvider) TestConnection(ctx context.Context) error {
	input := &iam.ListGroupsInput{
		MaxItems: aws.Int32(1),
	}

	_, err := p.client.ListGroups(ctx, input)
	if err != nil {
		return fmt.Errorf("AWS IAM connection test failed: %w", err)
	}

	return nil
}
