package tui

import (
	"bytes"
	"context"
	"image"
	"image/png"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi/kitty"

	"github.com/grota/gomd/internal/config"
	"github.com/grota/gomd/internal/parser"
)

func pngBytes(t *testing.T, w, h int) []byte {
	t.Helper()
	var buf bytes.Buffer
	if err := png.Encode(&buf, image.NewRGBA(image.Rect(0, 0, w, h))); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

// imageServer serves body at every path and counts the requests.
func imageServer(t *testing.T, body []byte, chunked bool) (*httptest.Server, *int) {
	t.Helper()
	hits := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		if r.URL.Path == "/missing.png" {
			http.NotFound(w, r)
			return
		}
		if chunked {
			w.Write(body[:1])
			w.(http.Flusher).Flush() // no Content-Length
			w.Write(body[1:])
			return
		}
		w.Write(body)
	}))
	t.Cleanup(srv.Close)
	return srv, &hits
}

func testRemote(t *testing.T, maxMB int) *remoteImages {
	t.Helper()
	dir := t.TempDir()
	old := remoteImagesDir
	remoteImagesDir = func() string { return dir }
	t.Cleanup(func() { remoteImagesDir = old })
	return newRemoteImages(config.ImageConfig{Remote: true, RemoteMaxMB: maxMB})
}

func TestRemoteImageDownloadAndCache(t *testing.T) {
	r := testRemote(t, 10)
	body := pngBytes(t, 4, 4)
	srv, _ := imageServer(t, body, false)
	url := srv.URL + "/a.png"
	if path, fresh := r.cached(url); path != "" || fresh {
		t.Fatalf("cached before download: %q %v", path, fresh)
	}
	path, err := r.download(context.Background(), url)
	if err != nil {
		t.Fatal(err)
	}
	if got, _ := os.ReadFile(path); !bytes.Equal(got, body) {
		t.Fatal("cached copy differs from the download")
	}
	if p, fresh := r.cached(url); p != path || !fresh {
		t.Fatalf("after download: %q %v", p, fresh)
	}
	old := time.Now().Add(-remoteImageTTL - time.Hour)
	os.Chtimes(path, old, old)
	if p, fresh := r.cached(url); p != path || fresh {
		t.Fatalf("expired copy: %q %v, want the path and not fresh", p, fresh)
	}
}

func TestRemoteImageRefusals(t *testing.T) {
	r := testRemote(t, 1)
	big := append(pngBytes(t, 4, 4), make([]byte, 2<<20)...)
	for _, chunked := range []bool{false, true} {
		srv, _ := imageServer(t, big, chunked)
		if _, err := r.download(context.Background(), srv.URL+"/big.png"); err == nil || !strings.Contains(err.Error(), "larger than 1 MB") {
			t.Errorf("chunked=%v: want a size error, got %v", chunked, err)
		}
	}
	srv, _ := imageServer(t, []byte("<html>not an image</html>"), false)
	if _, err := r.download(context.Background(), srv.URL+"/page.png"); err == nil || !strings.Contains(err.Error(), "not an image") {
		t.Errorf("want a not-an-image error, got %v", err)
	}
	if _, err := r.download(context.Background(), srv.URL+"/missing.png"); err == nil || !strings.Contains(err.Error(), "404") {
		t.Errorf("want a 404 error, got %v", err)
	}
	entries, _ := os.ReadDir(r.dir)
	if len(entries) != 0 {
		t.Errorf("refused downloads left %d files in the cache", len(entries))
	}
}

func TestRemoteImageCleanup(t *testing.T) {
	r := testRemote(t, 10)
	os.MkdirAll(r.dir, 0o755)
	file := func(name string, size int64, age time.Duration) string {
		p := filepath.Join(r.dir, name)
		f, _ := os.Create(p)
		f.Truncate(size) // sparse, so no real disk use
		f.Close()
		mod := time.Now().Add(-age)
		os.Chtimes(p, mod, mod)
		return p
	}
	expired := file("expired", 10, remoteImageMaxAge+time.Hour)
	oldest := file("oldest", 60<<20, 3*time.Hour)
	newest := file("newest", 60<<20, time.Hour)
	small := file("small", 10, 2*time.Hour)
	leftover := file(".download-1", 10, 2*time.Hour)
	r.cleanup()
	for p, want := range map[string]bool{expired: false, oldest: false, newest: true, small: true, leftover: false} {
		if _, err := os.Stat(p); (err == nil) != want {
			t.Errorf("%s: kept=%v, want %v", filepath.Base(p), err == nil, want)
		}
	}
}

func TestTUIDownloadsRemoteImages(t *testing.T) {
	testRemote(t, 10)
	srv, hits := imageServer(t, pngBytes(t, 160, 32), false)
	md := "# T\n\n![logo](" + srv.URL + "/logo.png)\n\n![gone](" + srv.URL + "/missing.png)\n"
	cfg := config.Default()
	cfg.Images.Enabled = true
	a := NewApp(parser.ParseMarkdown(md), "d.md", "", cfg)
	_, cmd := a.Update(tea.WindowSizeMsg{Width: 60, Height: 30})
	if cmd == nil || len(a.downloads) != 2 {
		t.Fatalf("want two downloads started, got %v", a.downloads)
	}
	if strings.ContainsRune(a.View().Content, kitty.Placeholder) {
		t.Fatal("image shown before its download")
	}
	for _, url := range []string{srv.URL + "/logo.png", srv.URL + "/missing.png"} {
		a.Update(a.downloadCmd(url)())
	}
	if !strings.ContainsRune(a.View().Content, kitty.Placeholder) {
		t.Fatal("downloaded image not shown")
	}
	if !strings.Contains(a.statusMsg, "missing.png") {
		t.Errorf("failed download not reported: %q", a.statusMsg)
	}
	// Rendering again neither downloads the fresh image nor retries the failure.
	a.renderedLines = nil
	a.ensureRenderedLines()
	if len(a.pendingDownloads) != 0 || *hits != 2 {
		t.Errorf("downloads repeated: pending %v, %d requests", a.pendingDownloads, *hits)
	}
}

func TestStaleRemoteImageShownWhileRefreshing(t *testing.T) {
	r := testRemote(t, 10)
	srv, _ := imageServer(t, pngBytes(t, 8, 8), false)
	url := srv.URL + "/a.png"
	path, err := r.download(context.Background(), url)
	if err != nil {
		t.Fatal(err)
	}
	old := time.Now().Add(-remoteImageTTL - time.Hour)
	os.Chtimes(path, old, old)
	cfg := config.Default()
	cfg.Images.Enabled = true
	a := NewApp(parser.ParseMarkdown("![a]("+url+")\n"), "d.md", "", cfg)
	if got := a.remoteImage(url); got != path || len(a.pendingDownloads) != 1 {
		t.Fatalf("want the stale copy and a refresh, got %q, %v", got, a.pendingDownloads)
	}
	srv.Close() // offline: render mode falls back to the stale copy
	var warned error
	fetch := RemoteImageFetcher(cfg.Images, func(err error) { warned = err })
	if got := fetch(url); got != path || warned == nil {
		t.Fatalf("render mode offline: got %q, warning %v", got, warned)
	}
}

func TestRemoteImagesOff(t *testing.T) {
	testRemote(t, 10)
	if newRemoteImages(config.ImageConfig{Remote: false, RemoteMaxMB: 10}) != nil {
		t.Error("remote = false still downloads")
	}
	if RemoteImageFetcher(config.ImageConfig{Remote: false}, nil) != nil {
		t.Error("render mode fetches with remote = false")
	}
}
