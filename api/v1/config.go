package v1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type CorootCloudSpec struct {
	// Coroot Cloud API key. Can be obtained from the UI after connecting to Coroot Cloud.
	APIKey string `json:"apiKey,omitempty"`
	// Secret containing the API key.
	APIKeySecret *corev1.SecretKeySelector `json:"apiKeySecret,omitempty"`
	// Root Cause Analysis (RCA) configuration.
	RCA *CorootCloudRCASpec `json:"rca,omitempty"`
}

type CorootCloudRCASpec struct {
	// If true, incidents will not be investigated automatically.
	DisableIncidentsAutoInvestigation bool `json:"disableIncidentsAutoInvestigation,omitempty"`
}

type SSOSpec struct {
	Enabled bool `json:"enabled,omitempty"`
	// Provider is set automatically based on which configuration section (saml or oidc) is defined.
	Provider string `json:"provider,omitempty"`
	// Default role for authenticated users (Admin, Editor, Viewer, or a custom role).
	DefaultRole string `json:"defaultRole,omitempty"`
	// SAML configuration. Define this section to use SAML SSO.
	SAML *SSOSAMLSpec `json:"saml,omitempty"`
	// OIDC configuration. Define this section to use OIDC SSO.
	OIDC *SSOOIDCSpec `json:"oidc,omitempty"`
}

type SSOSAMLSpec struct {
	// Identity Provider Metadata XML.
	Metadata string `json:"metadata,omitempty"`
	// Secret containing the Metadata XML.
	MetadataSecret *corev1.SecretKeySelector `json:"metadataSecret,omitempty"`
}

type SSOOIDCSpec struct {
	// OIDC provider issuer URL (e.g., https://accounts.google.com).
	// +kubebuilder:validation:Pattern="^https?://.+$"
	IssuerURL string `json:"issuerURL,omitempty"`
	// OAuth client ID.
	ClientID string `json:"clientID,omitempty"`
	// OAuth client secret.
	ClientSecret string `json:"clientSecret,omitempty"`
	// Secret containing the client secret.
	ClientSecretSecret *corev1.SecretKeySelector `json:"clientSecretSecret,omitempty"`
}

type AISpec struct {
	// AI model provider (anthropic, openai, or openai_compatible).
	// +kubebuilder:validation:Enum=anthropic;openai;openai_compatible
	Provider string `json:"provider"`
	// Anthropic configuration.
	Anthropic *AnthropicSpec `json:"anthropic,omitempty"`
	// OpenAI configuration.
	OpenAI *OpenAISpec `json:"openai,omitempty"`
	// OpenAI-compatible configuration.
	OpenAICompatible *OpenAICompatibleSpec `json:"openaiCompatible,omitempty"`
}

type AnthropicSpec struct {
	// Anthropic API key.
	APIKey string `json:"apiKey,omitempty"`
	// Secret containing the API key.
	APIKeySecret *corev1.SecretKeySelector `json:"apiKeySecret,omitempty"`
}

type OpenAISpec struct {
	// OpenAI API key.
	APIKey string `json:"apiKey,omitempty"`
	// Secret containing the API key.
	APIKeySecret *corev1.SecretKeySelector `json:"apiKeySecret,omitempty"`
}

type OpenAICompatibleSpec struct {
	// API key.
	APIKey string `json:"apiKey,omitempty"`
	// Secret containing the API key.
	APIKeySecret *corev1.SecretKeySelector `json:"apiKeySecret,omitempty"`
	// Base URL (eg., https://generativelanguage.googleapis.com/v1beta/openai).
	// +kubebuilder:validation:Pattern="^https?://.+$"
	BaseUrl string `json:"baseURL"`
	// Model name (eg., gemini-2.5-pro-preview-06-05).
	Model string `json:"model"`
}

