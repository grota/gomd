package tui

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"image"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"sync"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/grota/gomd/internal/config"
)

// Remote images: http(s) images are downloaded into the user cache
// directory, one file per URL, and shown like local files. A copy is used
// for remoteImageTTL, then downloaded again; until the new copy arrives the
// old one is shown, and it stays when the download fails (offline).

const (
	remoteImageTTL        = 7 * 24 * time.Hour  // a copy older than this is downloaded again
	remoteImageMaxAge     = 30 * 24 * time.Hour // cleanup removes copies older than this
	remoteImageCacheBytes = 100 << 20           // cleanup keeps the cache below this size
)

// errUnsupportedImage reports a download in a format gomd cannot show. It is
// expected (most README badges are SVG), so it is not reported as a failure.
var errUnsupportedImage = errors.New("not a PNG, JPEG or GIF image")

// shownImageTypes are the media types of the formats gomd decodes.
var shownImageTypes = []string{"image/png", "image/jpeg", "image/gif"}

// remoteImagesDir is a variable so tests can point it at a temp dir.
var remoteImagesDir = func() string {
	dir, err := os.UserCacheDir()
	if err != nil {
		return ""
	}
	return filepath.Join(dir, "gomd", "images")
}

// remoteImages downloads and caches http(s) images.
type remoteImages struct {
	dir      string
	maxBytes int64
	clean    sync.Once
}

// newRemoteImages returns nil when remote images are off.
func newRemoteImages(cfg config.ImageConfig) *remoteImages {
	dir := remoteImagesDir()
	if !cfg.Remote || dir == "" {
		return nil
	}
	return &remoteImages{dir: dir, maxBytes: int64(max(cfg.RemoteMaxMB, 1)) << 20}
}

func isRemoteImage(src string) bool {
	return strings.HasPrefix(src, "http://") || strings.HasPrefix(src, "https://")
}

func (r *remoteImages) pathFor(url string) string {
	sum := sha256.Sum256([]byte(url))
	return filepath.Join(r.dir, hex.EncodeToString(sum[:]))
}

// cached returns the cached copy of url ("" when there is none) and whether
// it is younger than remoteImageTTL.
func (r *remoteImages) cached(url string) (path string, fresh bool) {
	path = r.pathFor(url)
	fi, err := os.Stat(path)
	if err != nil {
		return "", false
	}
	return path, time.Since(fi.ModTime()) < remoteImageTTL
}

// download fetches url into the cache and returns the file. It refuses
// anything over maxBytes and anything gomd cannot decode.
func (r *remoteImages) download(ctx context.Context, url string) (string, error) {
	r.clean.Do(r.cleanup)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("%s: %s", url, resp.Status)
	}
	// A declared type gomd cannot show (SVG badges, WebP, HTML error pages)
	// is refused before the body is read. Other types, possibly mislabelled,
	// go through the decode check below.
	if mt, _, err := mime.ParseMediaType(resp.Header.Get("Content-Type")); err == nil &&
		(mt == "text/html" || strings.HasPrefix(mt, "image/") && !slices.Contains(shownImageTypes, mt)) {
		return "", fmt.Errorf("%s: %w (%s)", url, errUnsupportedImage, mt)
	}
	tooLarge := fmt.Errorf("%s: larger than %d MB (images.remote_max_mb)", url, r.maxBytes>>20)
	if resp.ContentLength > r.maxBytes {
		return "", tooLarge
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, r.maxBytes+1))
	if err != nil {
		return "", fmt.Errorf("%s: %w", url, err)
	}
	if int64(len(data)) > r.maxBytes {
		return "", tooLarge
	}
	// The bytes are already in memory: keep them only when they decode, so
	// the cache holds nothing that cannot be shown.
	if _, _, err := image.DecodeConfig(bytes.NewReader(data)); err != nil {
		return "", fmt.Errorf("%s: %w", url, errUnsupportedImage)
	}
	if err := os.MkdirAll(r.dir, 0o755); err != nil {
		return "", err
	}
	// Write and rename, so a half-written file is never used.
	tmp, err := os.CreateTemp(r.dir, ".download-")
	if err != nil {
		return "", err
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return "", err
	}
	if err := tmp.Close(); err != nil {
		return "", err
	}
	path := r.pathFor(url)
	return path, os.Rename(tmp.Name(), path)
}

// cleanup removes copies older than remoteImageMaxAge, then the oldest ones
// until the cache is below remoteImageCacheBytes. A copy in use is younger
// than remoteImageTTL, since it is downloaded again after that.
func (r *remoteImages) cleanup() {
	entries, err := os.ReadDir(r.dir)
	if err != nil {
		return
	}
	type file struct {
		path string
		mod  time.Time
		size int64
	}
	var files []file
	var total int64
	for _, e := range entries {
		fi, err := e.Info()
		if err != nil || !fi.Mode().IsRegular() {
			continue
		}
		path := filepath.Join(r.dir, e.Name())
		if time.Since(fi.ModTime()) > remoteImageMaxAge ||
			strings.HasPrefix(e.Name(), ".download-") && time.Since(fi.ModTime()) > time.Hour {
			os.Remove(path)
			continue
		}
		files = append(files, file{path, fi.ModTime(), fi.Size()})
		total += fi.Size()
	}
	sort.Slice(files, func(i, j int) bool { return files[i].mod.Before(files[j].mod) })
	for _, f := range files {
		if total <= remoteImageCacheBytes {
			break
		}
		if os.Remove(f.path) == nil {
			total -= f.size
		}
	}
}

// RemoteImageFetcher returns the resolver render mode uses for http(s)
// images: a fresh cached copy, else a download (a stale copy when it
// fails), or nil when remote images are off. warn reports failed downloads.
func RemoteImageFetcher(cfg config.ImageConfig, warn func(error)) func(url string) string {
	r := newRemoteImages(cfg)
	if r == nil {
		return nil
	}
	return func(url string) string {
		path, fresh := r.cached(url)
		if fresh {
			return path
		}
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		got, err := r.download(ctx, url)
		if err != nil {
			if !errors.Is(err, errUnsupportedImage) {
				warn(err)
			}
			return path
		}
		return got
	}
}

// imageDownloadedMsg reports the end of a background image download.
type imageDownloadedMsg struct {
	url, path string
	err       error
}

// remoteImage is the TUI's resolver for http(s) images: it returns the
// cached copy, possibly stale or "", and starts a background download when
// there is no fresh one. A failed download is not retried in this session.
func (a *App) remoteImage(url string) string {
	path, fresh := a.remote.cached(url)
	if !fresh && !a.downloads[url] {
		a.downloads[url] = true
		a.pendingDownloads = append(a.pendingDownloads, url)
	}
	return path
}

func (a *App) downloadCmd(url string) tea.Cmd {
	r := a.remote
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		path, err := r.download(ctx, url)
		return imageDownloadedMsg{url, path, err}
	}
}

// handleImageDownloaded renders again with the new copy, replacing the
// image uploaded from the old one.
func (a *App) handleImageDownloaded(msg imageDownloadedMsg) {
	if msg.err != nil {
		if !errors.Is(msg.err, errUnsupportedImage) {
			a.statusMsg = "Image download failed: " + msg.err.Error()
		}
		return
	}
	delete(a.downloads, msg.url)
	if k, ok := a.images[msg.path]; ok {
		a.pendingUploads = append(a.pendingUploads, k.deleteSeq())
		delete(a.images, msg.path)
	}
	a.renderedLines = nil
}
