package sshdrop_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReleaseIdentityIsCanonical(t *testing.T) {
	staleModulePath := "github.com/dineshpandiyan" + "/ssh-drop"
	goMod := readFile(t, "go.mod")
	if !strings.Contains(goMod, "module github.com/flexdinesh/ssh-drop") {
		t.Fatalf("go.mod should use github.com/flexdinesh/ssh-drop:\n%s", goMod)
	}

	readme := readFile(t, "README.md")
	if !strings.Contains(readme, "go install github.com/flexdinesh/ssh-drop/cmd/ssh-drop@latest") {
		t.Fatalf("README should document the canonical Go install path")
	}

	for _, path := range goFiles(t, ".") {
		contents := readFile(t, path)
		if strings.Contains(contents, staleModulePath) {
			t.Fatalf("%s still references %s", path, staleModulePath)
		}
	}
}

func TestGoReleaserPackagesSnapshotsForSupportedPlatforms(t *testing.T) {
	config := readFile(t, ".goreleaser.yaml")
	for _, want := range []string{
		"project_name: ssh-drop",
		"main: ./cmd/ssh-drop",
		"binary: ssh-drop",
		"CGO_ENABLED=0",
		"darwin",
		"linux",
		"amd64",
		"arm64",
		"-X main.version={{.Version}}",
		"checksums.txt",
	} {
		if !strings.Contains(config, want) {
			t.Fatalf(".goreleaser.yaml should contain %q", want)
		}
	}

	ci := readFile(t, ".github/workflows/ci.yml")
	for _, want := range []string{
		"go test ./...",
		"go build ./cmd/ssh-drop",
		"args: release --snapshot --clean",
	} {
		if !strings.Contains(ci, want) {
			t.Fatalf("CI workflow should contain %q", want)
		}
	}
}

func TestStableReleaseWorkflowPublishesSemverTags(t *testing.T) {
	workflow := readFile(t, ".github/workflows/release.yml")
	for _, want := range []string{
		"workflow_dispatch",
		"contents: write",
		"ref: main",
		"fetch-depth: 0",
		"go test ./...",
		"git tag --points-at HEAD",
		"git tag -l 'v0.1.*'",
		"create=false",
		"next=\"v0.1.0\"",
		"git push origin",
		"steps.tag.outputs.create",
		"args: release --clean",
	} {
		if !strings.Contains(workflow, want) {
			t.Fatalf("release workflow should contain %q", want)
		}
	}
}

func TestGoReleaserPublishesHomebrewTapPullRequest(t *testing.T) {
	config := readFile(t, ".goreleaser.yaml")
	for _, want := range []string{
		"release:",
		"replace_existing_artifacts: true",
		"brews:",
		"name: ssh-drop",
		"owner: flexdinesh",
		"name: homebrew-tap",
		"token: \"{{ .Env.HOMEBREW_TAP_TOKEN }}\"",
		"branch: \"ssh-drop-{{ .Tag }}\"",
		"pull_request:",
		"enabled: true",
		"dependencies:",
		"- rsync",
		"assert_match version.to_s, shell_output(\"#{bin}/ssh-drop --version\")",
	} {
		if !strings.Contains(config, want) {
			t.Fatalf(".goreleaser.yaml should contain %q", want)
		}
	}

	workflow := readFile(t, ".github/workflows/release.yml")
	for _, want := range []string{
		"version: v2.9.0",
		"HOMEBREW_TAP_TOKEN: ${{ secrets.HOMEBREW_TAP_TOKEN }}",
	} {
		if !strings.Contains(workflow, want) {
			t.Fatalf("release workflow should contain %q", want)
		}
	}
}

func TestReleaseDocsExplainHomebrewChannel(t *testing.T) {
	readme := readFile(t, "README.md")
	for _, want := range []string{
		"brew install flexdinesh/tap/ssh-drop",
		"go install github.com/flexdinesh/ssh-drop/cmd/ssh-drop@latest",
		"wl-clipboard",
		"xclip",
	} {
		if !strings.Contains(readme, want) {
			t.Fatalf("README should contain %q", want)
		}
	}

	releaseDoc := readFile(t, "docs/release.md")
	for _, want := range []string{
		"HOMEBREW_TAP_TOKEN",
		"v0.1.0",
		"brew install flexdinesh/tap/ssh-drop",
		"goreleaser release --snapshot --clean",
		"Do not create a moving `latest` tag",
	} {
		if !strings.Contains(releaseDoc, want) {
			t.Fatalf("docs/release.md should contain %q", want)
		}
	}

	adr := readFile(t, "docs/adr/0002-homebrew-tap-releases.md")
	if !strings.Contains(adr, "flexdinesh/homebrew-tap") {
		t.Fatalf("Homebrew release ADR should record the custom tap")
	}
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(contents)
}

func goFiles(t *testing.T, root string) []string {
	t.Helper()
	var paths []string
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			switch path {
			case ".git", ".scratch", "bin":
				return filepath.SkipDir
			}
			return nil
		}
		if strings.HasSuffix(path, ".go") {
			paths = append(paths, path)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk %s: %v", root, err)
	}
	return paths
}
