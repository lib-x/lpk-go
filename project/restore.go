package project

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	lpkgo "github.com/lib-x/lzc-toolkit-go"
	"github.com/lib-x/lzc-toolkit-go/archive"
	"github.com/lib-x/lzc-toolkit-go/lpk"
	"github.com/lib-x/lzc-toolkit-go/manifest"
)

// RestoreResult describes a project reconstructed from a published LPK.
type RestoreResult struct {
	Root       string
	Inspection Inspection
}

// RestoreOptions controls reconstruction of a published package.
type RestoreOptions struct {
	// PreferUpstreamImages replaces image fields with the upstream references
	// preserved in `# upstream:` comments. Fields without such a comment stay
	// unchanged because a LazyCat registry reference cannot be reverse mapped.
	PreferUpstreamImages bool
}

// Restore extracts an LPK into a buildable project directory. v2 content.tar
// is unpacked into content/, while package metadata and the manifest are
// converted to the source-project filenames expected by build.Build.
// The destination must be empty or not exist.
func Restore(ctx context.Context, src io.Reader, destination string) (RestoreResult, error) {
	return RestoreWithOptions(ctx, src, destination, RestoreOptions{})
}

// RestoreWithOptions is Restore with image-source reconstruction controls.
func RestoreWithOptions(ctx context.Context, src io.Reader, destination string, options RestoreOptions) (RestoreResult, error) {
	if ctx == nil || src == nil || destination == "" {
		return RestoreResult{}, restoreError(lpkgo.CodeInvalidArgument, "project.restore", errors.New("context, source, and destination are required"))
	}
	if err := ctx.Err(); err != nil {
		return RestoreResult{}, restoreError(lpkgo.CodeCancelled, "project.restore", err)
	}
	root, err := filepath.Abs(destination)
	if err != nil {
		return RestoreResult{}, restoreError(lpkgo.CodeInvalidArgument, "project.restore", err)
	}
	if info, statErr := os.Stat(root); statErr == nil {
		if !info.IsDir() {
			return RestoreResult{}, restoreError(lpkgo.CodeInvalidArgument, "project.restore", errors.New("destination is not a directory"))
		}
		entries, readErr := os.ReadDir(root)
		if readErr != nil {
			return RestoreResult{}, restoreError(lpkgo.CodeCommandFailed, "project.restore", readErr)
		}
		if len(entries) != 0 {
			return RestoreResult{}, restoreError(lpkgo.CodeConflict, "project.restore", errors.New("destination is not empty"))
		}
	} else if !errors.Is(statErr, os.ErrNotExist) {
		return RestoreResult{}, restoreError(lpkgo.CodeCommandFailed, "project.restore", statErr)
	} else if err := os.MkdirAll(root, 0o755); err != nil {
		return RestoreResult{}, restoreError(lpkgo.CodeCommandFailed, "project.restore", err)
	}

	reader, err := lpk.Open(ctx, src)
	if err != nil {
		return RestoreResult{}, err
	}
	defer reader.Close()
	staging, err := os.MkdirTemp("", "lzc-toolkit-restore-*")
	if err != nil {
		return RestoreResult{}, restoreError(lpkgo.CodeCommandFailed, "project.restore", err)
	}
	defer os.RemoveAll(staging)
	if err := reader.Extract(ctx, staging); err != nil {
		return RestoreResult{}, err
	}
	if err := copyIfRegular(filepath.Join(staging, "manifest.yml"), filepath.Join(root, "lzc-manifest.yml")); err != nil {
		return RestoreResult{}, err
	}
	if options.PreferUpstreamImages {
		if err := restoreUpstreamImages(filepath.Join(root, "lzc-manifest.yml")); err != nil {
			return RestoreResult{}, err
		}
	}
	if err := copyIfRegular(filepath.Join(staging, "package.yml"), filepath.Join(root, "package.yml")); err != nil {
		return RestoreResult{}, err
	}
	if err := copyIfRegular(filepath.Join(staging, "icon.png"), filepath.Join(root, "icon.png")); err != nil {
		return RestoreResult{}, err
	}
	contentArchive := filepath.Join(staging, "content.tar")
	hasContent := false
	if info, statErr := os.Stat(contentArchive); statErr == nil && info.Mode().IsRegular() {
		hasContent = true
		contentRoot := filepath.Join(root, "content")
		if err := os.MkdirAll(contentRoot, 0o755); err != nil {
			return RestoreResult{}, restoreError(lpkgo.CodeCommandFailed, "project.restore", err)
		}
		contentReader, openErr := archive.OpenFile(ctx, contentArchive)
		if openErr != nil {
			return RestoreResult{}, openErr
		}
		extractErr := contentReader.Extract(ctx, contentRoot)
		closeErr := contentReader.Close()
		if extractErr != nil {
			return RestoreResult{}, extractErr
		}
		if closeErr != nil {
			return RestoreResult{}, closeErr
		}
	}
	config := "manifest: lzc-manifest.yml\nicon: icon.png\n"
	if hasContent {
		config = "manifest: lzc-manifest.yml\ncontentdir: content\nicon: icon.png\n"
	}
	if err := os.WriteFile(filepath.Join(root, "lzc-build.yml"), []byte(config), 0o644); err != nil {
		return RestoreResult{}, restoreError(lpkgo.CodeCommandFailed, "project.restore", err)
	}
	inspection, err := Inspect(ctx, InspectRequest{Root: root})
	if err != nil {
		return RestoreResult{}, fmt.Errorf("%w: restored project inspection failed", err)
	}
	return RestoreResult{Root: root, Inspection: inspection}, nil
}

func restoreUpstreamImages(filename string) error {
	data, err := os.ReadFile(filename)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return restoreError(lpkgo.CodeCommandFailed, "project.restore.images", err)
	}
	analysis, err := manifest.Analyze(data)
	if err != nil {
		return err
	}
	document := analysis.Document()
	for _, image := range analysis.Summary().Images {
		if !image.Editable || strings.TrimSpace(image.UpstreamRef) == "" {
			continue
		}
		path := strings.Split(image.Target, ".")
		if err := document.Set(image.UpstreamRef, path...); err != nil {
			return err
		}
	}
	data, err = document.Bytes()
	if err != nil {
		return err
	}
	if err := os.WriteFile(filename, data, 0o644); err != nil {
		return restoreError(lpkgo.CodeCommandFailed, "project.restore.images", err)
	}
	return nil
}

func copyIfRegular(source, destination string) error {
	data, err := os.ReadFile(source)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return restoreError(lpkgo.CodeCommandFailed, "project.restore", err)
	}
	if err := os.WriteFile(destination, data, 0o644); err != nil {
		return restoreError(lpkgo.CodeCommandFailed, "project.restore", err)
	}
	return nil
}

func restoreError(code lpkgo.Code, op string, cause error) error {
	return &lpkgo.Error{Code: code, Op: op, Cause: cause}
}
