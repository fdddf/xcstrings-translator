package database

import (
	"time"

	"gorm.io/gorm"
)

// User represents a user in the system
type User struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	CreatedAt time.Time      `json:"createdAt"`
	UpdatedAt time.Time      `json:"updatedAt"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deletedAt,omitempty"`

	Username       string `gorm:"uniqueIndex;not null" json:"username"`
	Email          string `gorm:"uniqueIndex;not null" json:"email"`
	Password       string `gorm:"not null" json:"password"` // hashed password
	IsActive       bool   `gorm:"default:true" json:"isActive"`
	IsActivated    bool   `gorm:"default:false" json:"isActivated"`
	ActivationCode string `json:"activationCode,omitempty"`

	// Role and subscription fields
	Role string `gorm:"type:varchar(20);default:'user'" json:"role"` // admin, user

	IsSubscribed     bool       `json:"isSubscribed"`
	SubscriptionType string     `json:"subscriptionType"` // free, basic, premium
	SubscriptionEnd  *time.Time `json:"subscriptionEnd,omitempty"`

	// Usage limits based on subscription
	MaxApps         int `json:"maxApps"`          // Max number of apps allowed
	MaxTranslations int `json:"maxTranslations"`  // Max number of translations per month
	CurrentUsage    int `json:"currentUsage"`     // Current monthly usage
	CurrentAppCount int `json:"currentAppCount"` // Current number of apps
}

// Project represents a project (xcstrings or app group)
type Project struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	CreatedAt time.Time      `json:"createdAt"`
	UpdatedAt time.Time      `json:"updatedAt"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deletedAt,omitempty"`

	Name        string `gorm:"not null" json:"name"`
	Description string `json:"description"`
	UserID      uint   `gorm:"not null" json:"userId"` // Foreign key to User
	User        User   `json:"user"`

	// Project type: xcstrings or app_group
	ProjectType string `gorm:"type:varchar(20);default:'xcstrings'" json:"projectType"`

	// Store the original xcstrings file content
	FileContent string `gorm:"type:text" json:"fileContent"`
	FileName    string `json:"fileName"`

	// Source language for this project
	SourceLanguage string `json:"sourceLanguage"`

	// JSON field to store the parsed content structure
	ContentStructure map[string]interface{} `gorm:"type:jsonb;serializer:json" json:"contentStructure"`

	// Store project settings
	Settings map[string]interface{} `gorm:"type:jsonb;serializer:json" json:"settings"`
}

// Translation represents a translated string
type Translation struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	CreatedAt time.Time      `json:"createdAt"`
	UpdatedAt time.Time      `json:"updatedAt"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deletedAt,omitempty"`

	// Foreign key to Project
	ProjectID uint    `gorm:"not null" json:"projectId"`
	Project   Project `json:"project"`

	// The original key in the xcstrings file
	Key string `gorm:"not null" json:"key"`

	// The source text
	SourceText string `gorm:"not null" json:"sourceText"`

	// The translated text
	TargetText string `json:"targetText"`

	// Target language code (e.g., "zh-Hans", "ja", etc.)
	TargetLanguage string `gorm:"not null" json:"targetLanguage"`

	// State of the translation (e.g., "translated", "needs_review", etc.)
	State string `json:"state"`

	// Optional: translation provider used
	TranslationProvider string `json:"translationProvider"`
}

// UserActivity represents user activity logs
type UserActivity struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	CreatedAt time.Time      `json:"createdAt"`
	UpdatedAt time.Time      `json:"updatedAt"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deletedAt,omitempty"`

	UserID    uint   `gorm:"not null" json:"userId"` // Foreign key to User
	User      User   `json:"user"`
	Action    string `gorm:"not null" json:"action"`
	Details   string `json:"details"`
	IPAddress string `json:"ipAddress"`
	UserAgent string `json:"userAgent"`
}

// ProviderConfig represents user-specific translation provider configurations
type ProviderConfig struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	CreatedAt time.Time      `json:"createdAt"`
	UpdatedAt time.Time      `json:"updatedAt"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deletedAt,omitempty"`

	UserID uint `gorm:"not null" json:"userId"` // Foreign key to User
	User   User `json:"user"`

	// Provider type (e.g., "google", "deepl", "baidu", "openai")
	ProviderType string `gorm:"not null" json:"providerType"`

	// Configuration settings (stored as JSON for flexibility)
	ConfigData map[string]interface{} `gorm:"serializer:json" json:"configData"`

	// Whether this configuration is the default one
	IsDefault bool `json:"isDefault"`
}

