package services

import (
	"fmt"
	"strings"
	"time"

	"github.com/fdddf/opentrans/internal/database"
	"github.com/fdddf/opentrans/pkg/appleconnect"
)

// SyncDirection represents the direction of synchronization
type SyncDirection string

const (
	SyncDirectionPull SyncDirection = "pull" // Apple -> Local
	SyncDirectionPush SyncDirection = "push" // Local -> Apple
	SyncDirectionBoth SyncDirection = "both" // Bidirectional
)

// ConflictResolutionStrategy represents how to handle conflicts
type ConflictResolutionStrategy string

const (
	ConflictResolutionAppleFirst ConflictResolutionStrategy = "apple_first" // Always use Apple's version
	ConflictResolutionLocalFirst ConflictResolutionStrategy = "local_first" // Always use local version
	ConflictResolutionManual     ConflictResolutionStrategy = "manual"      // Require manual resolution
)

// AppleConnectService wraps Apple Connect sync operations.
type AppleConnectService struct {
	AppService             *AppService
	AppLocalizationService *AppLocalizationService
}

// SyncApps syncs apps from Apple Connect into the DB.
func (s *AppleConnectService) SyncApps(userID uint, issuerID, keyID, privateKeyPath, privateKey string) ([]database.App, error) {
	// Log the sync start
	fmt.Printf("Starting Apple Connect app sync for user: %d\n", userID)

	client, err := appleconnect.NewAppleConnectClient(issuerID, keyID, privateKeyPath, privateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create Apple Connect client: %v", err)
	}

	appsResponse, err := client.GetApps()
	if err != nil {
		return nil, fmt.Errorf("failed to get apps from Apple Connect: %v", err)
	}

	var syncedApps []database.App
	var failedApps []string
	var updatedApps []string
	var createdApps []string

	for _, appData := range appsResponse.Data {
		fmt.Printf("[DEBUG] Apple API returned app: id=%s, bundleId=%s, name='%s', primaryLocale='%s'\n",
			appData.ID, appData.Attributes.BundleID, appData.Attributes.Name, appData.Attributes.PrimaryLocale)

		// Try to get the app name from its primary locale localization
		appName := appData.Attributes.Name
		if appName == "" {
			// If name is not available in the app data, try to get it from localizations
			localizationsResponse, _, _, err := client.GetAppLocalizations(appData.ID)
			if err == nil && len(localizationsResponse.Data) > 0 {
				// Try to find the primary locale localization
				for _, loc := range localizationsResponse.Data {
					if loc.Attributes.Locale == appData.Attributes.PrimaryLocale && loc.Attributes.Name != "" {
						appName = loc.Attributes.Name
						fmt.Printf("[DEBUG] Got app name from primary locale localization: '%s'\n", appName)
						break
					}
				}
				// If primary locale not found, use the first localization with a name
				if appName == "" {
					for _, loc := range localizationsResponse.Data {
						if loc.Attributes.Name != "" {
							appName = loc.Attributes.Name
							fmt.Printf("[DEBUG] Got app name from first localization: '%s'\n", appName)
							break
						}
					}
				}
			}
		}

		existingApp, err := s.AppService.GetAppByBundleID(appData.Attributes.BundleID)
		if err == nil && existingApp != nil {
			err = s.AppService.UpdateApp(existingApp.ID, map[string]interface{}{
				"Name":          appName,
				"AppleID":       appData.ID,
				"PrimaryLocale": appData.Attributes.PrimaryLocale,
				"Origin":        "synced",
			})
			if err != nil {
				failedApps = append(failedApps, fmt.Sprintf("%s: %v", appData.Attributes.BundleID, err))
				continue
			}
			syncedApps = append(syncedApps, *existingApp)
			updatedApps = append(updatedApps, appData.Attributes.BundleID)
			continue
		}

		newApp, err := s.AppService.CreateApp(userID, appName, "", appData.Attributes.BundleID, appData.ID, appData.Attributes.PrimaryLocale, nil)
		if err != nil {
			failedApps = append(failedApps, fmt.Sprintf("%s: %v", appData.Attributes.BundleID, err))
			continue
		}
		// Update origin to "synced" since this app came from Apple Connect
		err = s.AppService.UpdateApp(newApp.ID, map[string]interface{}{
			"Origin": "synced",
		})
		if err != nil {
			failedApps = append(failedApps, fmt.Sprintf("%s: failed to set origin: %v", appData.Attributes.BundleID, err))
			continue
		}
		syncedApps = append(syncedApps, *newApp)
		createdApps = append(createdApps, appData.Attributes.BundleID)
	}

	// Log sync results
	fmt.Printf("Apple Connect app sync completed. Updated: %d, Created: %d, Failed: %d\n",
		len(updatedApps), len(createdApps), len(failedApps))

	if len(failedApps) > 0 {
		fmt.Printf("Failed apps: %v\n", failedApps)
	}

	return syncedApps, nil
}