// +kubebuilder:validation:XValidation:rule="(has(self.memberProjects) && size(self.memberProjects) > 0 ? 1 : 0) + (has(self.remoteCoroot) ? 1 : 0) + (has(self.apiKeys) && size(self.apiKeys) > 0 ? 1 : 0) == 1",message="Exactly one of memberProjects, remoteCoroot, or apiKeys must be set."
type ProjectSpec struct {
	// Project name (e.g., production, staging; required).
	// +kubebuilder:validation:Required
	Name string `json:"name,omitempty"`
	// Names of existing projects to aggregate (multi-cluster mode).
	MemberProjects []string `json:"memberProjects,omitempty"`
	// Use another Coroot instance as the data source for this project.
	RemoteCoroot *RemoteCorootSpec `json:"remoteCoroot,omitempty"`
	// Project API keys, used by agents to send telemetry data (required unless memberProjects or remoteCoroot is set).
	ApiKeys []ApiKeySpec `json:"apiKeys,omitempty"`
	// Notification integrations.
	NotificationIntegrations *NotificationIntegrationsSpec `json:"notificationIntegrations,omitempty"`
	// Application category settings.
	ApplicationCategories []ApplicationCategorySpec `json:"applicationCategories,omitempty"`
	// Custom applications.
	CustomApplications []CustomApplicationSpec `json:"customApplications,omitempty"`
	// Alerting rules. Rules defined here are shown with a lock icon in the UI and cannot be edited or deleted through the UI.
	AlertingRules []AlertingRuleSpec `json:"alertingRules,omitempty"`
	// Inspection overrides.
	InspectionOverrides *InspectionOverrides `json:"inspectionOverrides,omitempty"`
}

type ApiKeySpec struct {
	// Plain-text API key. Must be unique. Prefer using KeySecret for better security.
	Key string `json:"key,omitempty"`
	// Secret with the API key. Created automatically if missing.
	KeySecret *corev1.SecretKeySelector `json:"keySecret,omitempty"`
	// API key description (optional).
	Description string `json:"description,omitempty"`
}

type RemoteCorootSpec struct {
	// Base URL of the remote Coroot instance.
	// +kubebuilder:validation:Pattern="^https?://.+$"
	Url string `json:"url,omitempty"`
	// Whether to skip verification of the Coroot server's TLS certificate.
	TlsSkipVerify bool `json:"tlsSkipVerify,omitempty"`
	// API key of the remote project.
	ApiKey string `json:"apiKey,omitempty"`
	// Secret containing the API key.
	ApiKeySecret *corev1.SecretKeySelector `json:"apiKeySecret,omitempty"`
	// Prometheus query resolution/refresh interval (e.g. 15s).
	// +kubebuilder:validation:Pattern="^[0-9]+[smhdwy]$"
	MetricResolution string `json:"metricResolution,omitempty"`
}

type NotificationIntegrationsSpec struct {
	// The URL of Coroot instance (required). Used for generating links in notifications.
	// +kubebuilder:validation:Pattern="^https?://.+$"
	BaseUrl string `json:"baseURL"`
	// Slack configuration.
	Slack *NotificationIntegrationSlackSpec `json:"slack,omitempty"`
	// Microsoft Teams configuration.
	Teams *NotificationIntegrationTeamsSpec `json:"teams,omitempty"`
	// PagerDuty configuration.
	Pagerduty *NotificationIntegrationPagerdutySpec `json:"pagerduty,omitempty"`
	// Opsgenie configuration.
	Opsgenie *NotificationIntegrationOpsgenieSpec `json:"opsgenie,omitempty"`
	// Webhook configuration.
	Webhook *NotificationIntegrationWebhookSpec `json:"webhook,omitempty"`
}

type NotificationIntegrationSlackSpec struct {
	// Slack Bot User OAuth Token.
	Token string `json:"token,omitempty"`
	// Secret containing the Token.
	TokenSecret *corev1.SecretKeySelector `json:"tokenSecret,omitempty"`
	// Default Slack channel name.
	DefaultChannel string `json:"defaultChannel"`
	// Notify of incidents (SLO violation).
	Incidents bool `json:"incidents,omitempty"`
	// Notify of deployments.
	Deployments bool `json:"deployments,omitempty"`
	// Notify of alerts.
	Alerts *bool `json:"alerts,omitempty"`
}

