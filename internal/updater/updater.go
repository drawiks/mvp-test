package updater

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/mod/semver"
)

const (
	repo              = "drawiks/mvp-test"
	AssetName         = "mvp-windows-amd64.exe"
	checksumAssetName = "checksums.txt"
)

// apiBase is a variable so tests can point at an httptest server.
var apiBase = "https://api.github.com"

// Release is the update asset information from the latest GitHub release.
type Release struct {
	Version     string
	Notes       string
	AssetURL    string
	ChecksumURL string
}

// Progress is emitted while downloading and verifying an update.
type Progress struct {
	Stage      string
	Downloaded int64
	Total      int64
}

type githubRelease struct {
	TagName string        `json:"tag_name"`
	Body    string        `json:"body"`
	Assets  []githubAsset `json:"assets"`
}

type githubAsset struct {
	Name string `json:"name"`
	URL  string `json:"browser_download_url"`
}

// Check returns the latest release when it is newer than currentVersion.
func Check(ctx context.Context, currentVersion string) (Release, bool, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiBase+"/repos/"+repo+"/releases/latest", nil)
	if err != nil {
		return Release{}, false, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "mvp-test-updater")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return Release{}, false, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return Release{}, false, nil
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return Release{}, false, fmt.Errorf("github: %s: %s", resp.Status, strings.TrimSpace(string(body)))
	}

	var gr githubRelease
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&gr); err != nil {
		return Release{}, false, err
	}
	if !newer(currentVersion, gr.TagName) {
		return Release{}, false, nil
	}

	rel := Release{Version: strings.TrimPrefix(gr.TagName, "v"), Notes: gr.Body}
	for _, a := range gr.Assets {
		switch a.Name {
		case AssetName:
			rel.AssetURL = a.URL
		case checksumAssetName:
			rel.ChecksumURL = a.URL
		}
	}
	if rel.AssetURL == "" || rel.ChecksumURL == "" {
		return Release{}, false, fmt.Errorf("в релизе %s нет %s или %s", gr.TagName, AssetName, checksumAssetName)
	}
	return rel, true, nil
}

// Download fetches the release binary, verifies its SHA-256 and returns the
// temporary file path.
func Download(ctx context.Context, rel Release, progress func(Progress)) (string, error) {
	want, err := fetchChecksum(ctx, rel.ChecksumURL)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rel.AssetURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "mvp-test-updater")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download: %s", resp.Status)
	}

	dir, err := os.MkdirTemp("", "mvp-update-*")
	if err != nil {
		return "", err
	}
	path := filepath.Join(dir, AssetName)
	f, err := os.Create(path)
	if err != nil {
		os.RemoveAll(dir)
		return "", err
	}

	total := resp.ContentLength
	hash := sha256.New()
	buf := make([]byte, 64*1024)
	var downloaded int64
	for {
		n, rerr := resp.Body.Read(buf)
		if n > 0 {
			if _, werr := f.Write(buf[:n]); werr != nil {
				f.Close()
				os.RemoveAll(dir)
				return "", werr
			}
			hash.Write(buf[:n])
			downloaded += int64(n)
			if progress != nil {
				progress(Progress{Stage: "download", Downloaded: downloaded, Total: total})
			}
		}
		if rerr == io.EOF {
			break
		}
		if rerr != nil {
			f.Close()
			os.RemoveAll(dir)
			return "", rerr
		}
	}
	if err := f.Close(); err != nil {
		os.RemoveAll(dir)
		return "", err
	}

	if progress != nil {
		progress(Progress{Stage: "verify", Downloaded: downloaded, Total: total})
	}
	got := hex.EncodeToString(hash.Sum(nil))
	if !strings.EqualFold(got, want) {
		os.RemoveAll(dir)
		return "", fmt.Errorf("checksum mismatch: got %s, want %s", got, want)
	}
	return path, nil
}

func fetchChecksum(ctx context.Context, url string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "mvp-test-updater")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("checksums: %s", resp.Status)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return "", err
	}
	return lookupChecksum(data, AssetName)
}

func lookupChecksum(data []byte, name string) (string, error) {
	for _, line := range strings.Split(string(data), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		file := strings.TrimPrefix(fields[len(fields)-1], "*")
		if filepath.Base(file) == name {
			return fields[0], nil
		}
	}
	return "", fmt.Errorf("checksum for %s not found", name)
}

func newer(current, latest string) bool {
	c, l := normalizeVersion(current), normalizeVersion(latest)
	if !semver.IsValid(c) || !semver.IsValid(l) {
		return false
	}
	return semver.Compare(l, c) > 0
}

func normalizeVersion(v string) string {
	v = strings.TrimSpace(v)
	if v == "" {
		return ""
	}
	if !strings.HasPrefix(v, "v") {
		return "v" + v
	}
	return v
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return err
	}
	return out.Close()
}
