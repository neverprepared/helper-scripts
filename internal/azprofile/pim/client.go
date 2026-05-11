// Ported from github.com/neverprepared/az-pim-cli/pkg/pim/client.go.
// Original: Copyright © 2023 netr0m <netr0m@pm.me>. Reorganized for
// azprofile: returns errors instead of calling os.Exit, drops azidentity
// + jwt deps, drops the Client interface (callers use the package
// functions directly).
package pim

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/google/uuid"
)

// Client wraps an ARM base URL and the http.Client used for requests.
type Client struct {
	ARMBaseURL string
	HTTP       *http.Client
}

// New returns a Client configured for the named cloud (global/usgov/china).
// Unknown clouds fall back to global.
func New(cloud string) *Client {
	base, ok := ARMBaseURLs[cloud]
	if !ok {
		base = ARMGlobalBaseURL
	}
	return &Client{ARMBaseURL: base, HTTP: http.DefaultClient}
}

const maxRetries = 2 // up to 3 total attempts (initial + 2 retries)

// Request performs the HTTP exchange described by req and JSON-decodes the
// response body into responseModel. Retries on transport errors, 429, and
// 5xx with exponential backoff.
func (c *Client) Request(ctx context.Context, req *PIMRequest, responseModel any) error {
	httpClient := c.HTTP
	if httpClient == nil {
		httpClient = http.DefaultClient
	}

	var payloadBytes []byte
	if req.Payload != nil {
		buf := new(bytes.Buffer)
		if err := json.NewEncoder(buf).Encode(req.Payload); err != nil {
			return fmt.Errorf("encode payload: %w", err)
		}
		payloadBytes = buf.Bytes()
	}

	var (
		httpReq *http.Request
		res     *http.Response
		body    []byte
		err     error
	)

	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			backoff := time.Duration(1<<(attempt-1)) * time.Second
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(backoff):
			}
		}

		var bodyReader io.Reader
		if payloadBytes != nil {
			bodyReader = bytes.NewReader(payloadBytes)
		}
		httpReq, err = http.NewRequestWithContext(ctx, req.Method, req.URL, bodyReader)
		if err != nil {
			return fmt.Errorf("build request: %w", err)
		}
		httpReq.Header.Set("Content-Type", "application/json")
		httpReq.Header.Set("Accept", "application/json")
		httpReq.Header.Set("Authorization", "Bearer "+req.Token)

		q := httpReq.URL.Query()
		for k, v := range req.Params {
			q.Set(k, v)
		}
		httpReq.URL.RawQuery = q.Encode()

		res, err = httpClient.Do(httpReq)
		if err != nil {
			if attempt < maxRetries {
				continue
			}
			return fmt.Errorf("%s %s: %w", req.Method, req.URL, err)
		}

		body, err = io.ReadAll(res.Body)
		_ = res.Body.Close()
		if err != nil {
			if attempt < maxRetries {
				continue
			}
			return fmt.Errorf("read response: %w", err)
		}

		if res.StatusCode == 429 || res.StatusCode >= 500 {
			if attempt < maxRetries {
				continue
			}
		}
		break
	}

	if res.StatusCode >= 400 {
		return fmt.Errorf("%s %s: %s: %s", req.Method, req.URL, res.Status, string(body))
	}
	if responseModel == nil {
		return nil
	}
	if err := json.Unmarshal(body, responseModel); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}
	return nil
}

// ── Resource (Azure resource) role assignments ───────────────────────

func (c *Client) GetEligibleResourceAssignments(ctx context.Context, token string) (*ResourceAssignmentResponse, error) {
	resp := &ResourceAssignmentResponse{}
	err := c.Request(ctx, &PIMRequest{
		URL:    fmt.Sprintf("%s/%s/roleEligibilityScheduleInstances", c.ARMBaseURL, ARMBasePath),
		Token:  token,
		Method: http.MethodGet,
		Params: map[string]string{
			"api-version": PIMAPIVersion,
			"$filter":     "asTarget()",
		},
	}, resp)
	return resp, err
}

func (c *Client) GetActiveResourceAssignments(ctx context.Context, token string) (*ActiveResourceAssignmentResponse, error) {
	resp := &ActiveResourceAssignmentResponse{}
	err := c.Request(ctx, &PIMRequest{
		URL:    fmt.Sprintf("%s/%s/roleAssignmentScheduleInstances", c.ARMBaseURL, ARMBasePath),
		Token:  token,
		Method: http.MethodGet,
		Params: map[string]string{
			"api-version": PIMAPIVersion,
			"$filter":     "asTarget()",
		},
	}, resp)
	return resp, err
}

