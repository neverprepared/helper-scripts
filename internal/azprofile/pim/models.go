// Ported from github.com/neverprepared/az-pim-cli/pkg/pim/models.go.
// Original: Copyright © 2023 netr0m <netr0m@pm.me>. Reorganized for
// azprofile: jwt-claims struct removed (we parse JWT manually in token.go).
package pim

// PIMRequest is the internal request descriptor used by Request().
type PIMRequest struct {
	URL     string
	Token   string
	Method  string
	Headers map[string][]string
	Payload any
	Params  map[string]string
}

// AzureUserInfo captures the JWT claims we care about for PIM activation.
type AzureUserInfo struct {
	ObjectID string `json:"oid"`
	Email    string `json:"unique_name"`
}

// ── Resource (Azure resource) role assignments ───────────────────────

type ResourceExpandedProperty struct {
	ID          string `json:"id"`
	DisplayName string `json:"displayName"`
	Type        string `json:"type"`
	Email       string `json:"email"`
}

type ResourceExpandedProperties struct {
	Principal      *ResourceExpandedProperty `json:"principal"`
	RoleDefinition *ResourceExpandedProperty `json:"roleDefinition"`
	Scope          *ResourceExpandedProperty `json:"scope"`
}

type ResourceProperties struct {
	RoleEligibilityScheduleID string                      `json:"roleEligibilityScheduleId"`
	Scope                     string                      `json:"scope"`
	RoleDefinitionID          string                      `json:"roleDefinitionId"`
	PrincipalID               string                      `json:"principalId"`
	PrincipalType             string                      `json:"principalType"`
	Status                    string                      `json:"status"`
	StartDateTime             string                      `json:"startDateTime"`
	EndDateTime               string                      `json:"endDateTime"`
	MemberType                string                      `json:"memberType"`
	CreatedOn                 string                      `json:"createdOn"`
	Condition                 string                      `json:"condition"`
	ConditionVersion          string                      `json:"conditionVersion"`
	ExpandedProperties        *ResourceExpandedProperties `json:"expandedProperties"`
}

type ResourceAssignment struct {
	Properties *ResourceProperties `json:"properties"`
	Name       string              `json:"name"`
	ID         string              `json:"id"`
	Type       string              `json:"type"`
}

type ResourceAssignmentResponse struct {
	Value []ResourceAssignment `json:"value"`
}

type ActiveResourceProperties struct {
	RoleAssignmentScheduleID        string                      `json:"roleAssignmentScheduleId"`
	LinkedRoleEligibilityScheduleID string                      `json:"linkedRoleEligibilityScheduleId"`
	Scope                           string                      `json:"scope"`
	RoleDefinitionID                string                      `json:"roleDefinitionId"`
	PrincipalID                     string                      `json:"principalId"`
	PrincipalType                   string                      `json:"principalType"`
	Status                          string                      `json:"status"`
	StartDateTime                   string                      `json:"startDateTime"`
	EndDateTime                     string                      `json:"endDateTime"`
	AssignmentType                  string                      `json:"assignmentType"`
	MemberType                      string                      `json:"memberType"`
	CreatedOn                       string                      `json:"createdOn"`
	ExpandedProperties              *ResourceExpandedProperties `json:"expandedProperties"`
}

type ActiveResourceAssignment struct {
	Properties *ActiveResourceProperties `json:"properties"`
	Name       string                    `json:"name"`
	ID         string                    `json:"id"`
	Type       string                    `json:"type"`
}

type ActiveResourceAssignmentResponse struct {
	Value []ActiveResourceAssignment `json:"value"`
}

type TicketInfo struct {
	TicketNumber string `json:"ticketNumber"`
	TicketSystem string `json:"ticketSystem"`
}

type ScheduleInfoExpiration struct {
	Type     string `json:"type"`
	Duration string `json:"duration"`
}

type ScheduleInfo struct {
	StartDateTime any                     `json:"startDateTime"`
	Expiration    *ScheduleInfoExpiration `json:"expiration"`
	EndDateTime   any                     `json:"endDateTime"`
}

type ResourceAssignmentValidationProperties struct {
	LinkedRoleEligibilityScheduleID string                      `json:"linkedRoleEligibilityScheduleId"`
	TargetRoleAssignmentScheduleID  string                      `json:"targetRoleAssignmentScheduleId"`
	Scope                           string                      `json:"scope"`
	RoleDefinitionID                string                      `json:"roleDefinitionId"`
	PrincipalID                     string                      `json:"principalId"`
	PrincipalType                   string                      `json:"principalType"`
	RequestType                     string                      `json:"requestType"`
	Status                          string                      `json:"status"`
	ScheduleInfo                    *ScheduleInfo               `json:"scheduleInfo"`
	TicketInfo                      *TicketInfo                 `json:"ticketInfo"`
	Justification                   string                      `json:"justification"`
	RequestorID                     string                      `json:"requestorId"`
	CreatedOn                       string                      `json:"createdOn"`
	ExpandedProperties              *ResourceExpandedProperties `json:"expandedProperties"`
}

