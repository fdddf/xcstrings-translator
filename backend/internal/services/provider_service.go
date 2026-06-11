package services

import (
	"errors"
	"fmt"

	"github.com/fdddf/opentrans/internal/dao/query"
	"github.com/fdddf/opentrans/internal/database"
	"gorm.io/gorm"
)

// ProviderService handles provider configuration operations
type ProviderService struct {
	DB    *database.Database
	Query *query.Query
}

// CreateProviderConfig creates a new provider configuration for a user
func (s *ProviderService) CreateProviderConfig(userID uint, providerType string, configData map[string]interface{}, isDefault bool) (*database.ProviderConfig, error) {
	// If setting this as default, unset any existing default for this provider type
	if isDefault {
		err := s.UnsetDefaultProviderConfig(userID, providerType)
		if err != nil {
			return nil, fmt.Errorf("failed to unset existing default: %v", err)
		}
	}

	config := &database.ProviderConfig{
		UserID:       userID,
		ProviderType: providerType,
		ConfigData:   configData,
		IsDefault:    isDefault,
	}

	if err := s.Query.ProviderConfig.Create(config); err != nil {
		return nil, fmt.Errorf("failed to create provider config: %v", err)
	}

	return config, nil
}

// GetProviderConfig retrieves a specific provider configuration by ID
func (s *ProviderService) GetProviderConfig(configID uint) (*database.ProviderConfig, error) {
	config, err := s.Query.ProviderConfig.First()
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, errors.New("provider configuration not found")
	}

	if err != nil {
		return nil, fmt.Errorf("database error: %v", err)
	}

	// Get by ID
	config, err = s.Query.ProviderConfig.Where(s.Query.ProviderConfig.ID.Eq(configID)).First()
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, errors.New("provider configuration not found")
	}

	if err != nil {
		return nil, fmt.Errorf("database error: %v", err)
	}

	return config, nil
}

// GetProviderConfigsByUser retrieves all provider configurations for a user
func (s *ProviderService) GetProviderConfigsByUser(userID uint) ([]database.ProviderConfig, error) {
	configs, err := s.Query.ProviderConfig.Where(
		s.Query.ProviderConfig.UserID.Eq(userID),
	).Order(s.Query.ProviderConfig.CreatedAt.Desc()).Find()
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve provider configs: %v", err)
	}

	// Convert slice of pointers to slice of values
	result := make([]database.ProviderConfig, len(configs))
	for i, config := range configs {
		result[i] = *config
	}

	return result, nil
}

// GetProviderConfigsByUserAndType retrieves provider configurations for a user filtered by provider type
func (s *ProviderService) GetProviderConfigsByUserAndType(userID uint, providerType string) ([]database.ProviderConfig, error) {
	configs, err := s.Query.ProviderConfig.Where(
		s.Query.ProviderConfig.UserID.Eq(userID),
		s.Query.ProviderConfig.ProviderType.Eq(providerType),
	).Order(s.Query.ProviderConfig.CreatedAt.Desc()).Find()
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve provider configs: %v", err)
	}

	// Convert slice of pointers to slice of values
	result := make([]database.ProviderConfig, len(configs))
	for i, config := range configs {
		result[i] = *config
	}

	return result, nil
}

// GetDefaultProviderConfig retrieves the default provider configuration for a user and provider type
func (s *ProviderService) GetDefaultProviderConfig(userID uint, providerType string) (*database.ProviderConfig, error) {
	config, err := s.Query.ProviderConfig.Where(
		s.Query.ProviderConfig.UserID.Eq(userID),
		s.Query.ProviderConfig.ProviderType.Eq(providerType),
		s.Query.ProviderConfig.IsDefault.Is(true),
	).First()
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, errors.New("no default provider configuration found")
	}

	if err != nil {
		return nil, fmt.Errorf("database error: %v", err)
	}

	return config, nil
}