func (c *Client) RequestResourceAssignment(ctx context.Context, scope string, body *ResourceAssignmentRequestRequest, token string) (*ResourceAssignmentRequestResponse, error) {
	resp := &ResourceAssignmentRequestResponse{}
	err := c.Request(ctx, &PIMRequest{
		URL: fmt.Sprintf("%s/%s/%s/roleAssignmentScheduleRequests/%s",
			c.ARMBaseURL, scope, ARMBasePath, uuid.NewString()),
		Token:   token,
		Method:  http.MethodPut,
		Params:  map[string]string{"api-version": PIMAPIVersion},
		Payload: body,
	}, resp)
	return resp, err
}

func (c *Client) GetResourceAssignmentRequest(ctx context.Context, scope, name, token string) (*ResourceAssignmentRequestResponse, error) {
	resp := &ResourceAssignmentRequestResponse{}
	err := c.Request(ctx, &PIMRequest{
		URL: fmt.Sprintf("%s/%s/%s/roleAssignmentScheduleRequests/%s",
			c.ARMBaseURL, scope, ARMBasePath, name),
		Token:  token,
		Method: http.MethodGet,
		Params: map[string]string{"api-version": PIMAPIVersion},
	}, resp)
	return resp, err
}

// ── Governance (Entra roles + AAD groups) role assignments ───────────

func (c *Client) GetEligibleGovernanceRoleAssignments(ctx context.Context, roleType, subjectID, token string) (*GovernanceRoleAssignmentResponse, error) {
	if !IsGovernanceRoleType(roleType) {
		return nil, fmt.Errorf("invalid governance role type %q (want %q or %q)", roleType, RoleTypeAADGroups, RoleTypeEntraRoles)
	}
	resp := &GovernanceRoleAssignmentResponse{}
	err := c.Request(ctx, &PIMRequest{
		URL:    fmt.Sprintf("%s/%s/%s/roleAssignments", RBACBaseURL, RBACBasePath, roleType),
		Token:  token,
		Method: http.MethodGet,
		Params: map[string]string{
			"$expand": "linkedEligibleRoleAssignment,subject,scopedResource,roleDefinition($expand=resource)",
			"$filter": fmt.Sprintf("(subject/id eq '%s') and (assignmentState eq 'Eligible')", subjectID),
		},
	}, resp)
	return resp, err
}

func (c *Client) GetActiveGovernanceRoleAssignments(ctx context.Context, roleType, subjectID, token string) (*GovernanceRoleAssignmentResponse, error) {
	if !IsGovernanceRoleType(roleType) {
		return nil, fmt.Errorf("invalid governance role type %q", roleType)
	}
	resp := &GovernanceRoleAssignmentResponse{}
	err := c.Request(ctx, &PIMRequest{
		URL:    fmt.Sprintf("%s/%s/%s/roleAssignments", RBACBaseURL, RBACBasePath, roleType),
		Token:  token,
		Method: http.MethodGet,
		Params: map[string]string{
			"$expand": "linkedEligibleRoleAssignment,subject,scopedResource,roleDefinition($expand=resource)",
			"$filter": fmt.Sprintf("(subject/id eq '%s') and (assignmentState eq 'Active')", subjectID),
		},
	}, resp)
	return resp, err
}

func (c *Client) RequestGovernanceRoleAssignment(ctx context.Context, roleType string, body *GovernanceRoleAssignmentRequest, token string) (*GovernanceRoleAssignmentRequestResponse, error) {
	if !IsGovernanceRoleType(roleType) {
		return nil, fmt.Errorf("invalid governance role type %q", roleType)
	}
	resp := &GovernanceRoleAssignmentRequestResponse{}
	err := c.Request(ctx, &PIMRequest{
		URL:     fmt.Sprintf("%s/%s/%s/roleAssignmentRequests", RBACBaseURL, RBACBasePath, roleType),
		Token:   token,
		Method:  http.MethodPost,
		Payload: body,
	}, resp)
	return resp, err
}

func (c *Client) GetGovernanceRoleAssignmentRequest(ctx context.Context, roleType, id, token string) (*GovernanceRoleAssignmentRequestResponse, error) {
	if !IsGovernanceRoleType(roleType) {
		return nil, fmt.Errorf("invalid governance role type %q", roleType)
	}
	resp := &GovernanceRoleAssignmentRequestResponse{}
	err := c.Request(ctx, &PIMRequest{
		URL:    fmt.Sprintf("%s/%s/%s/roleAssignmentRequests/%s", RBACBaseURL, RBACBasePath, roleType, id),
		Token:  token,
		Method: http.MethodGet,
	}, resp)
	return resp, err
}
