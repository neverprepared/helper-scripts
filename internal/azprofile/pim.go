package azprofile

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/neverprepared/helper-scripts/internal/azprofile/pim"
	"github.com/neverprepared/helper-scripts/internal/ui"
)

// PimTypeAll matches any of the three PIM categories.
const (
	PimTypeAll      = "all"
	PimTypeResource = "resource"
	PimTypeRole     = "role"
	PimTypeGroup    = "group"
)

var validPimTypes = map[string]bool{
	PimTypeAll: true, PimTypeResource: true, PimTypeRole: true, PimTypeGroup: true,
}

// ActivateOptions are the user-supplied flags for `azprofile pim activate`.
type ActivateOptions struct {
	Type         string // "" or "all" → search all types
	Role         string // optional role-name disambiguator
	DurationMin  int
	Reason       string
	StartDate    string // DD/MM/YYYY
	StartTime    string // HH:MM
	TicketSystem string
	TicketNumber string
	Wait         bool
	WaitTimeout  int
}

// DeactivateOptions are the user-supplied flags for `azprofile pim deactivate`.
type DeactivateOptions struct {
	Type string
	Role string
}

func pimHeader() {
	current := GetCurrent()
	fmt.Printf("%s%sProfile:%s %s\n", ui.Bold, ui.Blue, ui.NC, current)
	fmt.Printf("%s──────────────%s\n", ui.Dim, ui.NC)
}

func validateType(t string) (string, error) {
	if t == "" {
		return PimTypeAll, nil
	}
	if !validPimTypes[t] {
		return "", fmt.Errorf("invalid --type %q (want one of: all, resource, role, group)", t)
	}
	return t, nil
}

// PimList prints eligible assignments. typeFilter is one of all/resource/role/group.
func PimList(typeFilter string) error {
	t, err := validateType(typeFilter)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	pimHeader()
	c := pim.New("global")

	armToken, rbacToken, subjectID, err := pimFetchTokens(ctx, t)
	if err != nil {
		return err
	}

	if t == PimTypeAll || t == PimTypeResource {
		resp, err := c.GetEligibleResourceAssignments(ctx, armToken)
		if err != nil {
			return fmt.Errorf("list resource: %w", err)
		}
		printResourceEligible(resp.Value)
	}
	if t == PimTypeAll || t == PimTypeRole {
		resp, err := c.GetEligibleGovernanceRoleAssignments(ctx, pim.RoleTypeEntraRoles, subjectID, rbacToken)
		if err != nil {
			return fmt.Errorf("list role: %w", err)
		}
		printGovernanceEligible("Entra roles", resp.Value)
	}
	if t == PimTypeAll || t == PimTypeGroup {
		resp, err := c.GetEligibleGovernanceRoleAssignments(ctx, pim.RoleTypeAADGroups, subjectID, rbacToken)
		if err != nil {
			return fmt.Errorf("list group: %w", err)
		}
		printGovernanceEligible("AAD groups", resp.Value)
	}
	return nil
}

// PimActive prints currently-active assignments.
func PimActive(typeFilter string) error {
	t, err := validateType(typeFilter)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	pimHeader()
	c := pim.New("global")

	armToken, rbacToken, subjectID, err := pimFetchTokens(ctx, t)
	if err != nil {
		return err
	}

	if t == PimTypeAll || t == PimTypeResource {
		resp, err := c.GetActiveResourceAssignments(ctx, armToken)
		if err != nil {
			return fmt.Errorf("list active resource: %w", err)
		}
		printResourceActive(resp.Value)
	}
	if t == PimTypeAll || t == PimTypeRole {
		resp, err := c.GetActiveGovernanceRoleAssignments(ctx, pim.RoleTypeEntraRoles, subjectID, rbacToken)
		if err != nil {
			return fmt.Errorf("list active role: %w", err)
		}
		printGovernanceActive("Entra roles", resp.Value)
	}
	if t == PimTypeAll || t == PimTypeGroup {
		resp, err := c.GetActiveGovernanceRoleAssignments(ctx, pim.RoleTypeAADGroups, subjectID, rbacToken)
		if err != nil {
			return fmt.Errorf("list active group: %w", err)
		}
		printGovernanceActive("AAD groups", resp.Value)
	}
	return nil
}