type NotificationIntegrationTeamsSpec struct {
	// MS Teams Webhook URL.
	WebhookURL string `json:"webhookURL,omitempty"`
	// Secret containing the Webhook URL.
	WebhookURLSecret *corev1.SecretKeySelector `json:"webhookURLSecret,omitempty"`
	// Notify of incidents (SLO violation).
	Incidents bool `json:"incidents,omitempty"`
	// Notify of deployments.
	Deployments bool `json:"deployments,omitempty"`
	// Notify of alerts.
	Alerts *bool `json:"alerts,omitempty"`
}

type NotificationIntegrationPagerdutySpec struct {
	// PagerDuty Integration Key.
	IntegrationKey string `json:"integrationKey,omitempty"`
	// Secret containing the Integration Key.
	IntegrationKeySecret *corev1.SecretKeySelector `json:"integrationKeySecret,omitempty"`
	// Notify of incidents (SLO violation).
	Incidents bool `json:"incidents,omitempty"`
	// Notify of alerts.
	Alerts *bool `json:"alerts,omitempty"`
}

type NotificationIntegrationOpsgenieSpec struct {
	// Opsgenie API Key.
	ApiKey string `json:"apiKey,omitempty"`
	// Secret containing the API key.
	ApiKeySecret *corev1.SecretKeySelector `json:"apiKeySecret,omitempty"`
	// EU instance of Opsgenie.
	EUInstance bool `json:"euInstance,omitempty"`
	// Notify of incidents (SLO violation).
	Incidents bool `json:"incidents,omitempty"`
	// Notify of alerts.
	Alerts *bool `json:"alerts,omitempty"`
}

type NotificationIntegrationWebhookSpec struct {
	// Webhook URL (required).
	// +kubebuilder:validation:Pattern="^https?://.+$"
	Url string `json:"url"`
	// Whether to skip verification of the Webhook server's TLS certificate.
	TlsSkipVerify bool `json:"tlsSkipVerify,omitempty"`
	// Basic auth credentials.
	BasicAuth *BasicAuthSpec `json:"basicAuth,omitempty"`
	// Custom headers to include in requests.
	CustomHeaders []HeaderSpec `json:"customHeaders,omitempty"`
	// Notify of incidents (SLO violation).
	Incidents bool `json:"incidents,omitempty"`
	// Notify of deployments.
	Deployments bool `json:"deployments,omitempty"`
	// Notify of alerts.
	Alerts *bool `json:"alerts,omitempty"`
	// Incident template (required if `incidents: true`).
	IncidentTemplate string `json:"incidentTemplate,omitempty"`
	// Deployment template (required if `deployments: true`).
	DeploymentTemplate string `json:"deploymentTemplate,omitempty"`
	// Alert template (required if `alerts: true`).
	AlertTemplate string `json:"alertTemplate,omitempty"`
}

type ApplicationCategorySpec struct {
	// Application category name (required).
	Name string `json:"name"`
	// List of glob patterns in the <namespace>/<application_name> format (e.g., "staging/*", "*/mongo-*").
	CustomPatterns []string `json:"customPatterns,omitempty"`
	// Application category notification settings.
	NotificationSettings ApplicationCategoryNotificationSettingsSpec `json:"notificationSettings,omitempty"`
}

type ApplicationCategoryNotificationSettingsSpec struct {
	// Notify of incidents (SLO violation).
	Incidents ApplicationCategoryNotificationSettingsIncidentsSpec `json:"incidents,omitempty"`
	// Notify of deployments.
	Deployments ApplicationCategoryNotificationSettingsDeploymentsSpec `json:"deployments,omitempty"`
	// Notify of alerts.
	Alerts ApplicationCategoryNotificationSettingsAlertsSpec `json:"alerts,omitempty"`
}

