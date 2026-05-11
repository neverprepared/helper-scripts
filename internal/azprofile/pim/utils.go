// Ported from github.com/neverprepared/az-pim-cli/pkg/pim/utils.go.
// Original: Copyright © 2024 netr0m <netr0m@pm.me>. Reorganized for
// azprofile: helpers return errors instead of os.Exit'ing.
package pim

import (
	"context"
	"fmt"
	"time"
)

// IsGovernanceRoleType reports whether s is a recognized PIM governance role
// type (Entra roles or AAD groups).
func IsGovernanceRoleType(s string) bool {
	return s == RoleTypeAADGroups || s == RoleTypeEntraRoles
}

// IsResourceRequestFailed reports whether a resource assignment request's
// status indicates failure.
func IsResourceRequestFailed(r *ResourceAssignmentRequestResponse) bool {
	switch r.Properties.Status {
	case StatusAdminDenied, StatusCanceled, StatusDenied, StatusFailed,
		StatusFailedAsResourceIsLocked, StatusInvalid, StatusRevoked, StatusTimedOut:
		return true
	}
	return false
}

func IsResourceRequestPending(r *ResourceAssignmentRequestResponse) bool {
	switch r.Properties.Status {
	case StatusPendingAdminDecision, StatusPendingApproval, StatusPendingApprovalProvisioning,
		StatusPendingEvaluation, StatusPendingExternalProvisioning, StatusPendingProvisioning,
		StatusPendingRevocation, StatusPendingScheduleCreation:
		return true
	}
	return false
}

func IsResourceRequestOK(r *ResourceAssignmentRequestResponse) bool {
	switch r.Properties.Status {
	case StatusAccepted, StatusAdminApproved, StatusGranted, StatusProvisioned,
		StatusProvisioningStarted, StatusScheduleCreated:
		return true
	}
	return false
}

func IsGovernanceRequestFailed(r *GovernanceRoleAssignmentRequestResponse) bool {
	if r.Status == nil {
		return false
	}
	switch r.Status.SubStatus {
	case StatusAdminDenied, StatusCanceled, StatusDenied, StatusFailed,
		StatusFailedAsResourceIsLocked, StatusInvalid, StatusRevoked, StatusTimedOut:
		return true
	}
	return false
}

func IsGovernanceRequestPending(r *GovernanceRoleAssignmentRequestResponse) bool {
	if r.Status == nil {
		return false
	}
	switch r.Status.SubStatus {
	case StatusPendingAdminDecision, StatusPendingApproval, StatusPendingApprovalProvisioning,
		StatusPendingEvaluation, StatusPendingExternalProvisioning, StatusPendingProvisioning,
		StatusPendingRevocation, StatusPendingScheduleCreation:
		return true
	}
	return false
}

func IsGovernanceRequestOK(r *GovernanceRoleAssignmentRequestResponse) bool {
	if r.Status == nil {
		return false
	}
	switch r.Status.SubStatus {
	case StatusAccepted, StatusAdminApproved, StatusGranted, StatusProvisioned,
		StatusProvisioningStarted, StatusScheduleCreated:
		return true
	}
	return false
}

// NewResourceAssignmentScheduleInfo constructs a ScheduleInfo with the given
// duration in minutes. If startDate/startTime are non-empty they are parsed
// as DD/MM/YYYY and HH:MM respectively in the local timezone.
func NewResourceAssignmentScheduleInfo(durationMin int, startDate, startTime string) (*ScheduleInfo, error) {
	var start any
	if startDate != "" || startTime != "" {
		t, err := parseStartDateTime(startDate, startTime)
		if err != nil {
			return nil, err
		}
		start = t
	}
	return &ScheduleInfo{
		StartDateTime: start,
		Expiration: &ScheduleInfoExpiration{
			Type:     "AfterDuration",
			Duration: fmt.Sprintf("PT%dM", durationMin),
		},
	}, nil
}