// PimActivate activates one or more eligible assignments by name. When opts.Type
// is "all" or empty, the lookup searches across all three categories and errors
// on ambiguity. When the chosen resource has multiple eligible roles, opts.Role
// disambiguates.
func PimActivate(names []string, opts ActivateOptions) error {
	t, err := validateType(opts.Type)
	if err != nil {
		return err
	}
	if opts.DurationMin <= 0 {
		opts.DurationMin = pim.DefaultDurationMinutes
	}
	if opts.Reason == "" {
		opts.Reason = pim.DefaultReason
	}
	if opts.WaitTimeout <= 0 {
		opts.WaitTimeout = pim.DefaultWaitTimeoutSeconds
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	pimHeader()
	c := pim.New("global")

	armToken, rbacToken, subjectID, err := pimFetchTokens(ctx, t)
	if err != nil {
		return err
	}

	resourceEligible, entraEligible, groupEligible, err := pimFetchEligible(ctx, c, t, subjectID, armToken, rbacToken)
	if err != nil {
		return err
	}

	for _, name := range names {
		match, err := resolveActivate(name, opts.Role, resourceEligible, entraEligible, groupEligible)
		if err != nil {
			return fmt.Errorf("%s: %w", name, err)
		}
		fmt.Printf("\n%s%s%s Activating %s%s%s (%s)\n",
			ui.Cyan, ui.Arrow, ui.NC, ui.Bold, name, ui.NC, match.kind)
		if err := match.activate(ctx, c, subjectID, armToken, rbacToken, opts); err != nil {
			return fmt.Errorf("%s: %w", name, err)
		}
		fmt.Printf("%s%s%s Activation complete\n", ui.Green, ui.Check, ui.NC)
	}
	return nil
}

// PimDeactivate releases currently-active assignments by name.
func PimDeactivate(names []string, opts DeactivateOptions) error {
	t, err := validateType(opts.Type)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	pimHeader()
	c := pim.New("global")

	armToken, rbacToken, subjectID, err := pimFetchTokens(ctx, t)
	if err != nil {
		return err
	}

	resourceActive, entraActive, groupActive, err := pimFetchActive(ctx, c, t, subjectID, armToken, rbacToken)
	if err != nil {
		return err
	}

	for _, name := range names {
		match, err := resolveDeactivate(name, opts.Role, resourceActive, entraActive, groupActive)
		if err != nil {
			return fmt.Errorf("%s: %w", name, err)
		}
		fmt.Printf("\n%s%s%s Deactivating %s%s%s (%s)\n",
			ui.Cyan, ui.Arrow, ui.NC, ui.Bold, name, ui.NC, match.kind)
		if err := match.deactivate(ctx, c, subjectID, armToken, rbacToken); err != nil {
			return fmt.Errorf("%s: %w", name, err)
		}
		fmt.Printf("%s%s%s Deactivation complete\n", ui.Green, ui.Check, ui.NC)
	}
	return nil
}

// ── Token + listing helpers ──────────────────────────────────────────

func pimFetchTokens(ctx context.Context, t string) (armToken, rbacToken, subjectID string, err error) {
	_ = ctx // tokens come from `az`, not a context-aware HTTP call
	if t == PimTypeAll || t == PimTypeResource {
		armToken, err = pim.GetAccessToken(pim.ARMGlobalBaseURL)
		if err != nil {
			return "", "", "", err
		}
	}
	if t == PimTypeAll || t == PimTypeRole || t == PimTypeGroup {
		rbacToken, err = pim.GetAccessToken(pim.RBACTokenScope)
		if err != nil {
			return "", "", "", err
		}
	}
	// Subject (oid) — read from whichever token we have. Both ARM and RBAC
	// tokens carry the same oid claim.
	tokenForOID := armToken
	if tokenForOID == "" {
		tokenForOID = rbacToken
	}
	info, err := pim.GetUserInfo(tokenForOID)
	if err != nil {
		return "", "", "", fmt.Errorf("parse subject from access token: %w", err)
	}
	return armToken, rbacToken, info.ObjectID, nil
}

func pimFetchEligible(ctx context.Context, c *pim.Client, t, subjectID, armToken, rbacToken string) (
	resource []pim.ResourceAssignment,
	entra, group []pim.GovernanceRoleAssignment,
	err error,
) {
	if t == PimTypeAll || t == PimTypeResource {
		resp, err := c.GetEligibleResourceAssignments(ctx, armToken)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("fetch resource eligible: %w", err)
		}
		resource = resp.Value
	}
	if t == PimTypeAll || t == PimTypeRole {
		resp, err := c.GetEligibleGovernanceRoleAssignments(ctx, pim.RoleTypeEntraRoles, subjectID, rbacToken)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("fetch entra role eligible: %w", err)
		}
		entra = resp.Value
	}
	if t == PimTypeAll || t == PimTypeGroup {
		resp, err := c.GetEligibleGovernanceRoleAssignments(ctx, pim.RoleTypeAADGroups, subjectID, rbacToken)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("fetch group eligible: %w", err)
		}
		group = resp.Value
	}
	return resource, entra, group, nil
}