type ResourceAssignmentRequestResponse struct {
	Properties *ResourceAssignmentValidationProperties `json:"properties"`
	Name       string                                  `json:"name"`
	ID         string                                  `json:"id"`
	Type       string                                  `json:"type"`
}

type ResourceAssignmentRequestProperties struct {
	PrincipalID                     string        `json:"PrincipalId"`
	RoleDefinitionID                string        `json:"RoleDefinitionId"`
	RequestType                     string        `json:"RequestType"`
	LinkedRoleEligibilityScheduleID string        `json:"LinkedRoleEligibilityScheduleId"`
	Justification                   string        `json:"Justification"`
	ScheduleInfo                    *ScheduleInfo `json:"ScheduleInfo"`
	TicketInfo                      *TicketInfo   `json:"TicketInfo"`
	IsValidationOnly                bool          `json:"IsValidationOnly"`
	// IsActivativation matches the Azure API's typo; do not rename.
	IsActivativation bool `json:"IsActivativation"`
}

type ResourceAssignmentRequestRequest struct {
	Properties ResourceAssignmentRequestProperties `json:"Properties"`
}

// ── Governance (Entra roles + AAD groups) role assignments ───────────

type GovernanceRoleAssignmentSubject struct {
	ID            string `json:"id"`
	Type          string `json:"type"`
	DisplayName   string `json:"displayName"`
	PrincipalName string `json:"principalName"`
	Email         string `json:"email"`
}

type GovernanceRoleResource struct {
	ID          string `json:"id"`
	Type        string `json:"type"`
	DisplayName string `json:"displayName"`
	Status      string `json:"status"`
}

type GovernanceRoleDefinition struct {
	ID          string                  `json:"id"`
	ResourceID  string                  `json:"resourceId"`
	Type        string                  `json:"type"`
	DisplayName string                  `json:"displayName"`
	Resource    *GovernanceRoleResource `json:"resource"`
}

type GovernanceRoleAssignment struct {
	ID               string                           `json:"id"`
	ResourceID       string                           `json:"resourceId"`
	RoleDefinitionID string                           `json:"roleDefinitionId"`
	SubjectID        string                           `json:"subjectId"`
	AssignmentState  string                           `json:"assignmentState"`
	EndDateTime      string                           `json:"endDateTime,omitempty"`
	Status           string                           `json:"status"`
	Subject          *GovernanceRoleAssignmentSubject `json:"subject"`
	RoleDefinition   *GovernanceRoleDefinition        `json:"roleDefinition"`
}

type GovernanceRoleAssignmentResponse struct {
	Value []GovernanceRoleAssignment `json:"value"`
}

type GovernanceRoleAssignmentSchedule struct {
	Type          string `json:"type"`
	StartDateTime any    `json:"startDateTime"`
	EndDateTime   any    `json:"endDateTime"`
	Duration      string `json:"duration"`
}

type GovernanceRoleAssignmentRequest struct {
	RoleDefinitionID               string                            `json:"roleDefinitionId"`
	ResourceID                     string                            `json:"resourceId"`
	SubjectID                      string                            `json:"subjectId"`
	AssignmentState                string                            `json:"assignmentState"`
	Type                           string                            `json:"type"`
	Reason                         string                            `json:"reason"`
	TicketNumber                   string                            `json:"ticketNumber"`
	TicketSystem                   string                            `json:"ticketSystem"`
	Schedule                       *GovernanceRoleAssignmentSchedule `json:"schedule"`
	LinkedEligibleRoleAssignmentID string                            `json:"linkedEligibleRoleAssignmentId"`
	ScopedResourceID               string                            `json:"scopedResourceId"`
}

type GovernanceRoleAssignmentRequestStatus struct {
	Status        string              `json:"status"`
	SubStatus     string              `json:"subStatus"`
	StatusDetails []map[string]string `json:"statusDetails"`
}

type GovernanceRoleAssignmentRequestResponse struct {
	ID                             string                                 `json:"id"`
	ResourceID                     string                                 `json:"resourceId"`
	RoleDefinitionID               string                                 `json:"roleDefinitionId"`
	SubjectID                      string                                 `json:"subjectId"`
	ScopedResourceID               string                                 `json:"scopedResourceId"`
	LinkedEligibleRoleAssignmentID string                                 `json:"linkedEligibleRoleAssignmentId"`
	Type                           string                                 `json:"type"`
	AssignmentState                string                                 `json:"assignmentState"`
	RequestedDateTime              string                                 `json:"requestedDateTime"`
	RoleAssignmentStartDateTime    string                                 `json:"roleAssignmentStartDateTime"`
	RoleAssignmentEndDateTime      string                                 `json:"roleAssignmentEndDateTime"`
	Reason                         string                                 `json:"reason"`
	TicketNumber                   string                                 `json:"ticketNumber"`
	TicketSystem                   string                                 `json:"ticketSystem"`
	Condition                      string                                 `json:"condition"`
	ConditionVersion               string                                 `json:"conditionVersion"`
	ConditionDescription           string                                 `json:"conditionDescription"`
	Status                         *GovernanceRoleAssignmentRequestStatus `json:"status"`
	Schedule                       *GovernanceRoleAssignmentSchedule      `json:"schedule"`
	Metadata                       map[string]any                         `json:"metadata"`
}
