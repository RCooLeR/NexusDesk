package buildinfo

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"
)

const (
	AppID   = "com.rcooler.nexus"
	AppName = "Nexus Augentic Studio"
	Tagline = "Agentic work. Augmented by context."
)

var (
	Version   = "0.0.0-dev"
	Commit    = "unknown"
	BuildDate = "unknown"
)

var semverPattern = regexp.MustCompile(`^\d+\.\d+\.\d+(?:-[0-9A-Za-z.-]+)?(?:\+[0-9A-Za-z.-]+)?$`)

type Info struct {
	AppID     string
	AppName   string
	Tagline   string
	Version   string
	Commit    string
	BuildDate string
}

func Current() Info {
	return Info{
		AppID:     AppID,
		AppName:   AppName,
		Tagline:   Tagline,
		Version:   strings.TrimSpace(Version),
		Commit:    strings.TrimSpace(Commit),
		BuildDate: strings.TrimSpace(BuildDate),
	}
}

func (info Info) Validate() error {
	if strings.TrimSpace(info.AppID) == "" {
		return errors.New("app id is required")
	}
	if strings.TrimSpace(info.AppName) == "" {
		return errors.New("app name is required")
	}
	if !semverPattern.MatchString(strings.TrimSpace(info.Version)) {
		return fmt.Errorf("version %q must be semantic version metadata like 1.2.3 or 1.2.3-beta.1", info.Version)
	}
	if strings.TrimSpace(info.Commit) == "" {
		return errors.New("commit metadata is required")
	}
	buildDate := strings.TrimSpace(info.BuildDate)
	if buildDate != "" && buildDate != "unknown" {
		if _, err := time.Parse(time.RFC3339, buildDate); err != nil {
			return fmt.Errorf("build date %q must be RFC3339 UTC metadata: %w", info.BuildDate, err)
		}
	}
	return nil
}

func AboutText() string {
	info := Current()
	lines := []string{
		info.AppName,
		info.Tagline,
		"",
		"Native local-first workbench.",
		"",
		"Version: " + firstNonEmpty(info.Version, "unknown"),
		"Commit: " + firstNonEmpty(info.Commit, "unknown"),
		"Build: " + firstNonEmpty(info.BuildDate, "unknown"),
	}
	return strings.Join(lines, "\n")
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}