type ApplicationCategoryNotificationSettingsAlertsSpec struct {
	Enabled   bool                                                  `json:"enabled,omitempty"`
	Slack     *ApplicationCategoryNotificationSettingsSlackSpec     `json:"slack,omitempty"`
	Teams     *ApplicationCategoryNotificationSettingsTeamsSpec     `json:"teams,omitempty"`
	Pagerduty *ApplicationCategoryNotificationSettingsPagerdutySpec `json:"pagerduty,omitempty"`
	Opsgenie  *ApplicationCategoryNotificationSettingsOpsgenieSpec  `json:"opsgenie,omitempty"`
	Webhook   *ApplicationCategoryNotificationSettingsWebhookSpec   `json:"webhook,omitempty"`
}

type ApplicationCategoryNotificationSettingsIncidentsSpec struct {
	Enabled   bool                                                  `json:"enabled,omitempty"`
	Slack     *ApplicationCategoryNotificationSettingsSlackSpec     `json:"slack,omitempty"`
	Teams     *ApplicationCategoryNotificationSettingsTeamsSpec     `json:"teams,omitempty"`
	Pagerduty *ApplicationCategoryNotificationSettingsPagerdutySpec `json:"pagerduty,omitempty"`
	Opsgenie  *ApplicationCategoryNotificationSettingsOpsgenieSpec  `json:"opsgenie,omitempty"`
	Webhook   *ApplicationCategoryNotificationSettingsWebhookSpec   `json:"webhook,omitempty"`
}

type ApplicationCategoryNotificationSettingsDeploymentsSpec struct {
	Enabled bool                                                `json:"enabled,omitempty"`
	Slack   *ApplicationCategoryNotificationSettingsSlackSpec   `json:"slack,omitempty"`
	Teams   *ApplicationCategoryNotificationSettingsTeamsSpec   `json:"teams,omitempty"`
	Webhook *ApplicationCategoryNotificationSettingsWebhookSpec `json:"webhook,omitempty"`
}

type ApplicationCategoryNotificationSettingsSlackSpec struct {
	Enabled bool   `json:"enabled,omitempty"`
	Channel string `json:"channel,omitempty"`
}

type ApplicationCategoryNotificationSettingsTeamsSpec struct {
	Enabled bool `json:"enabled,omitempty"`
}

type ApplicationCategoryNotificationSettingsPagerdutySpec struct {
	Enabled bool `json:"enabled,omitempty"`
}

type ApplicationCategoryNotificationSettingsOpsgenieSpec struct {
	Enabled bool `json:"enabled,omitempty"`
}

type ApplicationCategoryNotificationSettingsWebhookSpec struct {
	Enabled bool `json:"enabled,omitempty"`
}

type AlertingRuleSpec struct {
	// Rule ID (required). For built-in rules, use the existing rule ID (e.g., storage-space, memory-pressure).
	// For custom rules, choose any unique ID. The ID is shown in the rule detail dialog.
	// +kubebuilder:validation:Required
	Id string `json:"id"`
	// Rule name.
	Name *string `json:"name,omitempty"`
	// Alert source configuration.
	Source *AlertSourceSpec `json:"source,omitempty"`
	// Application selector.
	Selector *AppSelectorSpec `json:"selector,omitempty"`
	// Severity level (warning or critical).
	// +kubebuilder:validation:Enum=warning;critical
	Severity *string `json:"severity,omitempty"`
	// How long the condition must be true before firing (e.g., 5m, 1h).
	// +kubebuilder:validation:Pattern="^[0-9]+[smhdwy]$"
	For *string `json:"for,omitempty"`
	// How long to keep firing after condition clears (e.g., 5m).
	// +kubebuilder:validation:Pattern="^[0-9]+[smhdwy]$"
	KeepFiringFor *string `json:"keepFiringFor,omitempty"`
	// Notification templates.
	Templates *AlertTemplatesSpec `json:"templates,omitempty"`
	// Override the notification category for this rule.
	NotificationCategory *string `json:"notificationCategory,omitempty"`
	// Whether the rule is enabled.
	Enabled *bool `json:"enabled,omitempty"`
}