// SyncLocalizations syncs localizations from Apple Connect into the DB.
func (s *AppleConnectService) SyncLocalizations(appID uint, issuerID, keyID, privateKeyPath, privateKey string) ([]database.AppLocalization, error) {
	app, err := s.AppService.GetApp(appID)
	if err != nil {
		return nil, err
	}

	client, err := appleconnect.NewAppleConnectClient(issuerID, keyID, privateKeyPath, privateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create Apple Connect client: %v", err)
	}

	localizationsResponse, versionString, versionState, err := client.GetAppLocalizations(app.AppleID)
	if err != nil {
		return nil, fmt.Errorf("failed to get localizations from Apple Connect: %v", err)
	}

	var syncedLocalizations []database.AppLocalization
	for _, locData := range localizationsResponse.Data {
		fmt.Printf("[DEBUG] Apple API returned for locale %s: name='%s', subtitle='%s', description='%s'\n",
			locData.Attributes.Locale, locData.Attributes.Name, locData.Attributes.Subtitle, locData.Attributes.Description)

		existing, err := s.AppLocalizationService.GetAppLocalization(appID, locData.Attributes.Locale)
		if err == nil && existing != nil {
			// Prepare update data
			// If Apple doesn't provide a name, use the app's default name
			name := locData.Attributes.Name
			if name == "" {
				name = app.Name
			}

			updateData := map[string]interface{}{
				"Name":            name,
				"Subtitle":        locData.Attributes.Subtitle,
				"PrivacyURL":      locData.Attributes.PrivacyURL,
				"MarketingURL":    locData.Attributes.MarketingURL,
				"SupportURL":      locData.Attributes.SupportURL,
				"Description":     locData.Attributes.Description,
				"Keywords":        locData.Attributes.Keywords,
				"WhatsNew":        locData.Attributes.WhatsNew,
				"PromotionalText": locData.Attributes.PromotionalText,
				"Source":          "apple",
				"SyncStatus":      "synced",
				"SyncedAt":        time.Now(),
				"Version":         versionString,
				"VersionState":    versionState,
			}

			err = s.AppLocalizationService.UpdateAppLocalization(appID, locData.Attributes.Locale, updateData)
			if err != nil {
				return nil, fmt.Errorf("failed to update localization %s: %v", locData.Attributes.Locale, err)
			}
			syncedLocalizations = append(syncedLocalizations, *existing)
			continue
		}

		// If Apple doesn't provide a name, use the app's default name
		name := locData.Attributes.Name
		if name == "" {
			name = app.Name
		}

		newLoc, err := s.AppLocalizationService.CreateAppLocalization(
			appID,
			locData.Attributes.Locale,
			name,
			locData.Attributes.Subtitle,
			locData.Attributes.PrivacyURL,
			locData.Attributes.MarketingURL,
			locData.Attributes.SupportURL,
			locData.Attributes.Description,
			locData.Attributes.Keywords,
			locData.Attributes.WhatsNew,
			locData.Attributes.PromotionalText,
			"", // WhatToTest - not available in Apple Connect API response
		)
		if err != nil {
			return nil, fmt.Errorf("failed to create localization %s: %v", locData.Attributes.Locale, err)
		}

		_ = s.AppLocalizationService.UpdateAppLocalization(appID, locData.Attributes.Locale, map[string]interface{}{
			"Source":       "apple",
			"SyncStatus":   "synced",
			"SyncedAt":     time.Now(),
			"Version":      versionString,
			"VersionState": versionState,
		})
		syncedLocalizations = append(syncedLocalizations, *newLoc)
	}

	return syncedLocalizations, nil
}

