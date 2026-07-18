package ops

import (
	"fmt"
	"os"
	"path/filepath"
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

// ProtectedBranches are shared lines of work where pushing directly is risky.
var ProtectedBranches = map[string]bool{"main": true, "master": true}

// LargeFile is an oversized file, with its size already rendered for display.
type LargeFile struct {
	Path string
	Size string
}

// Risks is what a pre-save check turned up.
type Risks struct {
	Secrets []string
	Large   []LargeFile
}

// Any reports whether anything was flagged.
func (r Risks) Any() bool { return len(r.Secrets) > 0 || len(r.Large) > 0 }

// ScanRisks checks files about to be saved for secrets and oversized files. It
// only reports: whether to go ahead anyway is the caller's call, since only
// the caller can ask the user.
func ScanRisks(paths []string) Risks {
	var r Risks
	for _, p := range paths {
		if looksLikeSecret(p) {
			r.Secrets = append(r.Secrets, p)
		}
		if sz, ok := fileSize(p); ok && sz > largeFileBytes {
			r.Large = append(r.Large, LargeFile{Path: p, Size: HumanSize(sz)})
		}
	}
	return r
}

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

// HumanSize renders a byte count as a short human-friendly string.
func HumanSize(n int64) string {
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
