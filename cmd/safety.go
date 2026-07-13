package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/edbarbera/gitle/internal/ui"
)

// largeFileBytes is the size above which a file is flagged as likely too big to
// belong in version control (build output, videos, datasets, etc.).
const largeFileBytes = 10 * 1024 * 1024

// secretGlobs match file names that commonly hold passwords, keys or tokens.
// Committing these — especially then sending online — can leak credentials.
var secretGlobs = []string{
	".env", ".env.*",
	"*.pem", "*.key", "*.p12", "*.pfx", "*.keystore", "*.crt",
	"id_rsa", "id_dsa", "id_ecdsa", "id_ed25519",
	"credentials.json", ".npmrc", ".pypirc",
}

// protectedBranches are shared lines of work where pushing directly is risky.
var protectedBranches = map[string]bool{"main": true, "master": true}

// looksLikeSecret reports whether a path's name matches a known secret pattern.
func looksLikeSecret(path string) bool {
	base := filepath.Base(path)
	for _, g := range secretGlobs {
		if ok, _ := filepath.Match(g, base); ok {
			return true
		}
	}
	return false
}

// fileSize returns a path's size in bytes, or (0, false) if it can't be sized.
func fileSize(path string) (int64, bool) {
	fi, err := os.Stat(path)
	if err != nil || fi.IsDir() {
		return 0, false
	}
	return fi.Size(), true
}

// reviewRisks checks the files about to be saved for secrets and oversized
// files. If any are found it warns in plain English and asks the user to
// confirm. Returns true when it's safe (or confirmed) to proceed.
func reviewRisks(paths []string) bool {
	var secrets, large []string
	for _, p := range paths {
		if looksLikeSecret(p) {
			secrets = append(secrets, p)
		}
		if sz, ok := fileSize(p); ok && sz > largeFileBytes {
			large = append(large, fmt.Sprintf("%s (%s)", p, humanSize(sz)))
		}
	}
	if len(secrets) == 0 && len(large) == 0 {
		return true
	}

	if len(secrets) > 0 {
		ui.Warn("These look like private/secret files:")
		for _, p := range secrets {
			ui.Hint("  • %s", p)
		}
		ui.Hint("Saving them can leak passwords or keys — especially once you send online.")
	}
	if len(large) > 0 {
		ui.Warn("These files are large and may bloat your project:")
		for _, p := range large {
			ui.Hint("  • %s", p)
		}
	}

	if !ui.IsInteractive() {
		ui.Warn("Saving anyway — no terminal here to ask.")
		return true
	}
	return ui.ConfirmDefault("Save these anyway?", false)
}

// humanSize renders a byte count as a short human-friendly string.
func humanSize(n int64) string {
	switch {
	case n >= 1<<30:
		return fmt.Sprintf("%.1f GB", float64(n)/(1<<30))
	case n >= 1<<20:
		return fmt.Sprintf("%.1f MB", float64(n)/(1<<20))
	case n >= 1<<10:
		return fmt.Sprintf("%.1f KB", float64(n)/(1<<10))
	default:
		return fmt.Sprintf("%d B", n)
	}
}