type AlertSourceSpec struct {
	// Source type: check, log_patterns, or promql.
	// +kubebuilder:validation:Enum=check;log_patterns;promql
	Type string `json:"type"`
	// Check source configuration (required if type is check).
	Check *CheckSourceSpec `json:"check,omitempty"`
	// Log pattern source configuration (required if type is log_patterns).
	LogPattern *LogPatternSourceSpec `json:"logPattern,omitempty"`
	// PromQL source configuration (required if type is promql).
	PromQL *PromQLSourceSpec `json:"promql,omitempty"`
}

type CheckSourceSpec struct {
	// The inspection check ID (e.g., cpu-utilization, storage-space).
	CheckId string `json:"checkId"`
}

type LogPatternSourceSpec struct {
	// Log severities to match (e.g., ["error", "fatal"]).
	Severities []string `json:"severities"`
	// Minimum number of occurrences before alerting.
	MinCount *int `json:"minCount,omitempty"`
	// Maximum number of alerts per application for this rule.
	MaxAlertsPerApp *int `json:"maxAlertsPerApp,omitempty"`
	// Use AI to evaluate log patterns and reduce noise.
	EvaluateWithAi *bool `json:"evaluateWithAi,omitempty"`
}

type PromQLSourceSpec struct {
	// PromQL expression that triggers the alert when it returns results.
	Expression string `json:"expression"`
}

type AppSelectorSpec struct {
	// Selector type: all, category, or applications.
	// +kubebuilder:validation:Enum=all;category;applications
	Type string `json:"type"`
	// Application categories to match (when type is category).
	Categories []string `json:"categories,omitempty"`
	// Application ID patterns to match (when type is applications).
	ApplicationIdPatterns []string `json:"applicationIdPatterns,omitempty"`
}

type AlertTemplatesSpec struct {
	// Summary template.
	Summary string `json:"summary,omitempty"`
	// Description template.
	Description string `json:"description,omitempty"`
}

type CustomApplicationSpec struct {
	// Custom application name (required).
	Name string `json:"name"`
	// List of glob patterns for <instance_name>.
	InstancePatterns []string `json:"instancePatterns,omitempty"`
}

type InspectionOverrides struct {
	// SLO Availability overrides.
	SLOAvailability []SLOAvailabilityOverride `json:"sloAvailability,omitempty"`
	// SLO Latency overrides.
	SLOLatency []SLOLatencyOverride `json:"sloLatency,omitempty"`
}

type SLOAvailabilityOverride struct {
	// ApplicationId in the format <namespace>:<kind>:<name> (e.g., default:Deployment:catalog).
	// +kubebuilder:validation:Pattern="^[^:]*:[^:]*:[^:]*"
	ApplicationId string `json:"applicationId"`
	// The percentage of requests that should be served without errors (e.g., 95, 99, 99.9).
	ObjectivePercent Percent `json:"objectivePercent"`
}

type SLOLatencyOverride struct {
	// ApplicationId in the format <namespace>:<kind>:<name> (e.g., default:Deployment:catalog).
	// +kubebuilder:validation:Pattern="^[^:]*:[^:]*:[^:]*"
	ApplicationId string `json:"applicationId"`
	// The percentage of requests that should be served faster than ObjectiveThreshold (e.g., 95, 99, 99.9).
	ObjectivePercent Percent `json:"objectivePercent"`
	// The latency threshold (e.g., 5ms, 10ms, 25ms, 50ms, 100ms, 250ms, 500ms, 1s, 2.5s, 5s, 10s).
	// +kubebuilder:validation:Enum="5ms";"10ms";"25ms";"50ms";"100ms";"250ms";"500ms";"1s";"2.5s";"5s";"10s"
	ObjectiveThreshold metav1.Duration `json:"objectiveThreshold"`
}
