package updater

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestNewer(t *testing.T) {
	cases := []struct {
		current, latest string
		want            bool
	}{
		{"0.5.0", "v0.5.1", true},
		{"v0.5.1", "0.5.1", false},
		{"0.6.0", "0.5.9", false},
		{"dev", "1.0.0", false},
	}
	for _, c := range cases {
		if got := newer(c.current, c.latest); got != c.want {
			t.Errorf("newer(%q, %q) = %v, want %v", c.current, c.latest, got, c.want)
		}
	}
}

func TestLookupChecksum(t *testing.T) {
	data := []byte("abc123  other.exe\ndef456  *" + AssetName + "\n")
	got, err := lookupChecksum(data, AssetName)
	if err != nil || got != "def456" {
		t.Fatalf("got %q err %v", got, err)
	}
}

func TestCheck(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/repos/"+repo+"/releases/latest", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `{"tag_name":"v1.2.3","body":"notes","assets":[
			{"name":%q,"browser_download_url":"https://example.test/%s"},
			{"name":%q,"browser_download_url":"https://example.test/%s"}]}`,
			AssetName, AssetName, checksumAssetName, checksumAssetName)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()
	old := apiBase
	apiBase = srv.URL
	defer func() { apiBase = old }()

	rel, ok, err := Check(context.Background(), "1.2.2")
	if err != nil {
		t.Fatal(err)
	}
	if !ok || rel.Version != "1.2.3" || rel.AssetURL == "" || rel.ChecksumURL == "" {
		t.Fatalf("release %+v ok %v", rel, ok)
	}
	if _, ok, err := Check(context.Background(), "1.2.3"); err != nil || ok {
		t.Fatalf("same version: ok %v err %v", ok, err)
	}
}

func TestDownloadVerifiesChecksum(t *testing.T) {
	body := []byte("fake exe bytes")
	sum := sha256.Sum256(body)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/asset":
			_, _ = w.Write(body)
		case "/checksums":
			fmt.Fprintf(w, "%s  %s\n", hex.EncodeToString(sum[:]), AssetName)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	path, err := Download(context.Background(), Release{
		AssetURL:    srv.URL + "/asset",
		ChecksumURL: srv.URL + "/checksums",
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(filepath.Dir(path))
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, body) {
		t.Fatalf("downloaded %q", got)
	}
}
