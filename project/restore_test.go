package project_test

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/lib-x/lzc-toolkit-go/build"
	"github.com/lib-x/lzc-toolkit-go/project"
)

func TestRestoreBuildableV2Project(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	writeRestoreFile(t, root, "lzc-build.yml", "manifest: lzc-manifest.yml\ncontentdir: content\n")
	writeRestoreFile(t, root, "lzc-manifest.yml", "package: community.lazycat.app.restore\nversion: 1.0.0\napplication:\n  subdomain: restore\n  # upstream: docker.io/example/app:1.2.3\n  image: registry.lazycat.cloud/example/app:abcdef\n")
	writeRestoreFile(t, root, "package.yml", "package: community.lazycat.app.restore\nversion: 1.0.0\nname: Restore\n")
	writeRestoreFile(t, root, "content/run.sh", "#!/bin/sh\necho ok\n")
	var packageBytes bytes.Buffer
	if _, err := build.Build(ctx, &packageBytes, build.Request{Root: root, Strict: true}); err != nil {
		t.Fatal(err)
	}
	destination := filepath.Join(t.TempDir(), "restored")
	result, err := project.Restore(ctx, bytes.NewReader(packageBytes.Bytes()), destination)
	if err != nil {
		t.Fatal(err)
	}
	if result.Inspection.Package.Package != "community.lazycat.app.restore" || !result.Inspection.Application.HasImage {
		t.Fatalf("inspection = %#v", result.Inspection)
	}
	if _, err := os.Stat(filepath.Join(destination, "content", "run.sh")); err != nil {
		t.Fatal(err)
	}
}

func TestRestoreCanPreferUpstreamImageComments(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	writeRestoreFile(t, root, "lzc-build.yml", "manifest: lzc-manifest.yml\n")
	writeRestoreFile(t, root, "lzc-manifest.yml", "package: community.lazycat.app.restore-upstream\nversion: 1.0.0\napplication:\n  subdomain: restore-upstream\n  # upstream: ghcr.io/bestruirui/octopus:v0.12.1\n  image: registry.lazycat.cloud/czyt/bestruirui/octopus:adbebabecb16a621\n")
	var packageBytes bytes.Buffer
	if _, err := build.Build(ctx, &packageBytes, build.Request{Root: root}); err != nil {
		t.Fatal(err)
	}
	destination := filepath.Join(t.TempDir(), "restored")
	if _, err := project.RestoreWithOptions(ctx, bytes.NewReader(packageBytes.Bytes()), destination, project.RestoreOptions{PreferUpstreamImages: true}); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(destination, "lzc-manifest.yml"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(data, []byte("image: ghcr.io/bestruirui/octopus:v0.12.1")) {
		t.Fatalf("upstream image was not restored:\n%s", data)
	}
}

func writeRestoreFile(t *testing.T, root, name, contents string) {
	t.Helper()
	filename := filepath.Join(root, filepath.FromSlash(name))
	if err := os.MkdirAll(filepath.Dir(filename), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filename, []byte(contents), 0o644); err != nil {
		t.Fatal(err)
	}
}