// CheckLocalizationConflicts compares local and Apple Connect versions to detect conflicts
func (s *AppleConnectService) CheckLocalizationConflicts(appID uint, issuerID, keyID, privateKeyPath, privateKey string) ([]map[string]interface{}, error) {
	app, err := s.AppService.GetApp(appID)
	if err != nil {
		return nil, err
	}

	client, err := appleconnect.NewAppleConnectClient(issuerID, keyID, privateKeyPath, privateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create Apple Connect client: %v", err)
	}

	localizationsResponse, _, _, err := client.GetAppLocalizations(app.AppleID)
	if err != nil {
		return nil, fmt.Errorf("failed to get localizations from Apple Connect: %v", err)
	}

	var conflicts []map[string]interface{}

	for _, appleLoc := range localizationsResponse.Data {
		localLoc, err := s.AppLocalizationService.GetAppLocalization(appID, appleLoc.Attributes.Locale)
		if err != nil {
			// No local localization, so no conflict
			continue
		}

		// Compare fields to detect conflicts
		hasConflict := s.isLocalizationDifferent(localLoc, &appleLoc)

		if hasConflict {
			conflict := map[string]interface{}{
				"appId":        appID,
				"languageCode": appleLoc.Attributes.Locale,
				"localVersion": localLoc,
				"appleVersion": appleLoc,
				"hasConflict":  true,
				"checkedAt":    time.Now(),
			}
			conflicts = append(conflicts, conflict)
		}
	}

	return conflicts, nil
}

// isLocalizationDifferent compares a local localization with an Apple Connect localization to determine if they are different
func (s *AppleConnectService) isLocalizationDifferent(localLoc *database.AppLocalization, appleLoc *appleconnect.AppLocalization) bool {
	// Compare each field to see if there are differences
	return localLoc.Name != appleLoc.Attributes.Name ||
		localLoc.Subtitle != appleLoc.Attributes.Subtitle ||
		localLoc.PrivacyURL != appleLoc.Attributes.PrivacyURL ||
		localLoc.MarketingURL != appleLoc.Attributes.MarketingURL ||
		localLoc.SupportURL != appleLoc.Attributes.SupportURL ||
		localLoc.Description != appleLoc.Attributes.Description ||
		localLoc.Keywords != appleLoc.Attributes.Keywords ||
		localLoc.WhatsNew != appleLoc.Attributes.WhatsNew ||
		localLoc.PromotionalText != appleLoc.Attributes.PromotionalText
}

// GetChangedLocalizations returns localizations that have been changed since the last sync with Apple
func (s *AppleConnectService) GetChangedLocalizations(appID uint) ([]database.AppLocalization, error) {
	localizations, err := s.AppLocalizationService.GetAppLocalizations(appID)
	if err != nil {
		return nil, err
	}

	var changedLocalizations []database.AppLocalization
	for _, loc := range localizations {
		if loc.Source == "local" && loc.SyncStatus == "pending" {
			// This localization was changed locally and hasn't been synced to Apple yet
			changedLocalizations = append(changedLocalizations, loc)
		}
	}

	return changedLocalizations, nil
}

// ResolveConflict allows choosing between local or Apple version for a specific localization
func (s *AppleConnectService) ResolveConflict(appID uint, languageCode string, useLocalVersion bool, issuerID, keyID, privateKeyPath, privateKey string) error {
	if useLocalVersion {
		// Push local version to Apple
		return s.pushLocalizationToApple(appID, languageCode, issuerID, keyID, privateKeyPath, privateKey)
	} else {
		// Pull Apple version to local
		return s.pullLocalizationFromApple(appID, languageCode, issuerID, keyID, privateKeyPath, privateKey)
	}
}

// pullLocalizationFromApple syncs a single localization from Apple to local
func (s *AppleConnectService) pullLocalizationFromApple(appID uint, languageCode, issuerID, keyID, privateKeyPath, privateKey string) error {
	app, err := s.AppService.GetApp(appID)
	if err != nil {
		return err
	}

	client, err := appleconnect.NewAppleConnectClient(issuerID, keyID, privateKeyPath, privateKey)
	if err != nil {
		return fmt.Errorf("failed to create Apple Connect client: %v", err)
	}

	appleLocalization, err := client.GetAppLocalization(app.AppleID, languageCode)
	if err != nil {
		return fmt.Errorf("failed to get localization from Apple Connect: %v", err)
	}

	// Get current version string and state
	versionString, versionState, err := client.GetAppVersion(app.AppleID)
	if err != nil {
		return fmt.Errorf("failed to get app version: %v", err)
	}

	// If Apple doesn't provide a name, use the app's default name
	name := appleLocalization.Attributes.Name
	if name == "" {
		name = app.Name
	}

	// Prepare update data
	updates := map[string]interface{}{
		"Name":            name,
		"Subtitle":        appleLocalization.Attributes.Subtitle,
		"PrivacyURL":      appleLocalization.Attributes.PrivacyURL,
		"MarketingURL":    appleLocalization.Attributes.MarketingURL,
		"SupportURL":      appleLocalization.Attributes.SupportURL,
		"Description":     appleLocalization.Attributes.Description,
		"Keywords":        appleLocalization.Attributes.Keywords,
		"WhatsNew":        appleLocalization.Attributes.WhatsNew,
		"PromotionalText": appleLocalization.Attributes.PromotionalText,
		"Source":          "apple",
		"SyncStatus":      "synced",
		"SyncedAt":        time.Now(),
		"Version":         versionString,
		"VersionState":    versionState,
	}

	return s.AppLocalizationService.UpdateAppLocalization(appID, languageCode, updates)
}