// App represents an iOS application in the system
type App struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	CreatedAt time.Time      `json:"createdAt"`
	UpdatedAt time.Time      `json:"updatedAt"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deletedAt,omitempty"`

	Name        string `gorm:"not null" json:"name"`
	Description string `json:"description"`
	UserID      uint   `gorm:"not null" json:"userId"` // Foreign key to User
	User        User   `json:"user"`

	ProjectID *uint    `json:"projectId,omitempty"`
	Project   *Project `json:"project,omitempty"`

	BundleID          string `gorm:"not null;uniqueIndex" json:"bundleId"`           // App's bundle identifier
	AppleID           string `json:"appleId"`                                        // App's Apple ID
	PrimaryLocale     string `json:"primaryLocale"`                                  // Primary language of the app
	Subtitle          string `json:"subtitle"`                                        // App subtitle (primary language)
	AppleConnectToken string `json:"appleConnectToken"`                             // Token for connecting to App Store Connect
	Origin            string `gorm:"type:varchar(20);default:'manual'" json:"origin"` // manual, synced

	// App metadata
	ShortDescription string `json:"shortDescription"` // Short description / promotional text
	LongDescription  string `json:"longDescription"`  // Full app description
	Keywords    string `json:"keywords"` // Comma-separated keywords
	SupportURL  string `json:"supportUrl"`
	MarketingURL string `json:"marketingUrl"`
	PrivacyURL   string `json:"privacyUrl"`

	// App Store status
	Version          string `json:"version"`
	AppCategory      string `json:"app_category"`
	IsReadyForReview bool   `json:"is_ready_for_review"`
}

// AppLocalization represents localization data for an app in App Store Connect
type AppLocalization struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	CreatedAt time.Time      `json:"createdAt"`
	UpdatedAt time.Time      `json:"updatedAt"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deletedAt,omitempty"`

	AppID uint `gorm:"not null" json:"appId"` // Foreign key to App
	App   App  `json:"app"`

	LanguageCode string `gorm:"not null" json:"languageCode"` // Language code (e.g., "en-US", "zh-Hans")

	// App Store Connect localization fields
	Name            string `json:"name"`             // App name in this language
	Subtitle        string `json:"subtitle"`         // App subtitle in this language
	PrivacyURL      string `json:"privacyUrl"`      // Privacy URL in this language
	MarketingURL    string `json:"marketingUrl"`    // Marketing URL in this language
	SupportURL      string `json:"supportUrl"`      // Support URL in this language
	Description     string `json:"description"`     // Description in this language
	Keywords        string `json:"keywords"`         // Keywords in this language (comma-separated)
	WhatsNew        string `json:"whatsNew"`         // What's new in this version
	PromotionalText string `json:"promotionalText"` // Promotional text in this language
	WhatToTest      string `json:"whatToTest"`     // What to test notes for beta testing

	// Sync metadata
	SyncedAt    *time.Time `json:"syncedAt"`
	Source      string     `gorm:"type:varchar(20);default:'local'" json:"source"`
	SyncStatus  string     `gorm:"type:varchar(20);default:'pending'" json:"syncStatus"`
	Version     string     `gorm:"type:varchar(50)" json:"version"`         // App Store version this localization belongs to
	VersionState string    `gorm:"type:varchar(50)" json:"versionState"`   // App Store version state (e.g., READY_FOR_SALE)

	// Additional metadata that might be useful
	Locale string `json:"locale"` // Locale identifier
}

// Subscription represents user subscription information
type Subscription struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	CreatedAt time.Time      `json:"createdAt"`
	UpdatedAt time.Time      `json:"updatedAt"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deletedAt,omitempty"`

	UserID uint `gorm:"not null" json:"userId"` // Foreign key to User
	User   User `json:"user"`

	// Subscription details
	StripeSubscriptionID string     `json:"stripeSubscriptionId"` // Stripe subscription ID
	StripeCustomerID     string     `json:"stripeCustomerId"`     // Stripe customer ID
	SubscriptionType     string     `json:"subscriptionType"`      // free, basic, premium
	SubscriptionStatus   string     `json:"subscriptionStatus"`    // active, canceled, past_due, unpaid
	CurrentPeriodStart   time.Time  `json:"currentPeriodStart"`   // Current billing period start
	CurrentPeriodEnd     time.Time  `json:"currentPeriodEnd"`     // Current billing period end
	TrialEnd             *time.Time `json:"trialEnd,omitempty"`    // When trial period ends (if any)
	CancelAtPeriodEnd    bool       `json:"cancelAtPeriodEnd"`   // Whether to cancel at period end
}