func pimFetchActive(ctx context.Context, c *pim.Client, t, subjectID, armToken, rbacToken string) (
	resource []pim.ActiveResourceAssignment,
	entra, group []pim.GovernanceRoleAssignment,
	err error,
) {
	if t == PimTypeAll || t == PimTypeResource {
		resp, err := c.GetActiveResourceAssignments(ctx, armToken)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("fetch resource active: %w", err)
		}
		resource = resp.Value
	}
	if t == PimTypeAll || t == PimTypeRole {
		resp, err := c.GetActiveGovernanceRoleAssignments(ctx, pim.RoleTypeEntraRoles, subjectID, rbacToken)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("fetch entra role active: %w", err)
		}
		entra = resp.Value
	}
	if t == PimTypeAll || t == PimTypeGroup {
		resp, err := c.GetActiveGovernanceRoleAssignments(ctx, pim.RoleTypeAADGroups, subjectID, rbacToken)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("fetch group active: %w", err)
		}
		group = resp.Value
	}
	return resource, entra, group, nil
}

// ── Match resolution ────────────────────────────────────────────────

// activateTarget describes a resolved match ready to send.
type activateTarget struct {
	kind     string                        // "resource", "Entra role", "AAD group"
	resource *pim.ResourceAssignment       // set when kind == "resource"
	gov      *pim.GovernanceRoleAssignment // set when kind != "resource"
	roleType string                        // pim.RoleTypeEntraRoles / RoleTypeAADGroups (when gov)
}

func (t *activateTarget) activate(ctx context.Context, c *pim.Client, subjectID, armToken, rbacToken string, opts ActivateOptions) error {
	if t.kind == "resource" {
		scope, body, err := pim.NewResourceAssignmentRequest(
			subjectID, t.resource, opts.DurationMin, opts.StartDate, opts.StartTime,
			opts.Reason, opts.TicketSystem, opts.TicketNumber,
		)
		if err != nil {
			return err
		}
		resp, err := c.RequestResourceAssignment(ctx, scope, body, armToken)
		if err != nil {
			return err
		}
		if pim.IsResourceRequestFailed(resp) {
			return fmt.Errorf("status=%s", resp.Properties.Status)
		}
		if opts.Wait && !pim.IsResourceRequestOK(resp) {
			return c.WaitForResourceAssignment(ctx, scope, resp.Name, armToken, opts.WaitTimeout)
		}
		return nil
	}
	body, err := pim.NewGovernanceAssignmentRequest(
		subjectID, t.roleType, t.gov, opts.DurationMin, opts.StartDate, opts.StartTime,
		opts.Reason, opts.TicketSystem, opts.TicketNumber,
	)
	if err != nil {
		return err
	}
	resp, err := c.RequestGovernanceRoleAssignment(ctx, t.roleType, body, rbacToken)
	if err != nil {
		return err
	}
	if pim.IsGovernanceRequestFailed(resp) {
		sub := ""
		if resp.Status != nil {
			sub = resp.Status.SubStatus
		}
		return fmt.Errorf("substatus=%s", sub)
	}
	if opts.Wait && !pim.IsGovernanceRequestOK(resp) {
		return c.WaitForGovernanceRoleAssignment(ctx, t.roleType, resp.ID, rbacToken, opts.WaitTimeout)
	}
	return nil
}

type deactivateTarget struct {
	kind     string
	resource *pim.ActiveResourceAssignment
	gov      *pim.GovernanceRoleAssignment
	roleType string
}

func (t *deactivateTarget) deactivate(ctx context.Context, c *pim.Client, subjectID, armToken, rbacToken string) error {
	if t.kind == "resource" {
		scope, body, err := pim.NewResourceDeactivationRequest(subjectID, t.resource)
		if err != nil {
			return err
		}
		resp, err := c.RequestResourceAssignment(ctx, scope, body, armToken)
		if err != nil {
			return err
		}
		if pim.IsResourceRequestFailed(resp) {
			return fmt.Errorf("status=%s", resp.Properties.Status)
		}
		return nil
	}
	body := pim.NewGovernanceDeactivationRequest(subjectID, t.gov)
	resp, err := c.RequestGovernanceRoleAssignment(ctx, t.roleType, body, rbacToken)
	if err != nil {
		return err
	}
	if pim.IsGovernanceRequestFailed(resp) {
		sub := ""
		if resp.Status != nil {
			sub = resp.Status.SubStatus
		}
		return fmt.Errorf("substatus=%s", sub)
	}
	return nil
}

