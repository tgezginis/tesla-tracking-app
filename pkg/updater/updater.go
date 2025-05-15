package updater

import (
	"context"
	"errors"
	"fmt"
	"log"
	"runtime"
	"strings"

	"github.com/Masterminds/semver/v3"
	"github.com/creativeprojects/go-selfupdate"
	"github.com/tgezginis/tesla-tracking-app/pkg/version"
)

// Constants for the update configuration
const (
	// GitHub repository owner
	owner = "tgezginis"
	// GitHub repository name
	repo = "tesla-tracking-app"
)

// tryDetectRelease attempts to detect a release for the given platform
func tryDetectRelease(ctx context.Context, updater *selfupdate.Updater, repository selfupdate.Repository, goos, goarch string) (*selfupdate.Release, bool, error) {
	// Override OS and arch for detection
	config := selfupdate.Config{
		OS:   goos,
		Arch: goarch,
	}
	
	customUpdater, err := selfupdate.NewUpdater(config)
	if err != nil {
		return nil, false, err
	}
	
	return customUpdater.DetectLatest(ctx, repository)
}

// HasUpdate checks if there's a newer version available
func HasUpdate() (bool, *selfupdate.Release, error) {
	ctx := context.Background()
	
	log.Printf("Checking for updates using repository: %s/%s", owner, repo)
	
	// Create a repository object
	repository := selfupdate.NewRepositorySlug(owner, repo)

	// Create default updater
	updater := selfupdate.DefaultUpdater()
	
	// Get all releases from repo (platform bağımsız)
	log.Printf("Fetching all releases from repository...")
	
	// Önce platform bağımsız olarak tüm releaseları al
	source, err := selfupdate.NewGitHubSource(selfupdate.GitHubConfig{})
	if err != nil {
		log.Printf("Error creating GitHub source: %v", err)
		return false, nil, fmt.Errorf("error creating GitHub source: %w", err)
	}
	
	releases, err := source.ListReleases(ctx, repository)
	if err != nil {
		log.Printf("Error listing releases: %v", err)
		return false, nil, fmt.Errorf("error listing releases: %w", err)
	}
	
	if len(releases) == 0 {
		log.Printf("No releases found")
		return false, nil, nil
	}
	
	log.Printf("Found %d releases", len(releases))
	
	// En son release'i al (platform bağımsız)
	latest := releases[0]
	log.Printf("Latest release: %s", latest.GetTagName())
	
	// Get current version
	currentVersion := version.String()
	
	// Normalize versions by ensuring they start with 'v' prefix if needed
	normalizedCurrent := currentVersion
	if !strings.HasPrefix(normalizedCurrent, "v") {
		normalizedCurrent = "v" + normalizedCurrent
	}
	
	latestVersion := latest.GetTagName()
	normalizedLatest := latestVersion
	if !strings.HasPrefix(normalizedLatest, "v") {
		normalizedLatest = "v" + normalizedLatest
	}
	
	// Log versions for debugging
	log.Printf("Current version: %s (normalized: %s)", currentVersion, normalizedCurrent)
	log.Printf("Latest version: %s (normalized: %s)", latestVersion, normalizedLatest)
	
	// Parse versions using semver
	vCurrent, err := semver.NewVersion(normalizedCurrent)
	if err != nil {
		log.Printf("Error parsing current version: %v", err)
		return false, nil, fmt.Errorf("error parsing current version: %w", err)
	}
	
	vLatest, err := semver.NewVersion(normalizedLatest)
	if err != nil {
		log.Printf("Error parsing latest version: %v", err)
		return false, nil, fmt.Errorf("error parsing latest version: %w", err)
	}
	
	// Check if the latest version is newer than current
	hasUpdate := vLatest.GreaterThan(vCurrent)
	log.Printf("Update available: %v (current: %s, latest: %s)", hasUpdate, vCurrent, vLatest)
	
	if !hasUpdate {
		return false, nil, nil
	}
	
	// Eğer güncelleme varsa normal yönteme geri dön ve platform için dosyaları kontrol et
	log.Printf("Checking for platform-specific release files...")
	platformRelease, found, err := updater.DetectLatest(ctx, repository)
	
	// Eğer darwin-arm64 için dosya bulunamadıysa darwin-amd64 için deneyelim (Rosetta ile çalışacak)
	if !found && runtime.GOOS == "darwin" && runtime.GOARCH == "arm64" {
		log.Printf("No release found for darwin/arm64, trying darwin/amd64 (will use Rosetta)")
		platformRelease, found, err = tryDetectRelease(ctx, updater, repository, "darwin", "amd64")
		
		if found {
			log.Printf("Found darwin/amd64 release, will use it as fallback")
		}
	}
	
	if err != nil {
		log.Printf("Error detecting platform-specific release: %v", err)
		
		// Eğer hata bu platform için release bulunamadı ise, güncelleme yok gibi davran
		if strings.Contains(err.Error(), "could not find latest release for") {
			log.Printf("No release found for this platform, skipping update")
			return false, nil, nil
		}
		
		return false, nil, fmt.Errorf("error checking for platform-specific updates: %w", err)
	}
	
	if !found {
		log.Printf("No platform-specific release found for %s/%s", runtime.GOOS, runtime.GOARCH)
		return false, nil, nil // Güncelleme yok
	}
	
	log.Printf("Found platform-specific release: %s", platformRelease.Version())
	log.Printf("Asset URL: %s", platformRelease.AssetURL)
	log.Printf("Asset Name: %s", platformRelease.AssetName)
	
	return true, platformRelease, nil
}

// DoUpdate updates the application to the specified release
func DoUpdate(release *selfupdate.Release) error {
	if release == nil {
		return errors.New("invalid release: cannot be nil")
	}

	ctx := context.Background()

	// Find the path to the current executable
	exe, err := selfupdate.ExecutablePath()
	if err != nil {
		log.Printf("Could not locate executable path: %v", err)
		return fmt.Errorf("could not locate executable path: %w", err)
	}
	log.Printf("Found executable at: %s", exe)

	// Perform the update
	log.Printf("Starting update to version %s...", release.Version())
	log.Printf("Downloading from: %s", release.AssetURL)
	
	if err := selfupdate.UpdateTo(ctx, release.AssetURL, release.AssetName, exe); err != nil {
		log.Printf("Update failed: %v", err)
		return fmt.Errorf("error occurred while updating binary: %w", err)
	}

	log.Printf("Successfully updated to version %s", release.Version())
	return nil
}

// CheckAndUpdate checks for updates and performs the update if available
func CheckAndUpdate() (bool, error) {
	log.Printf("Starting update check...")
	hasUpdate, release, err := HasUpdate()
	if err != nil {
		log.Printf("Update check failed: %v", err)
		return false, err
	}

	if !hasUpdate {
		log.Printf("No update available")
		return false, nil
	}

	// If there is an update, perform it
	log.Printf("Update available, performing update...")
	if err := DoUpdate(release); err != nil {
		return true, fmt.Errorf("update failed: %w", err)
	}

	// If we successfully updated, return true
	log.Printf("Update completed successfully")
	return true, nil
} 