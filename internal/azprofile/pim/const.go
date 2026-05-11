// Package pim ports the Azure PIM REST client from
// github.com/neverprepared/az-pim-cli (Copyright © 2023 netr0m) into
// the azprofile binary. Auth is delegated to the user's `az` CLI via
// `az account get-access-token` rather than azidentity, so this package
// does not pull in the Azure SDK for Go.
package pim

const (
	ARMGlobalBaseURL = "https://management.azure.com"
	ARMUSGovBaseURL  = "https://management.usgovcloudapi.net"
	ARMChinaBaseURL  = "https://management.chinacloudapi.cn"
)

// ARMBaseURLs maps the Azure cloud name to its ARM endpoint.
var ARMBaseURLs = map[string]string{
	"global": ARMGlobalBaseURL,
	"usgov":  ARMUSGovBaseURL,
	"china":  ARMChinaBaseURL,
}

const (
	ARMBasePath = "providers/Microsoft.Authorization"

	RBACBaseURL  = "https://api.azrbac.mspim.azure.com"
	RBACBasePath = "api/v2/privilegedAccess"
	// RBACTokenScope is what `az account get-access-token --resource <scope>`
	// expects when fetching a token usable against the RBAC PIM endpoints.
	RBACTokenScope = RBACBaseURL

	PIMAPIVersion = "2020-10-01"

	DefaultReason             = "config"
	DefaultDurationMinutes    = 480
	DefaultWaitTimeoutSeconds = 300
	WaitPollIntervalSeconds   = 5
)

// Role types for the Governance Role API.
const (
	RoleTypeAADGroups  = "aadGroups"
	RoleTypeEntraRoles = "aadroles"
)

// PIM assignment request status values surfaced by the API.
const (
	StatusAccepted                    = "Accepted"
	StatusAdminApproved               = "AdminApproved"
	StatusAdminDenied                 = "AdminDenied"
	StatusCanceled                    = "Canceled"
	StatusDenied                      = "Denied"
	StatusFailed                      = "Failed"
	StatusFailedAsResourceIsLocked    = "FailedAsResourceIsLocked"
	StatusGranted                     = "Granted"
	StatusInvalid                     = "Invalid"
	StatusPendingAdminDecision        = "PendingAdminDecision"
	StatusPendingApproval             = "PendingApproval"
	StatusPendingApprovalProvisioning = "PendingApprovalProvisioning"
	StatusPendingEvaluation           = "PendingEvaluation"
	StatusPendingExternalProvisioning = "PendingExternalProvisioning"
	StatusPendingProvisioning         = "PendingProvisioning"
	StatusPendingRevocation           = "PendingRevocation"
	StatusPendingScheduleCreation     = "PendingScheduleCreation"
	StatusProvisioned                 = "Provisioned"
	StatusProvisioningStarted         = "ProvisioningStarted"
	StatusRevoked                     = "Revoked"
	StatusScheduleCreated             = "ScheduleCreated"
	StatusTimedOut                    = "TimedOut"
)
