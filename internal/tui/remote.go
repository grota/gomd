package tui

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
)

// ErrUnknownTheme reports a name that is neither built-in nor in the pinned set.
var ErrUnknownTheme = errors.New("unknown theme")

// ErrThemesNotDownloaded reports that the pinned theme set is not cached yet.
var ErrThemesNotDownloaded = errors.New("Ghostty themes not downloaded")

// Variables so tests can point them at a local server and a temp dir.
var (
	remoteThemesWantSHA256 = remoteThemesSHA256
	remoteThemesURL        = "https://github.com/mbadolato/iTerm2-Color-Schemes/releases/download/" +
		remoteThemesTag + "/ghostty-themes.tgz"
	remoteThemesDir = func() string {
		dir, err := os.UserCacheDir()
		if err != nil {
			return ""
		}
		return filepath.Join(dir, "gomd", "themes", remoteThemesTag)
	}
)

const (
	completeMarker  = ".complete"
	maxArchiveBytes = 16 << 20 // the archive is ~96 KB; refuse anything absurd
	maxThemeBytes   = 64 << 10
)

// IsBuiltinTheme reports whether name needs no download.
func IsBuiltinTheme(name string) bool {
	_, ok := themes[name]
	return ok
}

// RemoteThemesCached reports whether the pinned theme set is unpacked.
func RemoteThemesCached() bool {
	dir := remoteThemesDir()
	if dir == "" {
		return false
	}
	_, err := os.Stat(filepath.Join(dir, completeMarker))
	return err == nil
}

// EnsureRemoteThemes downloads, verifies and unpacks the pinned theme set
// unless it is already cached.
func EnsureRemoteThemes(ctx context.Context) error {
	if RemoteThemesCached() {
		return nil
	}
	dir := remoteThemesDir()
	if dir == "" {
		return errors.New("no user cache directory for themes")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, remoteThemesURL, nil)
	if err != nil {
		return err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("download themes: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download themes: %s", resp.Status)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, maxArchiveBytes+1))
	if err != nil {
		return fmt.Errorf("download themes: %w", err)
	}
	if len(data) > maxArchiveBytes {
		return errors.New("download themes: archive too large")
	}
	if sum := sha256.Sum256(data); hex.EncodeToString(sum[:]) != remoteThemesWantSHA256 {
		return errors.New("download themes: checksum mismatch, refusing the archive")
	}

	// Unpack into a temp dir and rename, so a half-written cache never counts
	// as complete.
	if err := os.MkdirAll(filepath.Dir(dir), 0o755); err != nil {
		return err
	}
	tmp, err := os.MkdirTemp(filepath.Dir(dir), ".themes-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmp)
	if err := extractThemes(bytes.NewReader(data), tmp); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(tmp, completeMarker), nil, 0o644); err != nil {
		return err
	}
	if err := os.RemoveAll(dir); err != nil {
		return err
	}
	return os.Rename(tmp, dir)
}

// extractThemes writes each regular file under "ghostty/" in the archive to
// dir, flat. Only the base name is used, so no entry can escape dir.
func extractThemes(r io.Reader, dir string) error {
	gz, err := gzip.NewReader(r)
	if err != nil {
		return fmt.Errorf("unpack themes: %w", err)
	}
	tr := tar.NewReader(gz)
	n := 0
	for {
		h, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("unpack themes: %w", err)
		}
		name := path.Base(h.Name)
		if h.Typeflag != tar.TypeReg || path.Dir(path.Clean(h.Name)) != "ghostty" || !validThemeName(name) {
			continue
		}
		data, err := io.ReadAll(io.LimitReader(tr, maxThemeBytes+1))
		if err != nil {
			return fmt.Errorf("unpack themes: %w", err)
		}
		if len(data) > maxThemeBytes {
			return fmt.Errorf("unpack themes: %q too large", name)
		}
		if err := os.WriteFile(filepath.Join(dir, name), data, 0o644); err != nil {
			return err
		}
		n++
	}
	if n == 0 {
		return errors.New("unpack themes: no themes in archive")
	}
	return nil
}

// validThemeName rejects names that are not a plain file name.
func validThemeName(name string) bool {
	return name != "" && !strings.HasPrefix(name, ".") && !strings.ContainsAny(name, `/\`)
}

// loadRemoteTheme loads a theme from the cached pinned set.
func loadRemoteTheme(name string) (Theme, error) {
	if !validThemeName(name) {
		return Theme{}, fmt.Errorf("%w %q", ErrUnknownTheme, name)
	}
	if !RemoteThemesCached() {
		return Theme{}, ErrThemesNotDownloaded
	}
	gt, err := parseGhosttyFile(filepath.Join(remoteThemesDir(), name))
	if errors.Is(err, os.ErrNotExist) {
		return Theme{}, fmt.Errorf("%w %q", ErrUnknownTheme, name)
	}
	if err != nil {
		return Theme{}, fmt.Errorf("theme %q: %w", name, err)
	}
	return ghosttyToTheme(name, gt), nil
}

// remoteThemeNames lists the cached pinned set, sorted by name.
func remoteThemeNames() ([]string, error) {
	if !RemoteThemesCached() {
		return nil, ErrThemesNotDownloaded
	}
	entries, err := os.ReadDir(remoteThemesDir())
	if err != nil {
		return nil, err
	}
	var names []string
	for _, e := range entries {
		if e.Type().IsRegular() && validThemeName(e.Name()) {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)
	return names, nil
}