func resolveActivate(name, roleFilter string,
	resource []pim.ResourceAssignment, entra, group []pim.GovernanceRoleAssignment,
) (*activateTarget, error) {
	var resMatches []pim.ResourceAssignment
	for i := range resource {
		if strings.EqualFold(eligibleResourceName(&resource[i]), name) {
			if roleFilter == "" || strings.EqualFold(resourceRoleName(&resource[i]), roleFilter) {
				resMatches = append(resMatches, resource[i])
			}
		}
	}
	var entraMatches, groupMatches []pim.GovernanceRoleAssignment
	for i := range entra {
		if strings.EqualFold(govName(&entra[i]), name) {
			if roleFilter == "" || strings.EqualFold(govRoleName(&entra[i]), roleFilter) {
				entraMatches = append(entraMatches, entra[i])
			}
		}
	}
	for i := range group {
		if strings.EqualFold(govName(&group[i]), name) {
			if roleFilter == "" || strings.EqualFold(govRoleName(&group[i]), roleFilter) {
				groupMatches = append(groupMatches, group[i])
			}
		}
	}

	categories := 0
	if len(resMatches) > 0 {
		categories++
	}
	if len(entraMatches) > 0 {
		categories++
	}
	if len(groupMatches) > 0 {
		categories++
	}
	if categories == 0 {
		return nil, fmt.Errorf("no eligible assignment matches %q.\n%s", name, suggestEligibleNames(name, resource, entra, group))
	}
	if categories > 1 {
		return nil, errors.New("ambiguous — matches in multiple categories; pass --type=resource|role|group")
	}
	switch {
	case len(resMatches) > 0:
		if len(resMatches) > 1 {
			return nil, fmt.Errorf("ambiguous — %d resource roles match; pass --role to disambiguate: %s",
				len(resMatches), describeResourceRoles(resMatches))
		}
		return &activateTarget{kind: "resource", resource: &resMatches[0]}, nil
	case len(entraMatches) > 0:
		if len(entraMatches) > 1 {
			return nil, fmt.Errorf("ambiguous — %d Entra roles match; pass --role to disambiguate: %s",
				len(entraMatches), describeGovRoles(entraMatches))
		}
		return &activateTarget{kind: "Entra role", gov: &entraMatches[0], roleType: pim.RoleTypeEntraRoles}, nil
	default:
		if len(groupMatches) > 1 {
			return nil, fmt.Errorf("ambiguous — %d AAD group roles match; pass --role to disambiguate: %s",
				len(groupMatches), describeGovRoles(groupMatches))
		}
		return &activateTarget{kind: "AAD group", gov: &groupMatches[0], roleType: pim.RoleTypeAADGroups}, nil
	}
}

func resolveDeactivate(name, roleFilter string,
	resource []pim.ActiveResourceAssignment, entra, group []pim.GovernanceRoleAssignment,
) (*deactivateTarget, error) {
	var resMatches []pim.ActiveResourceAssignment
	for i := range resource {
		if strings.EqualFold(activeResourceName(&resource[i]), name) {
			if roleFilter == "" || strings.EqualFold(activeResourceRoleName(&resource[i]), roleFilter) {
				resMatches = append(resMatches, resource[i])
			}
		}
	}
	var entraMatches, groupMatches []pim.GovernanceRoleAssignment
	for i := range entra {
		if strings.EqualFold(govName(&entra[i]), name) {
			if roleFilter == "" || strings.EqualFold(govRoleName(&entra[i]), roleFilter) {
				entraMatches = append(entraMatches, entra[i])
			}
		}
	}
	for i := range group {
		if strings.EqualFold(govName(&group[i]), name) {
			if roleFilter == "" || strings.EqualFold(govRoleName(&group[i]), roleFilter) {
				groupMatches = append(groupMatches, group[i])
			}
		}
	}
	categories := 0
	if len(resMatches) > 0 {
		categories++
	}
	if len(entraMatches) > 0 {
		categories++
	}
	if len(groupMatches) > 0 {
		categories++
	}
	if categories == 0 {
		return nil, fmt.Errorf("no active assignment matches %q.\n%s", name, suggestActiveNames(name, resource, entra, group))
	}
	if categories > 1 {
		return nil, errors.New("ambiguous — matches in multiple categories; pass --type")
	}
	switch {
	case len(resMatches) > 0:
		if len(resMatches) > 1 {
			return nil, errors.New("ambiguous — multiple active resource roles match; pass --role")
		}
		return &deactivateTarget{kind: "resource", resource: &resMatches[0]}, nil
	case len(entraMatches) > 0:
		if len(entraMatches) > 1 {
			return nil, errors.New("ambiguous — multiple active Entra roles match; pass --role")
		}
		return &deactivateTarget{kind: "Entra role", gov: &entraMatches[0], roleType: pim.RoleTypeEntraRoles}, nil
	default:
		if len(groupMatches) > 1 {
			return nil, errors.New("ambiguous — multiple active AAD group roles match; pass --role")
		}
		return &deactivateTarget{kind: "AAD group", gov: &groupMatches[0], roleType: pim.RoleTypeAADGroups}, nil
	}
}

