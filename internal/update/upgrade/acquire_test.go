package upgrade

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
)

func acquisitionArchive(t *testing.T, headers ...tar.Header) []byte {
	t.Helper()
	var out bytes.Buffer
	gz := gzip.NewWriter(&out)
	tw := tar.NewWriter(gz)
	for _, h := range headers {
		if err := tw.WriteHeader(&h); err != nil {
			t.Fatal(err)
		}
		if h.Size > 0 {
			if _, err := tw.Write(bytes.Repeat([]byte("x"), int(h.Size))); err != nil {
				t.Fatal(err)
			}
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}
	return out.Bytes()
}

// Write only the tar header: the extractor must reject its declared size before reading the payload.
func acquisitionOversizedHeader(t *testing.T) []byte {
	t.Helper()
	var out bytes.Buffer
	gz := gzip.NewWriter(&out)
	tw := tar.NewWriter(gz)
	if err := tw.WriteHeader(&tar.Header{Name: "gentle-ai", Mode: 0755, Size: maxAcquiredBinaryBytes + 1}); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}
	return out.Bytes()
}

func TestAcquireReleaseBinary(t *testing.T) {
	scenarios := []string{
		"valid", "tampered archive", "wrong tag", "wrong repo", "wrong key", "missing key",
		"duplicate checksum", "missing checksum", "checksum HTTP error", "signature HTTP error",
		"archive HTTP error", "oversized content length", "duplicate regular", "missing binary",
		"symlink after regular", "hardlink after regular", "malformed gzip", "truncated gzip",
		"oversized member", "canceled during request",
	}
	for _, scenario := range scenarios {
		t.Run(scenario, func(t *testing.T) {
			key := useTestReleaseKey(t)
			regular := tar.Header{Name: "gentle-ai", Mode: 0755, Size: 3}
			archive := acquisitionArchive(t, regular)
			switch scenario {
			case "duplicate regular":
				archive = acquisitionArchive(t, regular, regular)
			case "missing binary":
				archive = acquisitionArchive(t, tar.Header{Name: "other", Mode: 0755, Size: 3})
			case "symlink after regular":
				archive = acquisitionArchive(t, regular, tar.Header{Name: "gentle-ai", Typeflag: tar.TypeSymlink, Linkname: "other"})
			case "hardlink after regular":
				archive = acquisitionArchive(t, regular, tar.Header{Name: "gentle-ai", Typeflag: tar.TypeLink, Linkname: "other"})
			case "malformed gzip":
				archive = []byte("not gzip")
			case "truncated gzip":
				archive = archive[:len(archive)-4]
			case "oversized member":
				archive = acquisitionOversizedHeader(t)
			}
			name := "gentle-ai_5.0.0_darwin_arm64.tar.gz"
			manifest := []byte(fmt.Sprintf("%x  %s\n", sha256.Sum256(archive), name))
			if scenario == "duplicate checksum" {
				manifest = append(bytes.Clone(manifest), manifest...)
			}
			if scenario == "missing checksum" {
				manifest = []byte(fmt.Sprintf("%x  other.tar.gz\n", sha256.Sum256(archive)))
			}
			tag, repo := "5.0.0", "gentle-ai"
			if scenario == "wrong tag" {
				tag = "5.0.1"
			}
			if scenario == "wrong repo" {
				repo = "other"
			}
			if scenario == "wrong key" {
				_, key = minisignKeyFromDomain(t, testMinisignKeyDomain+" alternate")
			}
			signature := signTestManifest(t, key, manifest, "Gentleman-Programming", repo, tag)
			if scenario == "missing key" {
				releaseMinisignPublicKeys = unsetReleaseMinisignPublicKeys
			}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			var requests, assetRequests atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requests.Add(1)
				switch r.URL.Path {
				case "/checksums.txt":
					if scenario == "canceled during request" {
						cancel()
						return
					}
					if scenario == "checksum HTTP error" {
						http.Error(w, "failure", http.StatusBadGateway)
						return
					}
					_, _ = w.Write(manifest)
				case "/checksums.txt.minisig":
					if scenario == "signature HTTP error" {
						http.Error(w, "failure", http.StatusBadGateway)
						return
					}
					_, _ = w.Write(signature)
				case "/" + name:
					assetRequests.Add(1)
					if scenario == "archive HTTP error" {
						http.Error(w, "failure", http.StatusBadGateway)
						return
					}
					if scenario == "oversized content length" {
						w.Header().Set("Content-Length", fmt.Sprint(maxReleaseArchiveBytes+1))
						w.WriteHeader(http.StatusOK)
						return
					}
					if scenario == "tampered archive" {
						_, _ = w.Write([]byte("tampered"))
						return
					}
					_, _ = w.Write(archive)
				default:
					t.Errorf("unexpected request: %s", r.URL.Path)
					http.NotFound(w, r)
				}
			}))
			defer server.Close()
			oldAsset, oldChecksum, oldSignature, oldLookup := resolveAssetURLFn, resolveChecksumURLFn, resolveSignatureURLFn, lookPathFn
			t.Cleanup(func() {
				resolveAssetURLFn, resolveChecksumURLFn, resolveSignatureURLFn, lookPathFn = oldAsset, oldChecksum, oldSignature, oldLookup
			})
			resolveAssetURLFn = func(owner, repo, version, goos, goarch string) string {
				if owner != "Gentleman-Programming" || repo != "gentle-ai" || version != "5.0.0" || goos != "darwin" || goarch != "arm64" {
					t.Error("wrong release selection")
				}
				return server.URL + "/" + name
			}
			resolveChecksumURLFn = func(_, _, _ string) string { return server.URL + "/checksums.txt" }
			resolveSignatureURLFn = func(_, _, _ string) string { return server.URL + "/checksums.txt.minisig" }
			installed := filepath.Join(t.TempDir(), "gentle-ai")
			if err := os.WriteFile(installed, []byte("original"), 0755); err != nil {
				t.Fatal(err)
			}
			lookPathFn = func(string) (string, error) {
				t.Error("unexpected PATH lookup")
				return installed, nil
			}
			got, err := AcquireReleaseBinary(ctx, "Gentleman-Programming", "gentle-ai", "5.0.0", "darwin", "arm64")
			if scenario == "valid" {
				if err != nil || string(got) != "xxx" {
					t.Fatalf("acquire = %q, %v", got, err)
				}
			} else if err == nil || len(got) != 0 {
				t.Fatalf("rejected acquisition = %q, %v", got, err)
			}
			switch scenario {
			case "missing key", "wrong tag", "wrong repo", "wrong key", "duplicate checksum", "missing checksum", "checksum HTTP error", "signature HTTP error", "canceled during request":
				if assetRequests.Load() != 0 {
					t.Fatal("archive requested before authentication and checksum selection")
				}
			}
			if scenario == "missing key" && requests.Load() != 0 {
				t.Fatalf("missing key made %d requests", requests.Load())
			}
			if scenario == "canceled during request" && requests.Load() == 0 {
				t.Fatal("request did not reach handler")
			}
			if scenario == "oversized content length" && (err == nil || !strings.Contains(err.Error(), "exceeds")) {
				t.Fatalf("expected size limit error: %v", err)
			}
			data, readErr := os.ReadFile(installed)
			if readErr != nil || string(data) != "original" {
				t.Fatalf("installed binary changed: %q, %v", data, readErr)
			}
		})
	}
}