// pushLocalizationToApple syncs a single localization from local to Apple
func (s *AppleConnectService) pushLocalizationToApple(appID uint, languageCode, issuerID, keyID, privateKeyPath, privateKey string) error {
	app, err := s.AppService.GetApp(appID)
	if err != nil {
		return err
	}

	client, err := appleconnect.NewAppleConnectClient(issuerID, keyID, privateKeyPath, privateKey)
	if err != nil {
		return fmt.Errorf("failed to create Apple Connect client: %v", err)
	}

	localLoc, err := s.AppLocalizationService.GetAppLocalization(appID, languageCode)
	if err != nil {
		return fmt.Errorf("failed to get local localization: %v", err)
	}

	// Try to get existing localization from Apple
	appleLoc, err := client.GetAppLocalization(app.AppleID, languageCode)
	if err != nil {
		// Localization doesn't exist in Apple, create it
		_, err = client.CreateAppLocalization(
			app.AppleID,
			languageCode,
			localLoc.MarketingURL,
			localLoc.SupportURL,
			localLoc.Description,
			localLoc.Keywords,
			localLoc.WhatsNew,
			localLoc.PromotionalText,
		)
		if err != nil {
			// Check if it's a duplicate error (409)
			errStr := err.Error()
			if strings.Contains(errStr, "409") && strings.Contains(errStr, "already exists") {
				// Localization already exists, fetch it and update
				appleLoc, err = client.GetAppLocalization(app.AppleID, languageCode)
				if err != nil {
					return fmt.Errorf("failed to fetch existing localization for %s after duplicate error: %v", languageCode, err)
				}
				// Update the existing localization
				_, err = client.UpdateAppLocalization(
					appleLoc.ID,
					localLoc.MarketingURL,
					localLoc.SupportURL,
					localLoc.Description,
					localLoc.Keywords,
					localLoc.WhatsNew,
					localLoc.PromotionalText,
				)
				if err != nil {
					return fmt.Errorf("failed to update existing localization for %s: %v", languageCode, err)
				}
			} else {
				return fmt.Errorf("failed to create localization in Apple Connect: %v", err)
			}
		}
	} else {
		// Update existing localization
		_, err = client.UpdateAppLocalization(
			appleLoc.ID,
			localLoc.MarketingURL,
			localLoc.SupportURL,
			localLoc.Description,
			localLoc.Keywords,
			localLoc.WhatsNew,
			localLoc.PromotionalText,
		)
		if err != nil {
			return fmt.Errorf("failed to update localization in Apple Connect: %v", err)
		}
	}

	// Update local record
	updates := map[string]interface{}{
		"Source":     "local",
		"SyncStatus": "synced",
		"SyncedAt":   time.Now(),
	}

	return s.AppLocalizationService.UpdateAppLocalization(appID, languageCode, updates)
}