// ── Field accessors that tolerate missing nested structs ────────────

func eligibleResourceName(r *pim.ResourceAssignment) string {
	if r.Properties == nil || r.Properties.ExpandedProperties == nil ||
		r.Properties.ExpandedProperties.Scope == nil {
		return ""
	}
	return r.Properties.ExpandedProperties.Scope.DisplayName
}

func activeResourceName(r *pim.ActiveResourceAssignment) string {
	if r.Properties == nil || r.Properties.ExpandedProperties == nil ||
		r.Properties.ExpandedProperties.Scope == nil {
		return ""
	}
	return r.Properties.ExpandedProperties.Scope.DisplayName
}

func resourceRoleName(r *pim.ResourceAssignment) string {
	if r.Properties == nil || r.Properties.ExpandedProperties == nil ||
		r.Properties.ExpandedProperties.RoleDefinition == nil {
		return ""
	}
	return r.Properties.ExpandedProperties.RoleDefinition.DisplayName
}

func activeResourceRoleName(r *pim.ActiveResourceAssignment) string {
	if r.Properties == nil || r.Properties.ExpandedProperties == nil ||
		r.Properties.ExpandedProperties.RoleDefinition == nil {
		return ""
	}
	return r.Properties.ExpandedProperties.RoleDefinition.DisplayName
}

func govName(g *pim.GovernanceRoleAssignment) string {
	if g.RoleDefinition == nil || g.RoleDefinition.Resource == nil {
		return ""
	}
	return g.RoleDefinition.Resource.DisplayName
}

func govRoleName(g *pim.GovernanceRoleAssignment) string {
	if g.RoleDefinition == nil {
		return ""
	}
	return g.RoleDefinition.DisplayName
}

// suggestEligibleNames returns a short hint listing the eligible names that
// are close to query (case-insensitive substring) — or the full sorted list
// if nothing's close. Helps users realize they typed the wrong case or a
// near-but-not-equal name without forcing them to run `pim list` separately.
func suggestEligibleNames(query string,
	resource []pim.ResourceAssignment, entra, group []pim.GovernanceRoleAssignment,
) string {
	names := []string{}
	for i := range resource {
		if n := eligibleResourceName(&resource[i]); n != "" {
			names = append(names, n)
		}
	}
	for i := range entra {
		if n := govName(&entra[i]); n != "" {
			names = append(names, n)
		}
	}
	for i := range group {
		if n := govName(&group[i]); n != "" {
			names = append(names, n)
		}
	}
	return formatNameSuggestion(query, names, "eligible")
}

func suggestActiveNames(query string,
	resource []pim.ActiveResourceAssignment, entra, group []pim.GovernanceRoleAssignment,
) string {
	names := []string{}
	for i := range resource {
		if n := activeResourceName(&resource[i]); n != "" {
			names = append(names, n)
		}
	}
	for i := range entra {
		if n := govName(&entra[i]); n != "" {
			names = append(names, n)
		}
	}
	for i := range group {
		if n := govName(&group[i]); n != "" {
			names = append(names, n)
		}
	}
	return formatNameSuggestion(query, names, "active")
}