// NewResourceAssignmentRequest builds the body for a SelfActivate request on
// a resource role. Returns the scope (extracted from the assignment's expanded
// properties) alongside the body so callers can build the request URL.
func NewResourceAssignmentRequest(subjectID string, ra *ResourceAssignment, durationMin int, startDate, startTime, reason, ticketSystem, ticketNumber string) (scope string, body *ResourceAssignmentRequestRequest, err error) {
	if ra.Properties == nil || ra.Properties.ExpandedProperties == nil ||
		ra.Properties.ExpandedProperties.RoleDefinition == nil ||
		ra.Properties.ExpandedProperties.Scope == nil {
		return "", nil, fmt.Errorf("resource assignment %s missing expanded properties", ra.Name)
	}
	sched, err := NewResourceAssignmentScheduleInfo(durationMin, startDate, startTime)
	if err != nil {
		return "", nil, err
	}
	body = &ResourceAssignmentRequestRequest{
		Properties: ResourceAssignmentRequestProperties{
			PrincipalID:                     subjectID,
			RoleDefinitionID:                ra.Properties.ExpandedProperties.RoleDefinition.ID,
			RequestType:                     "SelfActivate",
			LinkedRoleEligibilityScheduleID: ra.Properties.RoleEligibilityScheduleID,
			Justification:                   reason,
			ScheduleInfo:                    sched,
			TicketInfo:                      &TicketInfo{TicketNumber: ticketNumber, TicketSystem: ticketSystem},
			IsValidationOnly:                false,
			IsActivativation:                true,
		},
	}
	// Strip the leading slash from the scope ID — the API expects unrooted
	// scopes when concatenated into the URL.
	scope = ra.Properties.ExpandedProperties.Scope.ID
	if len(scope) > 0 && scope[0] == '/' {
		scope = scope[1:]
	}
	return scope, body, nil
}

func NewResourceDeactivationRequest(subjectID string, active *ActiveResourceAssignment) (scope string, body *ResourceAssignmentRequestRequest, err error) {
	if active.Properties == nil || active.Properties.ExpandedProperties == nil ||
		active.Properties.ExpandedProperties.RoleDefinition == nil ||
		active.Properties.ExpandedProperties.Scope == nil {
		return "", nil, fmt.Errorf("active assignment %s missing expanded properties", active.Name)
	}
	body = &ResourceAssignmentRequestRequest{
		Properties: ResourceAssignmentRequestProperties{
			PrincipalID:                     subjectID,
			RoleDefinitionID:                active.Properties.ExpandedProperties.RoleDefinition.ID,
			RequestType:                     "SelfDeactivate",
			LinkedRoleEligibilityScheduleID: active.Properties.LinkedRoleEligibilityScheduleID,
			IsValidationOnly:                false,
			IsActivativation:                false,
		},
	}
	scope = active.Properties.ExpandedProperties.Scope.ID
	if len(scope) > 0 && scope[0] == '/' {
		scope = scope[1:]
	}
	return scope, body, nil
}

func NewGovernanceAssignmentScheduleInfo(durationMin int, startDate, startTime string) (*GovernanceRoleAssignmentSchedule, error) {
	var start any
	if startDate != "" || startTime != "" {
		t, err := parseStartDateTime(startDate, startTime)
		if err != nil {
			return nil, err
		}
		start = t
	}
	return &GovernanceRoleAssignmentSchedule{
		Type:          "Once",
		StartDateTime: start,
		EndDateTime:   nil,
		Duration:      fmt.Sprintf("PT%dM", durationMin),
	}, nil
}

