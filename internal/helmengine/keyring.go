package helmengine

import (
	"os"
	"path/filepath"
)

// GnuPGHome returns $GNUPGHOME, or ~/.gnupg, where the Helm CLI looks for
// the default public keyring used by --verify and --sign.
func GnuPGHome() string {
	if v, ok := os.LookupEnv("GNUPGHOME"); ok {
		return v
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ".gnupg"
	}
	return filepath.Join(home, ".gnupg")
}
