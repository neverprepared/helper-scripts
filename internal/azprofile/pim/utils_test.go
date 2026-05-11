package pim

import (
	"strings"
	"testing"
)

func mkResource(name, role, scopeID string) *ResourceAssignment {
	return &ResourceAssignment{
		Name: name,
		Properties: &ResourceProperties{
			RoleEligibilityScheduleID: "sched-" + name,
			ExpandedProperties: &ResourceExpandedProperties{
				Scope:          &ResourceExpandedProperty{ID: scopeID, DisplayName: name},
				RoleDefinition: &ResourceExpandedProperty{ID: "role-" + role, DisplayName: role},
				Principal:      &ResourceExpandedProperty{ID: "principal"},
			},
		},
	}
}

func TestNewResourceAssignmentRequestStripsLeadingSlash(t *testing.T) {
	ra := mkResource("eslcd", "Owner", "/subscriptions/abc/resourceGroups/x")
	scope, body, err := NewResourceAssignmentRequest("subj-1", ra, 60, "", "", "ops", "", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.HasPrefix(scope, "/") {
		t.Fatalf("scope should not start with /: %q", scope)
	}
	if body.Properties.RequestType != "SelfActivate" {
		t.Fatalf("RequestType = %q", body.Properties.RequestType)
	}
	if !body.Properties.IsActivativation {
		t.Fatal("IsActivativation must be true")
	}
	if body.Properties.IsValidationOnly {
		t.Fatal("IsValidationOnly must be false")
	}
	if body.Properties.LinkedRoleEligibilityScheduleID != "sched-eslcd" {
		t.Fatalf("LinkedRoleEligibilityScheduleId = %q", body.Properties.LinkedRoleEligibilityScheduleID)
	}
	if got := body.Properties.ScheduleInfo.Expiration.Duration; got != "PT60M" {
		t.Fatalf("duration = %q want PT60M", got)
	}
}

func TestNewResourceAssignmentRequestRejectsMissingExpand(t *testing.T) {
	ra := &ResourceAssignment{Name: "x", Properties: &ResourceProperties{}}
	if _, _, err := NewResourceAssignmentRequest("s", ra, 60, "", "", "r", "", ""); err == nil {
		t.Fatal("expected error for missing expanded properties")
	}
}

func TestStatusClassifiers(t *testing.T) {
	failed := &ResourceAssignmentRequestResponse{Properties: &ResourceAssignmentValidationProperties{Status: StatusDenied}}
	if !IsResourceRequestFailed(failed) {
		t.Fatal("denied should be failed")
	}
	if IsResourceRequestPending(failed) || IsResourceRequestOK(failed) {
		t.Fatal("denied is not pending/ok")
	}

	ok := &ResourceAssignmentRequestResponse{Properties: &ResourceAssignmentValidationProperties{Status: StatusProvisioned}}
	if !IsResourceRequestOK(ok) {
		t.Fatal("provisioned should be ok")
	}

	pending := &ResourceAssignmentRequestResponse{Properties: &ResourceAssignmentValidationProperties{Status: StatusPendingApproval}}
	if !IsResourceRequestPending(pending) {
		t.Fatal("pending-approval should be pending")
	}
}

func TestIsGovernanceRoleType(t *testing.T) {
	if !IsGovernanceRoleType(RoleTypeEntraRoles) {
		t.Fatal("entra roles should be valid")
	}
	if !IsGovernanceRoleType(RoleTypeAADGroups) {
		t.Fatal("aad groups should be valid")
	}
	if IsGovernanceRoleType("resource") {
		t.Fatal("resource should not be a governance role type")
	}
}

func TestNewResourceAssignmentScheduleInfoBadDate(t *testing.T) {
	if _, err := NewResourceAssignmentScheduleInfo(60, "not-a-date", ""); err == nil {
		t.Fatal("expected error parsing bad date")
	}
}
