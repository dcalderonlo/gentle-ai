package upgrade

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

const (
	maxAcquiredBinaryBytes = 128 << 20
	maxAcquiredTarBytes    = 256 << 20
	maxAcquiredTarMembers  = 4096
)

// AcquireReleaseBinary returns authenticated bytes for one exact release and
// platform. It neither resolves PATH nor executes or replaces an installed file.
// The caller is responsible for privately staging the returned executable.
func AcquireReleaseBinary(ctx context.Context, owner, repo, version, goos, goarch string) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := validateReleaseIdentity(owner, repo, version); err != nil {
		return nil, err
	}
	if (goos != "darwin" && goos != "linux") || (goarch != "amd64" && goarch != "arm64") {
		return nil, fmt.Errorf("unsupported release platform %s/%s", goos, goarch)
	}
	if _, err := configuredReleasePublicKeys(); err != nil {
		return nil, fmt.Errorf("authenticate release: %w", err)
	}
	name := resolveArchiveName(repo, version, goos, goarch)
	manifest, err := fetchChecksums(ctx, resolveChecksumURLFn(owner, repo, version))
	if err != nil {
		return nil, fmt.Errorf("fetch release checksums: %w", err)
	}
	signature, err := fetchChecksumSignature(ctx, resolveSignatureURLFn(owner, repo, version))
	if err != nil {
		return nil, fmt.Errorf("fetch release signature: %w", err)
	}
	if err := verifyChecksumsSignature([]byte(manifest), signature, owner, repo, version); err != nil {
		return nil, fmt.Errorf("authenticate release: %w", err)
	}
	expected, err := expectedChecksumFor(manifest, name)
	if err != nil {
		return nil, err
	}
	stage, err := os.MkdirTemp("", "gentle-ai-acquire-*")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(stage)
	archive := filepath.Join(stage, "release.tar.gz")
	actual, err := downloadToFile(ctx, resolveAssetURLFn(owner, repo, version, goos, goarch), archive, maxReleaseArchiveBytes)
	if err != nil {
		return nil, err
	}
	if actual != expected {
		return nil, errors.New("release archive checksum mismatch")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	file, err := os.Open(archive)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	return acquireArchiveBinary(ctx, file, repo, maxAcquiredTarBytes, maxAcquiredBinaryBytes)
}

// acquireArchiveBinary scans the whole bounded gzip stream, including its
// trailer, before releasing bytes. No archive member is materialized on disk.
func acquireArchiveBinary(ctx context.Context, source io.Reader, name string, tarLimit, binaryLimit int64) ([]byte, error) {
	gz, err := gzip.NewReader(source)
	if err != nil {
		return nil, err
	}
	defer gz.Close()
	limited := &io.LimitedReader{R: acquireContextReader{ctx, gz}, N: tarLimit + 1}
	tr := tar.NewReader(limited)
	var binary []byte
	found := false
	for count := 0; ; count++ {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		if count >= maxAcquiredTarMembers {
			return nil, errors.New("release archive has too many members")
		}
		if header.Name != name {
			continue
		}
		if found || (header.Typeflag != tar.TypeReg && header.Typeflag != tar.TypeRegA) {
			return nil, errors.New("release binary is duplicated or not a regular file")
		}
		if header.Size <= 0 || header.Size > binaryLimit {
			return nil, errors.New("release binary exceeds size bounds")
		}
		found = true
		binary, err = io.ReadAll(io.LimitReader(tr, binaryLimit+1))
		if err != nil || int64(len(binary)) != header.Size {
			return nil, errors.New("release binary readback failed")
		}
	}
	// tar EOF may precede gzip EOF. Drain within the same budget to verify
	// checksum/truncation and count trailing decompressed bytes as well.
	if _, err := io.Copy(io.Discard, limited); err != nil {
		return nil, err
	}
	if limited.N == 0 {
		return nil, errors.New("release archive exceeds decompressed size limit")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if !found {
		return nil, errors.New("release binary not found")
	}
	return binary, nil
}

type acquireContextReader struct {
	ctx    context.Context
	reader io.Reader
}

func (r acquireContextReader) Read(p []byte) (int, error) {
	if err := r.ctx.Err(); err != nil {
		return 0, err
	}
	return r.reader.Read(p)
}