// SyncWithDirectionAndStrategy syncs localizations with specified direction and conflict resolution strategy
func (s *AppleConnectService) SyncWithDirectionAndStrategy(appID uint, direction SyncDirection, strategy ConflictResolutionStrategy, issuerID, keyID, privateKeyPath, privateKey string) ([]database.AppLocalization, []map[string]interface{}, error) {
	app, err := s.AppService.GetApp(appID)
	if err != nil {
		return nil, nil, err
	}

	client, err := appleconnect.NewAppleConnectClient(issuerID, keyID, privateKeyPath, privateKey)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create Apple Connect client: %v", err)
	}

	var syncedLocalizations []database.AppLocalization
	var conflicts []map[string]interface{}

	// Get local localizations
	localLocalizations, err := s.AppLocalizationService.GetAppLocalizations(appID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get local localizations: %v", err)
	}

	// Get Apple localizations
	appleLocalizationsResponse, versionString, versionState, err := client.GetAppLocalizations(app.AppleID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get Apple localizations: %v", err)
	}

	// Create a map of Apple localizations by locale
	appleLocMap := make(map[string]*appleconnect.AppLocalization)
	for i := range appleLocalizationsResponse.Data {
		appleLocMap[appleLocalizationsResponse.Data[i].Attributes.Locale] = &appleLocalizationsResponse.Data[i]
	}

	// Sync based on direction
	if direction == SyncDirectionPull || direction == SyncDirectionBoth {
		// Pull from Apple to local
		for _, appleLoc := range appleLocalizationsResponse.Data {
			locale := appleLoc.Attributes.Locale
			localLoc, err := s.AppLocalizationService.GetAppLocalization(appID, locale)

			if err != nil {
				// Localization doesn't exist locally, create it
				// If Apple doesn't provide a name, use the app's default name
				name := appleLoc.Attributes.Name
				if name == "" {
					name = app.Name
				}

				newLoc, err := s.AppLocalizationService.CreateAppLocalization(
					appID,
					locale,
					name,
					appleLoc.Attributes.Subtitle,
					appleLoc.Attributes.PrivacyURL,
					appleLoc.Attributes.MarketingURL,
					appleLoc.Attributes.SupportURL,
					appleLoc.Attributes.Description,
					appleLoc.Attributes.Keywords,
					appleLoc.Attributes.WhatsNew,
					appleLoc.Attributes.PromotionalText,
					"",
				)
				if err != nil {
					return nil, nil, fmt.Errorf("failed to create localization %s: %v", locale, err)
				}

				_ = s.AppLocalizationService.UpdateAppLocalization(appID, locale, map[string]interface{}{
					"Source":       "apple",
					"SyncStatus":   "synced",
					"SyncedAt":     time.Now(),
					"Version":      versionString,
					"VersionState": versionState,
				})
				syncedLocalizations = append(syncedLocalizations, *newLoc)
			} else {
				// Check for conflict
				hasConflict := s.isLocalizationDifferent(localLoc, &appleLoc)

				if hasConflict {
					if strategy == ConflictResolutionManual {
						conflicts = append(conflicts, map[string]interface{}{
							"appId":        appID,
							"languageCode": locale,
							"localVersion": localLoc,
							"appleVersion": appleLoc,
							"hasConflict":  true,
							"checkedAt":    time.Now(),
						})
						continue
					} else if strategy == ConflictResolutionAppleFirst {
						// Use Apple's version
						_ = s.pullLocalizationFromApple(appID, locale, issuerID, keyID, privateKeyPath, privateKey)
						syncedLocalizations = append(syncedLocalizations, *localLoc)
					} else {
						// Local first - keep local version, mark as pending sync
						continue
					}
				} else {
					// No conflict, update local
					_ = s.pullLocalizationFromApple(appID, locale, issuerID, keyID, privateKeyPath, privateKey)
					syncedLocalizations = append(syncedLocalizations, *localLoc)
				}
			}
		}
	}

	if direction == SyncDirectionPush || direction == SyncDirectionBoth {
		// Push from local to Apple
		for _, localLoc := range localLocalizations {
			locale := localLoc.LanguageCode
			appleLoc, exists := appleLocMap[locale]

			if !exists {
				// Localization doesn't exist in Apple, create it
				_, err := client.CreateAppLocalization(
					app.AppleID,
					locale,
					localLoc.MarketingURL,
					localLoc.SupportURL,
					localLoc.Description,
					localLoc.Keywords,
					localLoc.WhatsNew,
					localLoc.PromotionalText,
				)
				if err != nil {
					// Check if it's a duplicate error (409)
					errStr := err.Error()
					if strings.Contains(errStr, "409") && strings.Contains(errStr, "already exists") {
						// Localization already exists, fetch it and update
						appleLoc, err = client.GetAppLocalization(app.AppleID, locale)
						if err != nil {
							return nil, nil, fmt.Errorf("failed to fetch existing localization for %s after duplicate error: %v", locale, err)
						}
						// Update the existing localization
						_, err = client.UpdateAppLocalization(
							appleLoc.ID,
							localLoc.MarketingURL,
							localLoc.SupportURL,
							localLoc.Description,
							localLoc.Keywords,
							localLoc.WhatsNew,
							localLoc.PromotionalText,
						)
						if err != nil {
							return nil, nil, fmt.Errorf("failed to update existing localization for %s: %v", locale, err)
						}
					} else {
						return nil, nil, fmt.Errorf("failed to create localization in Apple %s: %v", locale, err)
					}
				}

				_ = s.AppLocalizationService.UpdateAppLocalization(appID, locale, map[string]interface{}{
					"Source":     "local",
					"SyncStatus": "synced",
					"SyncedAt":   time.Now(),
				})
				syncedLocalizations = append(syncedLocalizations, localLoc)
			} else {
				// Check for conflict
				hasConflict := s.isLocalizationDifferent(&localLoc, appleLoc)

				if hasConflict {
					if strategy == ConflictResolutionManual {
						conflicts = append(conflicts, map[string]interface{}{
							"appId":        appID,
							"languageCode": locale,
							"localVersion": localLoc,
							"appleVersion": appleLoc,
							"hasConflict":  true,
							"checkedAt":    time.Now(),
						})
						continue
					} else if strategy == ConflictResolutionLocalFirst {
						// Use local version
						_ = s.pushLocalizationToApple(appID, locale, issuerID, keyID, privateKeyPath, privateKey)
						syncedLocalizations = append(syncedLocalizations, localLoc)
					} else {
						// Apple first - keep Apple version, don't push
						continue
					}
				} else {
					// No conflict, push to Apple
					_ = s.pushLocalizationToApple(appID, locale, issuerID, keyID, privateKeyPath, privateKey)
					syncedLocalizations = append(syncedLocalizations, localLoc)
				}
			}
		}
	}

	return syncedLocalizations, conflicts, nil
}

