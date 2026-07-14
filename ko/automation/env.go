package automation

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/joho/godotenv"
)

// LoadEnv layers two optional dotenv files so mage and CI resolve config the
// same way. `.env` (git-ignored: secrets + host-specific overrides) is loaded
// first so its values win; `deploy/environment/versions.env` (shared,
// non-sensitive version pins) is loaded second and only fills the gaps. godotenv
// is first-wins and never overrides already-set variables, so this makes
// versions.env a true fallback — a CI checkout with no `.env` runs from it alone,
// and neither file needs to duplicate the other.
func LoadEnv() error {
	for _, file := range []string{".env", "deploy/environment/versions.env"} {
		if err := godotenv.Load(file); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("loading %s: %w", file, err)
		}
	}

	return nil
}

func Env(name string, fallback string) string {
	value := os.Getenv(name)
	if value == "" {
		return fallback
	}

	return value
}

func RequiredEnv(name string) (string, error) {
	value := os.Getenv(name)
	if value == "" {
		return "", fmt.Errorf("required environment variable %s is not set", name)
	}

	return value, nil
}

func ExpandPath(path string) string {
	if path == "~" {
		home, _ := os.UserHomeDir()
		return home
	}

	if strings.HasPrefix(path, "~/") {
		home, _ := os.UserHomeDir()
		return filepath.Join(home, strings.TrimPrefix(path, "~/"))
	}

	return os.ExpandEnv(path)
}