// UpdateProviderConfig updates an existing provider configuration
func (s *ProviderService) UpdateProviderConfig(configID uint, updates map[string]interface{}) error {
	// Check if the config ID exists
	existing, err := s.Query.ProviderConfig.Where(s.Query.ProviderConfig.ID.Eq(configID)).First()
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return errors.New("provider configuration not found")
	}

	if err != nil {
		return fmt.Errorf("database error: %v", err)
	}

	// If updating IsDefault to true, unset any existing default for this provider type
	if isDefault, ok := updates["IsDefault"].(bool); ok && isDefault {
		err := s.UnsetDefaultProviderConfig(existing.UserID, existing.ProviderType)
		if err != nil {
			return fmt.Errorf("failed to unset existing default: %v", err)
		}
	}

	_, err = s.Query.ProviderConfig.Where(s.Query.ProviderConfig.ID.Eq(configID)).Updates(updates)
	if err != nil {
		return fmt.Errorf("failed to update provider config: %v", err)
	}

	return nil
}

// DeleteProviderConfig deletes a provider configuration
func (s *ProviderService) DeleteProviderConfig(configID uint) error {
	result, err := s.Query.ProviderConfig.Where(s.Query.ProviderConfig.ID.Eq(configID)).Delete()
	if err != nil {
		return fmt.Errorf("failed to delete provider config: %v", err)
	}

	if result.RowsAffected == 0 {
		return errors.New("provider configuration not found")
	}

	return nil
}

// UnsetDefaultProviderConfig sets all configurations of a specific type for a user to not be default
func (s *ProviderService) UnsetDefaultProviderConfig(userID uint, providerType string) error {
	_, err := s.Query.ProviderConfig.Where(
		s.Query.ProviderConfig.UserID.Eq(userID),
		s.Query.ProviderConfig.ProviderType.Eq(providerType),
		s.Query.ProviderConfig.IsDefault.Is(true),
	).Update(s.Query.ProviderConfig.IsDefault, false)

	if err != nil {
		return fmt.Errorf("failed to unset default provider config: %v", err)
	}

	return nil
}

// ValidateProviderConfigData validates the structure of provider configuration data
func (s *ProviderService) ValidateProviderConfigData(providerType string, configData map[string]interface{}) error {
	switch providerType {
	case "openai":
		// Required fields: apiKey
		if apiKey, exists := configData["apiKey"]; !exists || apiKey == "" {
			return errors.New("apiKey is required for OpenAI provider")
		}
	case "google":
		// Required fields: apiKey
		if apiKey, exists := configData["apiKey"]; !exists || apiKey == "" {
			return errors.New("apiKey is required for Google provider")
		}
	case "deepl":
		// Required fields: apiKey
		if apiKey, exists := configData["apiKey"]; !exists || apiKey == "" {
			return errors.New("apiKey is required for DeepL provider")
		}
	case "baidu":
		// Required fields: appId, appSecret
		if appID, exists := configData["appId"]; !exists || appID == "" {
			return errors.New("appId is required for Baidu provider")
		}
		if appSecret, exists := configData["appSecret"]; !exists || appSecret == "" {
			return errors.New("appSecret is required for Baidu provider")
		}
	case "appleconnect":
		// Required fields: issuerID, keyID, privateKey
		if issuerID, exists := configData["issuerID"]; !exists || issuerID == "" {
			return errors.New("issuerID is required for Apple Connect provider")
		}
		if keyID, exists := configData["keyID"]; !exists || keyID == "" {
			return errors.New("keyID is required for Apple Connect provider")
		}
		if privateKey, exists := configData["privateKey"]; !exists || privateKey == "" {
			return errors.New("privateKey is required for Apple Connect provider")
		}
	case "llama":
		// Required fields: modelPath, libPath
		if modelPath, exists := configData["modelPath"]; !exists || modelPath == "" {
			return errors.New("modelPath is required for Llama provider")
		}
		if libPath, exists := configData["libPath"]; !exists || libPath == "" {
			return errors.New("libPath is required for Llama provider")
		}
	default:
		return fmt.Errorf("unknown provider type: %s", providerType)
	}

	return nil
}

