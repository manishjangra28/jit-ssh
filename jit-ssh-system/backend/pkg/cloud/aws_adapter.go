package cloud

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/identitystore"
	"github.com/aws/aws-sdk-go-v2/service/identitystore/types"
)

// AWSCredentials represents the expected structure of the decrypted AWS credentials JSON.
type AWSCredentials struct {
	AccessKeyID     string `json:"access_key_id"`
	SecretAccessKey string `json:"secret_access_key"`
}

// AWSMetadata represents the non-secret configuration for AWS integration.
type AWSMetadata struct {
	Region          string `json:"region"`
	IdentityStoreID string `json:"identity_store_id"`
	SSOStartURL     string `json:"sso_start_url"`
}

// AWSProvider implements the Provider interface for AWS IAM Identity Center.
type AWSProvider struct {
	client          *identitystore.Client
	identityStoreID string
	ssoStartURL     string
}

// NewAWSProvider creates and initializes a new AWS Provider.
func NewAWSProvider(credentialsJSON string, metadataJSON string) (Provider, error) {
	var creds AWSCredentials
	if err := json.Unmarshal([]byte(credentialsJSON), &creds); err != nil {
		return nil, fmt.Errorf("failed to parse AWS credentials JSON: %w", err)
	}

	var meta AWSMetadata
	if err := json.Unmarshal([]byte(metadataJSON), &meta); err != nil {
		return nil, fmt.Errorf("failed to parse AWS metadata JSON: %w", err)
	}

	if meta.IdentityStoreID == "" {
		return nil, errors.New("identity_store_id is required in AWS metadata")
	}

	cfg, err := config.LoadDefaultConfig(context.Background(),
		config.WithRegion(meta.Region),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(creds.AccessKeyID, creds.SecretAccessKey, "")),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	client := identitystore.NewFromConfig(cfg)

	return &AWSProvider{
		client:          client,
		identityStoreID: meta.IdentityStoreID,
		ssoStartURL:     meta.SSOStartURL,
	}, nil
}

// ResolveUser attempts to find an AWS Identity Store user by their email address.
func (p *AWSProvider) ResolveUser(ctx context.Context, email string) (string, error) {
	// Identity Store uses an AlternateIdentifier to look up by specific attributes like UserName or Email.
	// To look up a user, we use ListUsers with a UserName filter.
	// In AWS Identity Center, UserName is typically the user's email address.
	input := &identitystore.ListUsersInput{
		IdentityStoreId: aws.String(p.identityStoreID),
		Filters: []types.Filter{
			{
				AttributePath:  aws.String("UserName"),
				AttributeValue: aws.String(email),
			},
		},
	}

	output, err := p.client.ListUsers(ctx, input)
	if err != nil {
		return "", fmt.Errorf("failed to list users for %s in AWS Identity Store: %w", email, err)
	}

	if len(output.Users) == 0 {
		// Auto-provision the user if they don't exist
		createInput := &identitystore.CreateUserInput{
			IdentityStoreId: aws.String(p.identityStoreID),
			UserName:        aws.String(email),
			DisplayName:     aws.String(email),
			Name: &types.Name{
				FamilyName: aws.String("JIT"),
				GivenName:  aws.String("User"),
			},
			Emails: []types.Email{
				{
					Primary: true,
					Value:   aws.String(email),
					Type:    aws.String("work"),
				},
			},
		}

		createOutput, createErr := p.client.CreateUser(ctx, createInput)
		if createErr != nil {
			return "", fmt.Errorf("user %s not found in AWS Identity Store and auto-provisioning failed: %w", email, createErr)
		}
		return aws.ToString(createOutput.UserId), nil
	}

	return aws.ToString(output.Users[0].UserId), nil
}

// GrantAccess adds the user to the specified AWS Identity Store group.
func (p *AWSProvider) GrantAccess(ctx context.Context, req AccessRequest) (*AccessResult, error) {
	userID, err := p.ResolveUser(ctx, req.UserEmail)
	if err != nil {
		return nil, err
	}

	input := &identitystore.CreateGroupMembershipInput{
		IdentityStoreId: aws.String(p.identityStoreID),
		GroupId:         aws.String(req.TargetGroupID),
		MemberId: &types.MemberIdMemberUserId{
			Value: userID,
		},
	}

	_, err = p.client.CreateGroupMembership(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to add user %s to group %s: %w", req.UserEmail, req.TargetGroupID, err)
	}

	result := &AccessResult{}
	if p.ssoStartURL != "" {
		result.ConsoleURL = p.ssoStartURL
	}

	return result, nil
}

// RevokeAccess removes the user from the specified AWS Identity Store group.
func (p *AWSProvider) RevokeAccess(ctx context.Context, req AccessRequest) error {
	userID, err := p.ResolveUser(ctx, req.UserEmail)
	if err != nil {
		// If the user doesn't exist, technically the access is already gone.
		return nil
	}

	// To delete a membership, we first need to find its MembershipId
	getMembershipInput := &identitystore.GetGroupMembershipIdInput{
		IdentityStoreId: aws.String(p.identityStoreID),
		GroupId:         aws.String(req.TargetGroupID),
		MemberId: &types.MemberIdMemberUserId{
			Value: userID,
		},
	}

	membershipOutput, err := p.client.GetGroupMembershipId(ctx, getMembershipInput)
	if err != nil {
		// If the membership doesn't exist, we consider the revocation successful (idempotency).
		return nil
	}

	// Now delete using the MembershipId
	deleteInput := &identitystore.DeleteGroupMembershipInput{
		IdentityStoreId: aws.String(p.identityStoreID),
		MembershipId:    membershipOutput.MembershipId,
	}

	_, err = p.client.DeleteGroupMembership(ctx, deleteInput)
	if err != nil {
		return fmt.Errorf("failed to remove user %s from group %s: %w", req.UserEmail, req.TargetGroupID, err)
	}

	return nil
}

// ListGroups fetches all available groups from AWS Identity Store.
func (p *AWSProvider) ListGroups(ctx context.Context) ([]CloudGroup, error) {
	input := &identitystore.ListGroupsInput{
		IdentityStoreId: aws.String(p.identityStoreID),
	}

	var groups []CloudGroup

	paginator := identitystore.NewListGroupsPaginator(p.client, input)
	for paginator.HasMorePages() {
		output, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to list AWS Identity Store groups: %w", err)
		}

		for _, group := range output.Groups {
			groups = append(groups, CloudGroup{
				ID:   aws.ToString(group.GroupId),
				Name: aws.ToString(group.DisplayName),
			})
		}
	}

	return groups, nil
}

// TestConnection attempts a lightweight call to verify the connection and credentials.
func (p *AWSProvider) TestConnection(ctx context.Context) error {
	// A lightweight call: List groups with a max result of 1
	input := &identitystore.ListGroupsInput{
		IdentityStoreId: aws.String(p.identityStoreID),
		MaxResults:      aws.Int32(1),
	}

	_, err := p.client.ListGroups(ctx, input)
	if err != nil {
		return fmt.Errorf("AWS connection test failed: %w", err)
	}

	return nil
}
