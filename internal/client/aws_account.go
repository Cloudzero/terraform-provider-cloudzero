//  SPDX-FileCopyrightText: Copyright (c) CloudZero, Inc. or its affiliates. All Rights Reserved.
//  SPDX-License-Identifier: Apache-2.0

package client

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

// AWSAccountLinkRequest is the payload POSTed to /accounts/v1/link.
// Matches the format the CloudZero Reactor expects from the notification Lambda.
type AWSAccountLinkRequest struct {
	Version       string                `json:"version"`
	MessageSource string                `json:"message_source"`
	MessageType   string                `json:"message_type"`
	Data          AWSAccountLinkData    `json:"data"`
}

type AWSAccountLinkData struct {
	Metadata  AWSAccountLinkMetadata  `json:"metadata"`
	Links     AWSAccountLinkLinks     `json:"links"`
	Discovery AWSAccountLinkDiscovery `json:"discovery"`
}

type AWSAccountLinkMetadata struct {
	CloudRegion        string `json:"cloud_region"`
	ExternalID         string `json:"external_id"`
	CloudAccountID     string `json:"cloud_account_id"`
	CZAccountName      string `json:"cz_account_name"`
	ReactorID          string `json:"reactor_id"`
	ReactorCallbackURL string `json:"reactor_callback_url"`
}

type AWSAccountLinkRole struct {
	RoleARN *string `json:"role_arn"`
}

type AWSAccountLinkLinks struct {
	Audit          AWSAccountLinkRole          `json:"audit"`
	CloudTrailOwner AWSAccountLinkCloudTrail   `json:"cloudtrail_owner"`
	MasterPayer    AWSAccountLinkRole          `json:"master_payer"`
	ResourceOwner  AWSAccountLinkRole          `json:"resource_owner"`
	Legacy         AWSAccountLinkRole          `json:"legacy"`
}

type AWSAccountLinkCloudTrail struct {
	SQSQueueARN        *string `json:"sqs_queue_arn"`
	SQSQueuePolicyName *string `json:"sqs_queue_policy_name"`
}

type AWSAccountLinkDiscovery struct {
	AuditCloudTrailBucketName    *string `json:"audit_cloudtrail_bucket_name"`
	AuditCloudTrailBucketPrefix  *string `json:"audit_cloudtrail_bucket_prefix"`
	CloudTrailSNSTopicARN        *string `json:"cloudtrail_sns_topic_arn"`
	CloudTrailTrailARN           *string `json:"cloudtrail_trail_arn"`
	IsAuditAccount               bool    `json:"is_audit_account"`
	IsCloudTrailOwnerAccount     bool    `json:"is_cloudtrail_owner_account"`
	IsMasterPayerAccount         bool    `json:"is_master_payer_account"`
	IsOrganizationMasterAccount  bool    `json:"is_organization_master_account"`
	IsOrganizationTrail          *bool   `json:"is_organization_trail"`
	IsResourceOwnerAccount       bool    `json:"is_resource_owner_account"`
	MasterPayerBillingBucketName *string `json:"master_payer_billing_bucket_name"`
	MasterPayerBillingBucketPath *string `json:"master_payer_billing_bucket_path"`
	RemoteCloudTrailBucket       bool    `json:"remote_cloudtrail_bucket"`
	VisibleCloudTrailARNs        *string `json:"visible_cloudtrail_arns"`
}

type AWSAccountLinkResponse struct {
	Success       bool   `json:"success"`
	TransactionID string `json:"transaction_id"`
	Result        string `json:"result"`
}

// RegisterAWSAccount posts to /accounts/v1/link to register an AWS account with CloudZero.
// Retries on AssumeRole errors to handle IAM propagation delay after role creation.
func (c *Client) RegisterAWSAccount(ctx context.Context, req AWSAccountLinkRequest) (*AWSAccountLinkResponse, error) {
	maxAttempts := 6
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		var resp AWSAccountLinkResponse
		err := c.do(ctx, http.MethodPost, "/accounts/v1/link", req, &resp)

		if err == nil {
			return &resp, nil
		}

		// Retry on AssumeRole errors (IAM propagation delay)
		errMsg := err.Error()
		if attempt < maxAttempts && (contains(errMsg, "AssumeRole") || contains(errMsg, "AccessDenied")) {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(time.Duration(attempt*5) * time.Second):
				continue
			}
		}

		return nil, err
	}

	return nil, fmt.Errorf("max registration attempts exceeded")
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