func TestAcquireReleaseBinaryRejectsEarly(t *testing.T) {
	oldChecksum, oldLookup := resolveChecksumURLFn, lookPathFn
	t.Cleanup(func() { resolveChecksumURLFn, lookPathFn = oldChecksum, oldLookup })
	resolveChecksumURLFn = func(_, _, _ string) string {
		t.Error("unexpected network resolution")
		return ""
	}
	lookPathFn = func(string) (string, error) {
		t.Error("unexpected PATH lookup")
		return "", nil
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	cases := []struct {
		name, version, goos, arch string
		ctx context.Context
	}{
		{"canceled", "5.0.0", "darwin", "arm64", ctx},
		{"unsupported OS", "5.0.0", "windows", "amd64", context.Background()},
		{"unsupported arch", "5.0.0", "linux", "386", context.Background()},
		{"dev", "dev", "darwin", "arm64", context.Background()},
		{"latest", "latest", "darwin", "arm64", context.Background()},
		{"prefix", "v5.0.0", "darwin", "arm64", context.Background()},
		{"prerelease", "5.0.0-rc.1", "darwin", "arm64", context.Background()},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := AcquireReleaseBinary(tc.ctx, "Gentleman-Programming", "gentle-ai", tc.version, tc.goos, tc.arch)
			if err == nil || len(got) != 0 {
				t.Fatalf("acquire = %q, %v", got, err)
			}
			if tc.name == "dev" || tc.name == "latest" || tc.name == "prefix" || tc.name == "prerelease" {
				if !strings.Contains(err.Error(), "exact stable semver") {
					t.Fatalf("version error: %v", err)
				}
			}
		})
	}
}

func TestAcquireArchiveBinaryRejectsUnsafeArchives(t *testing.T) {
	regular := tar.Header{Name: "gentle-ai", Mode: 0755, Size: 3}
	symlink := tar.Header{Name: "gentle-ai", Typeflag: tar.TypeSymlink, Linkname: "other"}
	hardlink := tar.Header{Name: "gentle-ai", Typeflag: tar.TypeLink, Linkname: "other"}
	valid := acquisitionArchive(t, regular)
	cases := []struct {
		name string
		archive []byte
		tarLimit, binaryLimit int64
	}{
		{"duplicate regular", acquisitionArchive(t, regular, regular), 16384, 32},
		{"symlink", acquisitionArchive(t, symlink), 16384, 32},
		{"symlink after regular", acquisitionArchive(t, regular, symlink), 16384, 32},
		{"hardlink", acquisitionArchive(t, hardlink), 16384, 32},
		{"hardlink after regular", acquisitionArchive(t, regular, hardlink), 16384, 32},
		{"oversized member", valid, 16384, 2},
		{"malformed gzip", []byte("not gzip"), 16384, 32},
		{"truncated gzip", valid[:len(valid)-4], 16384, 32},
		{"decompression limit", valid, 512, 32},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := acquireArchiveBinary(context.Background(), bytes.NewReader(tc.archive), "gentle-ai", tc.tarLimit, tc.binaryLimit)
			if err == nil || len(got) != 0 {
				t.Fatalf("extract = %q, %v", got, err)
			}
		})
	}
	got, err := acquireArchiveBinary(context.Background(), bytes.NewReader(valid), "gentle-ai", 16384, 32)
	if err != nil || string(got) != "xxx" {
		t.Fatalf("valid extract = %q, %v", got, err)
	}
}