func formatNameSuggestion(query string, names []string, label string) string {
	uniq := map[string]struct{}{}
	for _, n := range names {
		uniq[n] = struct{}{}
	}
	flat := make([]string, 0, len(uniq))
	for n := range uniq {
		flat = append(flat, n)
	}
	sort.Strings(flat)
	if len(flat) == 0 {
		return fmt.Sprintf("  (no %s assignments)", label)
	}

	q := strings.ToLower(query)
	var close []string
	for _, n := range flat {
		if strings.Contains(strings.ToLower(n), q) {
			close = append(close, n)
		}
	}
	if len(close) > 0 {
		return "  did you mean: " + strings.Join(close, ", ")
	}
	if len(flat) > 10 {
		return fmt.Sprintf("  %s names: %s, ... (%d total)", label, strings.Join(flat[:10], ", "), len(flat))
	}
	return fmt.Sprintf("  %s names: %s", label, strings.Join(flat, ", "))
}

func describeResourceRoles(rs []pim.ResourceAssignment) string {
	roles := make([]string, 0, len(rs))
	for i := range rs {
		roles = append(roles, resourceRoleName(&rs[i]))
	}
	sort.Strings(roles)
	return strings.Join(roles, ", ")
}

func describeGovRoles(gs []pim.GovernanceRoleAssignment) string {
	roles := make([]string, 0, len(gs))
	for i := range gs {
		roles = append(roles, govRoleName(&gs[i]))
	}
	sort.Strings(roles)
	return strings.Join(roles, ", ")
}

// ── Rendering ───────────────────────────────────────────────────────

func printResourceEligible(rs []pim.ResourceAssignment) {
	fmt.Printf("\n%s%sEligible — Azure resources%s\n", ui.Bold, ui.Blue, ui.NC)
	if len(rs) == 0 {
		fmt.Printf("  %s(none)%s\n", ui.Dim, ui.NC)
		return
	}
	tw := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "  NAME\tROLE\tSCOPE TYPE")
	sort.Slice(rs, func(i, j int) bool { return eligibleResourceName(&rs[i]) < eligibleResourceName(&rs[j]) })
	for i := range rs {
		scopeType := ""
		if rs[i].Properties != nil && rs[i].Properties.ExpandedProperties != nil &&
			rs[i].Properties.ExpandedProperties.Scope != nil {
			scopeType = rs[i].Properties.ExpandedProperties.Scope.Type
		}
		fmt.Fprintf(tw, "  %s\t%s\t%s\n", eligibleResourceName(&rs[i]), resourceRoleName(&rs[i]), scopeType)
	}
	tw.Flush()
}

func printResourceActive(rs []pim.ActiveResourceAssignment) {
	fmt.Printf("\n%s%sActive — Azure resources%s\n", ui.Bold, ui.Blue, ui.NC)
	if len(rs) == 0 {
		fmt.Printf("  %s(none)%s\n", ui.Dim, ui.NC)
		return
	}
	tw := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "  NAME\tROLE\tEXPIRES")
	sort.Slice(rs, func(i, j int) bool { return activeResourceName(&rs[i]) < activeResourceName(&rs[j]) })
	for i := range rs {
		end := ""
		if rs[i].Properties != nil {
			end = rs[i].Properties.EndDateTime
		}
		fmt.Fprintf(tw, "  %s\t%s\t%s\n", activeResourceName(&rs[i]), activeResourceRoleName(&rs[i]), end)
	}
	tw.Flush()
}

func printGovernanceEligible(label string, gs []pim.GovernanceRoleAssignment) {
	fmt.Printf("\n%s%sEligible — %s%s\n", ui.Bold, ui.Blue, label, ui.NC)
	if len(gs) == 0 {
		fmt.Printf("  %s(none)%s\n", ui.Dim, ui.NC)
		return
	}
	tw := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "  NAME\tROLE")
	sort.Slice(gs, func(i, j int) bool { return govName(&gs[i]) < govName(&gs[j]) })
	for i := range gs {
		fmt.Fprintf(tw, "  %s\t%s\n", govName(&gs[i]), govRoleName(&gs[i]))
	}
	tw.Flush()
}

func printGovernanceActive(label string, gs []pim.GovernanceRoleAssignment) {
	fmt.Printf("\n%s%sActive — %s%s\n", ui.Bold, ui.Blue, label, ui.NC)
	if len(gs) == 0 {
		fmt.Printf("  %s(none)%s\n", ui.Dim, ui.NC)
		return
	}
	tw := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "  NAME\tROLE\tEXPIRES")
	sort.Slice(gs, func(i, j int) bool { return govName(&gs[i]) < govName(&gs[j]) })
	for i := range gs {
		fmt.Fprintf(tw, "  %s\t%s\t%s\n", govName(&gs[i]), govRoleName(&gs[i]), gs[i].EndDateTime)
	}
	tw.Flush()
}