func NewGovernanceAssignmentRequest(subjectID, roleType string, ga *GovernanceRoleAssignment, durationMin int, startDate, startTime, reason, ticketSystem, ticketNumber string) (*GovernanceRoleAssignmentRequest, error) {
	if !IsGovernanceRoleType(roleType) {
		return nil, fmt.Errorf("invalid governance role type %q", roleType)
	}
	sched, err := NewGovernanceAssignmentScheduleInfo(durationMin, startDate, startTime)
	if err != nil {
		return nil, err
	}
	return &GovernanceRoleAssignmentRequest{
		RoleDefinitionID:               ga.RoleDefinitionID,
		ResourceID:                     ga.ResourceID,
		SubjectID:                      subjectID,
		AssignmentState:                "Active",
		Type:                           "UserAdd",
		Reason:                         reason,
		TicketNumber:                   ticketNumber,
		TicketSystem:                   ticketSystem,
		Schedule:                       sched,
		LinkedEligibleRoleAssignmentID: ga.ID,
		ScopedResourceID:               "",
	}, nil
}

func NewGovernanceDeactivationRequest(subjectID string, active *GovernanceRoleAssignment) *GovernanceRoleAssignmentRequest {
	return &GovernanceRoleAssignmentRequest{
		RoleDefinitionID:               active.RoleDefinitionID,
		ResourceID:                     active.ResourceID,
		SubjectID:                      subjectID,
		AssignmentState:                "Active",
		Type:                           "UserRemove",
		LinkedEligibleRoleAssignmentID: active.ID,
		ScopedResourceID:               "",
	}
}

// WaitForResourceAssignment polls the request status until it succeeds, fails,
// or the timeout elapses.
func (c *Client) WaitForResourceAssignment(ctx context.Context, scope, requestName, token string, timeoutSec int) error {
	deadline := time.Now().Add(time.Duration(timeoutSec) * time.Second)
	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(time.Duration(WaitPollIntervalSeconds) * time.Second):
		}
		resp, err := c.GetResourceAssignmentRequest(ctx, scope, requestName, token)
		if err != nil {
			return err
		}
		if IsResourceRequestOK(resp) {
			return nil
		}
		if IsResourceRequestFailed(resp) {
			return fmt.Errorf("activation failed: status=%s", resp.Properties.Status)
		}
	}
	return fmt.Errorf("timed out waiting %ds for role activation", timeoutSec)
}

func (c *Client) WaitForGovernanceRoleAssignment(ctx context.Context, roleType, requestID, token string, timeoutSec int) error {
	deadline := time.Now().Add(time.Duration(timeoutSec) * time.Second)
	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(time.Duration(WaitPollIntervalSeconds) * time.Second):
		}
		resp, err := c.GetGovernanceRoleAssignmentRequest(ctx, roleType, requestID, token)
		if err != nil {
			return err
		}
		if IsGovernanceRequestOK(resp) {
			return nil
		}
		if IsGovernanceRequestFailed(resp) {
			sub := ""
			if resp.Status != nil {
				sub = resp.Status.SubStatus
			}
			return fmt.Errorf("activation failed: substatus=%s", sub)
		}
	}
	return fmt.Errorf("timed out waiting %ds for role activation", timeoutSec)
}

func parseStartDateTime(dateStr, timeStr string) (string, error) {
	var d time.Time
	if dateStr == "" {
		now := time.Now().Local()
		d = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	} else {
		parsed, err := time.Parse("02/01/2006", dateStr)
		if err != nil {
			return "", fmt.Errorf("parse start date %q (want DD/MM/YYYY): %w", dateStr, err)
		}
		d = time.Date(parsed.Year(), parsed.Month(), parsed.Day(), 0, 0, 0, 0, time.Local)
	}
	if timeStr != "" {
		t, err := time.Parse("15:04", timeStr)
		if err != nil {
			return "", fmt.Errorf("parse start time %q (want HH:MM): %w", timeStr, err)
		}
		d = d.Add(time.Hour*time.Duration(t.Hour()) + time.Minute*time.Duration(t.Minute()))
	}
	return d.Format("2006-01-02T15:04:05-07:00"), nil
}
