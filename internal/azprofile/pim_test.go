package azprofile

import (
	"strings"
	"testing"

	"github.com/neverprepared/azprofile/internal/azprofile/pim"
)

func eligibleResource(name, role string) pim.ResourceAssignment {
	return pim.ResourceAssignment{
		Name: name,
		Properties: &pim.ResourceProperties{
			RoleEligibilityScheduleID: "sched-" + name,
			ExpandedProperties: &pim.ResourceExpandedProperties{
				Scope:          &pim.ResourceExpandedProperty{ID: "/subs/x", DisplayName: name, Type: "subscription"},
				RoleDefinition: &pim.ResourceExpandedProperty{ID: "role-" + role, DisplayName: role},
			},
		},
	}
}

func eligibleGov(name, role string) pim.GovernanceRoleAssignment {
	return pim.GovernanceRoleAssignment{
		ID:               "gov-" + name + "-" + role,
		RoleDefinitionID: "rd-" + role,
		RoleDefinition: &pim.GovernanceRoleDefinition{
			ID:          "rd-" + role,
			DisplayName: role,
			Resource:    &pim.GovernanceRoleResource{ID: "res-" + name, DisplayName: name},
		},
	}
}

func TestResolveActivateUniqueResource(t *testing.T) {
	res := []pim.ResourceAssignment{eligibleResource("eslcd", "Owner")}
	target, err := resolveActivate("eslcd", "", res, nil, nil)
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if target.kind != "resource" || target.resource == nil {
		t.Fatalf("expected resource target, got kind=%q", target.kind)
	}
}

func TestResolveActivateAmbiguousAcrossCategories(t *testing.T) {
	res := []pim.ResourceAssignment{eligibleResource("ops-team", "Owner")}
	entra := []pim.GovernanceRoleAssignment{eligibleGov("ops-team", "Reader")}
	_, err := resolveActivate("ops-team", "", res, entra, nil)
	if err == nil || !strings.Contains(err.Error(), "multiple categories") {
		t.Fatalf("expected multi-category error, got %v", err)
	}
}

func TestResolveActivateAmbiguousRoles(t *testing.T) {
	res := []pim.ResourceAssignment{
		eligibleResource("eslcd", "Owner"),
		eligibleResource("eslcd", "Contributor"),
	}
	_, err := resolveActivate("eslcd", "", res, nil, nil)
	if err == nil || !strings.Contains(err.Error(), "ambiguous") {
		t.Fatalf("expected ambiguous-role error, got %v", err)
	}
	// Role disambiguator should resolve it.
	target, err := resolveActivate("eslcd", "Contributor", res, nil, nil)
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if resourceRoleName(target.resource) != "Contributor" {
		t.Fatalf("expected Contributor target, got %q", resourceRoleName(target.resource))
	}
}

func TestResolveActivateNoMatch(t *testing.T) {
	res := []pim.ResourceAssignment{eligibleResource("eslcd", "Owner")}
	_, err := resolveActivate("missing", "", res, nil, nil)
	if err == nil || !strings.Contains(err.Error(), "no eligible") {
		t.Fatalf("expected no-match error, got %v", err)
	}
}

func TestResolveActivateCaseInsensitive(t *testing.T) {
	// Display name in Azure is mixed-case; user types lowercase.
	res := []pim.ResourceAssignment{eligibleResource("ESLCD", "Contributor")}
	target, err := resolveActivate("eslcd", "contributor", res, nil, nil)
	if err != nil {
		t.Fatalf("case-insensitive match should succeed: %v", err)
	}
	if target.kind != "resource" {
		t.Fatalf("kind=%q", target.kind)
	}
}

func TestSuggestEligibleNamesSuggestsClose(t *testing.T) {
	res := []pim.ResourceAssignment{
		eligibleResource("ESLCD", "Owner"),
		eligibleResource("ESL-Prod", "Owner"),
		eligibleResource("Sandbox", "Owner"),
	}
	got := suggestEligibleNames("esl", res, nil, nil)
	if !strings.Contains(got, "ESLCD") || !strings.Contains(got, "ESL-Prod") {
		t.Fatalf("expected ESLCD and ESL-Prod in suggestion, got: %q", got)
	}
	if strings.Contains(got, "Sandbox") {
		t.Fatalf("Sandbox is not close to 'esl', should not be suggested: %q", got)
	}
}

func TestSuggestEligibleNamesFallsBackToFullList(t *testing.T) {
	res := []pim.ResourceAssignment{eligibleResource("Sandbox", "Owner")}
	got := suggestEligibleNames("totally-unrelated", res, nil, nil)
	if !strings.Contains(got, "Sandbox") {
		t.Fatalf("with no close match, expected full list to include Sandbox, got: %q", got)
	}
}

func TestResolveActivateGovernanceUnique(t *testing.T) {
	entra := []pim.GovernanceRoleAssignment{eligibleGov("global-admin", "Global Administrator")}
	target, err := resolveActivate("global-admin", "", nil, entra, nil)
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if target.kind != "Entra role" || target.roleType != pim.RoleTypeEntraRoles {
		t.Fatalf("kind=%q roleType=%q", target.kind, target.roleType)
	}
}

func TestValidateType(t *testing.T) {
	for _, ok := range []string{"", "all", "resource", "role", "group"} {
		if _, err := validateType(ok); err != nil {
			t.Fatalf("%q should validate: %v", ok, err)
		}
	}
	if _, err := validateType("subscription"); err == nil {
		t.Fatal("invalid type should error")
	}
}