// SanitizeProviderConfig removes sensitive fields from config data before returning to client
func (s *ProviderService) SanitizeProviderConfig(config *database.ProviderConfig) *database.ProviderConfig {
	sanitized := *config

	// Create a copy of ConfigData to avoid modifying the original
	safeConfigData := make(map[string]interface{})
	for key, value := range config.ConfigData {
		// Don't include sensitive fields in the response
		if isSensitiveField(key) {
			safeConfigData[key] = "***REDACTED***" // Show placeholder for sensitive values
		} else {
			safeConfigData[key] = value
		}
	}

	sanitized.ConfigData = safeConfigData
	return &sanitized
}

// isSensitiveField checks if a configuration field is sensitive
func isSensitiveField(field string) bool {
	sensitiveFields := []string{
		"apiKey", "api_key", "password", "secret", "appSecret", "app_secret",
		"token", "access_token", "refresh_token", "private_key", "client_secret",
	}

	for _, sensitiveField := range sensitiveFields {
		if field == sensitiveField {
			return true
		}
	}

	return false
}

// GetProviderConfigForTranslation prepares configuration data specifically for translation services
func (s *ProviderService) GetProviderConfigForTranslation(config *database.ProviderConfig) map[string]interface{} {
	// Return the config data without sanitization for use in translation services
	// This is safe since it's only used internally in the translation process
	return config.ConfigData
}

// ResolveProviderConfigForUser fetches a saved provider configuration, verifies it belongs to the
// given user, and returns its provider type together with the raw (unsanitized) config data so it
// can be used by the translation services. Secrets such as the API key are never exposed to the
// client, so callers reference a configuration by ID and let the server resolve the real values.
func (s *ProviderService) ResolveProviderConfigForUser(userID, configID uint) (string, map[string]interface{}, error) {
	config, err := s.GetProviderConfig(configID)
	if err != nil {
		return "", nil, err
	}

	if config.UserID != userID {
		return "", nil, errors.New("provider configuration not found")
	}

	return config.ProviderType, config.ConfigData, nil
}

// Global functions for backward compatibility
var providerServiceInstance *ProviderService

func SetProviderService(db *database.Database) {
	providerServiceInstance = &ProviderService{
		DB:    db,
		Query: query.Use(db.DB),
	}
}

func CreateProviderConfig(userID uint, providerType string, configData map[string]interface{}, isDefault bool) (*database.ProviderConfig, error) {
	return providerServiceInstance.CreateProviderConfig(userID, providerType, configData, isDefault)
}

func GetProviderConfig(configID uint) (*database.ProviderConfig, error) {
	return providerServiceInstance.GetProviderConfig(configID)
}

func GetProviderConfigsByUser(userID uint) ([]database.ProviderConfig, error) {
	return providerServiceInstance.GetProviderConfigsByUser(userID)
}

func GetProviderConfigsByUserAndType(userID uint, providerType string) ([]database.ProviderConfig, error) {
	return providerServiceInstance.GetProviderConfigsByUserAndType(userID, providerType)
}

func GetDefaultProviderConfig(userID uint, providerType string) (*database.ProviderConfig, error) {
	return providerServiceInstance.GetDefaultProviderConfig(userID, providerType)
}

func UpdateProviderConfig(configID uint, updates map[string]interface{}) error {
	return providerServiceInstance.UpdateProviderConfig(configID, updates)
}

func DeleteProviderConfig(configID uint) error {
	return providerServiceInstance.DeleteProviderConfig(configID)
}

func UnsetDefaultProviderConfig(userID uint, providerType string) error {
	return providerServiceInstance.UnsetDefaultProviderConfig(userID, providerType)
}

func ValidateProviderConfigData(providerType string, configData map[string]interface{}) error {
	return providerServiceInstance.ValidateProviderConfigData(providerType, configData)
}

func SanitizeProviderConfig(config *database.ProviderConfig) *database.ProviderConfig {
	return providerServiceInstance.SanitizeProviderConfig(config)
}

func GetProviderConfigForTranslation(config *database.ProviderConfig) map[string]interface{} {
	return providerServiceInstance.GetProviderConfigForTranslation(config)
}

func ResolveProviderConfigForUser(userID, configID uint) (string, map[string]interface{}, error) {
	return providerServiceInstance.ResolveProviderConfigForUser(userID, configID)
}