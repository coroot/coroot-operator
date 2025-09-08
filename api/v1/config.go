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
	// Default role for authenticated users (Admin, Editor, Viewer, or a custom role).
	DefaultRole string `json:"defaultRole,omitempty"`
	// SAML configuration.
	SAML *SSOSAMLSpec `json:"saml,omitempty"`
}

type SSOSAMLSpec struct {
	// Identity Provider Metadata XML.
	Metadata string `json:"metadata,omitempty"`
	// Secret containing the Metadata XML.
	MetadataSecret *corev1.SecretKeySelector `json:"metadataSecret,omitempty"`
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

type ProjectSpec struct {
	// Project name (e.g., production, staging; required).
	// +kubebuilder:validation:Required
	Name string `json:"name,omitempty"`
	// Project API keys, used by agents to send telemetry data (required).
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinItems=1
	ApiKeys []ApiKeySpec `json:"apiKeys,omitempty"`
	// Notification integrations.
	NotificationIntegrations *NotificationIntegrationsSpec `json:"notificationIntegrations,omitempty"`
	// Application category settings.
	ApplicationCategories []ApplicationCategorySpec `json:"applicationCategories,omitempty"`
	// Custom applications.
	CustomApplications []CustomApplicationSpec `json:"customApplications,omitempty"`
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
}

type NotificationIntegrationPagerdutySpec struct {
	// PagerDuty Integration Key.
	IntegrationKey string `json:"integrationKey,omitempty"`
	// Secret containing the Integration Key.
	IntegrationKeySecret *corev1.SecretKeySelector `json:"integrationKeySecret,omitempty"`
	// Notify of incidents (SLO violation).
	Incidents bool `json:"incidents,omitempty"`
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
	// Incident template (required if `incidents: true`).
	IncidentTemplate string `json:"incidentTemplate,omitempty"`
	// Deployment template (required if `deployments: true`).
	DeploymentTemplate string `json:"deploymentTemplate,omitempty"`
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