// PushUpdateContentToApple pushes only whatsNew and promotionalText to Apple Connect for all localizations
func (s *AppleConnectService) PushUpdateContentToApple(appID uint, languageCodes []string, issuerID, keyID, privateKeyPath, privateKey string) ([]database.AppLocalization, error) {
	app, err := s.AppService.GetApp(appID)
	if err != nil {
		return nil, err
	}

	client, err := appleconnect.NewAppleConnectClient(issuerID, keyID, privateKeyPath, privateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create Apple Connect client: %v", err)
	}

	var syncedLocalizations []database.AppLocalization

	// Get local localizations
	localLocalizations, err := s.AppLocalizationService.GetAppLocalizations(appID)
	if err != nil {
		return nil, fmt.Errorf("failed to get local localizations: %v", err)
	}

	// Create a map for quick lookup
	localLocMap := make(map[string]database.AppLocalization)
	for _, loc := range localLocalizations {
		localLocMap[loc.LanguageCode] = loc
	}

	// If no specific languages provided, push all
	if len(languageCodes) == 0 {
		for _, loc := range localLocalizations {
			languageCodes = append(languageCodes, loc.LanguageCode)
		}
	}

	for _, langCode := range languageCodes {
		localLoc, exists := localLocMap[langCode]
		if !exists {
			fmt.Printf("Skipping %s: no local localization found\n", langCode)
			continue
		}

		// Skip if both fields are empty
		if localLoc.WhatsNew == "" && localLoc.PromotionalText == "" {
			fmt.Printf("Skipping %s: both whatsNew and promotionalText are empty\n", langCode)
			continue
		}

		// Get Apple localization to get the localization ID
		appleLoc, err := client.GetAppLocalization(app.AppleID, langCode)
		if err != nil {
			fmt.Printf("Warning: failed to get Apple localization for %s: %v, skipping\n", langCode, err)
			continue
		}

		// Update only whatsNew and promotionalText
		_, err = client.UpdateAppLocalizationContent(
			appleLoc.ID,
			localLoc.WhatsNew,
			localLoc.PromotionalText,
		)
		if err != nil {
			fmt.Printf("Warning: failed to update content for %s: %v\n", langCode, err)
			continue
		}

		// Update local record
		_ = s.AppLocalizationService.UpdateAppLocalization(appID, langCode, map[string]interface{}{
			"SyncStatus": "synced",
			"SyncedAt":   time.Now(),
		})

		syncedLocalizations = append(syncedLocalizations, localLoc)
		fmt.Printf("Successfully pushed update content for %s\n", langCode)
	}

	return syncedLocalizations, nil
}