// AppUser represents the many-to-many relationship between users and apps for collaboration
type AppUser struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`

	AppID  uint `gorm:"not null" json:"appId"`
	UserID uint `gorm:"not null" json:"userId"`

	// Role in the app: owner, admin, editor, viewer
	Role string `gorm:"default:'viewer'" json:"role"`

	App  App  `json:"app"`
	User User `json:"user"`
}

// AppProviderConfig represents the binding between an app and a provider configuration
type AppProviderConfig struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	CreatedAt time.Time      `json:"createdAt"`
	UpdatedAt time.Time      `json:"updatedAt"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deletedAt,omitempty"`

	AppID            uint           `gorm:"not null" json:"appId"`
	ProviderConfigID uint           `gorm:"not null" json:"providerConfigId"`
	ProviderType     string         `gorm:"not null" json:"providerType"`
	IsDefault        bool           `json:"isDefault"`
	App              App            `json:"app"`
	ProviderConfig   ProviderConfig `json:"providerConfig"`
}

// TranslationQueue represents items in the translation queue for batch processing
type TranslationQueue struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	CreatedAt time.Time      `json:"createdAt"`
	UpdatedAt time.Time      `json:"updatedAt"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deletedAt,omitempty"`

	// Foreign keys
	UserID    uint  `gorm:"not null" json:"userId"` // User who submitted the job
	ProjectID *uint `json:"projectId,omitempty"`    // Optional: if related to a project
	AppID     *uint `json:"appId,omitempty"`        // Optional: if related to an app

	// Queue job details
	Type     string `gorm:"not null" json:"type"`      // "xcstrings", "app_localization"
	Status   string `gorm:"not null" json:"status"`    // "pending", "processing", "completed", "failed"
	Priority int    `gorm:"default:0" json:"priority"` // Higher number = higher priority
	Progress int    `json:"progress"`                  // Percentage complete
	Total    int    `json:"total"`                     // Total items to process
	Done     int    `json:"done"`                      // Items completed
	Error    string `json:"error,omitempty"`           // Error message if failed

	// Configuration
	ProviderType    string                 `gorm:"serializer:json" json:"providerType"`    // "google", "deepl", etc.
	SourceLanguage  string                 `gorm:"serializer:json" json:"sourceLanguage"`  // Source language code
	TargetLanguages []string               `gorm:"serializer:json" json:"targetLanguages"` // Target language codes
	ConfigData      map[string]interface{} `gorm:"serializer:json" json:"configData"`      // Provider-specific config

	// Result data
	ResultData map[string]interface{} `gorm:"type:jsonb;serializer:json" json:"resultData"` // Result of translation job
}

// SyncHistory represents synchronization history for app localizations
type SyncHistory struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	CreatedAt time.Time      `json:"createdAt"`
	UpdatedAt time.Time      `json:"updatedAt"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`

	UserID    uint `gorm:"not null" json:"userId"` // User who initiated the sync
	AppID     uint `gorm:"not null" json:"appId"`  // App being synced
	ConfigID  uint `gorm:"not null" json:"configId"` // Provider config used

	// Sync details
	Direction     string `gorm:"type:varchar(20);not null" json:"direction"`     // "pull", "push", "both"
	Strategy      string `gorm:"type:varchar(20);default:'apple_first'" json:"strategy"` // "apple_first", "local_first", "manual"
	Status        string `gorm:"type:varchar(20);not null" json:"status"`        // "pending", "running", "completed", "failed"

	// Sync results
	TotalSynced   int `gorm:"default:0" json:"totalSynced"`   // Total number of localizations synced
	TotalFailed   int `gorm:"default:0" json:"totalFailed"`   // Total number of localizations that failed
	TotalConflicts int `gorm:"default:0" json:"totalConflicts"` // Total number of conflicts detected

	// Language codes synced
	LanguageCodes []string `gorm:"serializer:json" json:"languageCodes"` // Specific languages synced

	// Error details
	Error string `gorm:"type:text" json:"error,omitempty"` // Error message if failed

	// Snapshot data (for rollback capability)
	SnapshotData map[string]interface{} `gorm:"type:jsonb;serializer:json" json:"snapshotData,omitempty"` // Before/after snapshot

	// Metadata
	IPAddress string `gorm:"type:varchar(45)" json:"ipAddress,omitempty"` // IP address of the requester
	UserAgent string `gorm:"type:text" json:"userAgent,omitempty"`        // User agent string
}
