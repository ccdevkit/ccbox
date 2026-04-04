package docker

import (
	"fmt"
	"io"
	"strings"
	"testing"
)

func TestLocalImageName(t *testing.T) {
	got := LocalImageName("0.2.0", "2.1.16")
	want := "ccbox-local:0.2.0-2.1.16"
	if got != want {
		t.Fatalf("LocalImageName(\"0.2.0\", \"2.1.16\") = %q, want %q", got, want)
	}
}

func TestLocalImageName_DifferentVersions(t *testing.T) {
	got := LocalImageName("1.0.0", "3.0.0")
	want := "ccbox-local:1.0.0-3.0.0"
	if got != want {
		t.Fatalf("LocalImageName(\"1.0.0\", \"3.0.0\") = %q, want %q", got, want)
	}
}

func TestParseImageTag(t *testing.T) {
	ccbox, claude, err := ParseImageTag("0.2.0-2.1.16")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ccbox != "0.2.0" {
		t.Fatalf("ccboxVersion = %q, want %q", ccbox, "0.2.0")
	}
	if claude != "2.1.16" {
		t.Fatalf("claudeVersion = %q, want %q", claude, "2.1.16")
	}
}

func TestParseImageTag_InvalidNoHyphen(t *testing.T) {
	_, _, err := ParseImageTag("0.2.0")
	if err == nil {
		t.Fatal("expected error for tag without hyphen separator")
	}
}

func TestParseImageTag_InvalidEmptyParts(t *testing.T) {
	_, _, err := ParseImageTag("-2.1.16")
	if err == nil {
		t.Fatal("expected error for tag with empty ccbox version")
	}

	_, _, err = ParseImageTag("0.2.0-")
	if err == nil {
		t.Fatal("expected error for tag with empty claude version")
	}
}

func TestParseImageTag_MultipleHyphens(t *testing.T) {
	// Tag "0.2.0-2.1.16-beta" should split as ccbox="0.2.0", claude="2.1.16-beta"
	ccbox, claude, err := ParseImageTag("0.2.0-2.1.16-beta")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ccbox != "0.2.0" {
		t.Fatalf("ccboxVersion = %q, want %q", ccbox, "0.2.0")
	}
	if claude != "2.1.16-beta" {
		t.Fatalf("claudeVersion = %q, want %q", claude, "2.1.16-beta")
	}
}

// mockImageManager records calls to ImageManager methods for testing.
type mockImageManager struct {
	existingImages map[string]bool
	builtImages    []string
	removedImages  []string
	listedPrefix   string
	listResult     []string
	buildErr       error
	removeErr      error
	listErr        error
}

func newMockImageManager() *mockImageManager {
	return &mockImageManager{
		existingImages: make(map[string]bool),
	}
}

func (m *mockImageManager) ImageExists(imageName string) (bool, error) {
	return m.existingImages[imageName], nil
}

func (m *mockImageManager) BuildImage(imageName string, dockerfile string, context string) error {
	if m.buildErr != nil {
		return m.buildErr
	}
	m.builtImages = append(m.builtImages, imageName)
	m.existingImages[imageName] = true
	return nil
}

func (m *mockImageManager) RemoveImage(imageName string) error {
	if m.removeErr != nil {
		return m.removeErr
	}
	m.removedImages = append(m.removedImages, imageName)
	delete(m.existingImages, imageName)
	return nil
}

func (m *mockImageManager) ListImages(prefix string) ([]string, error) {
	m.listedPrefix = prefix
	if m.listErr != nil {
		return nil, m.listErr
	}
	return m.listResult, nil
}

func TestEnsureLocalImage_ExistsAndMatches(t *testing.T) {
	mgr := newMockImageManager()
	targetImage := LocalImageName("0.2.0", "2.1.16")
	mgr.existingImages[targetImage] = true

	err := EnsureLocalImage("0.2.0", "2.1.16", false, mgr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(mgr.builtImages) != 0 {
		t.Fatalf("expected no build, got %v", mgr.builtImages)
	}
	if len(mgr.removedImages) != 0 {
		t.Fatalf("expected no removal, got %v", mgr.removedImages)
	}
}

func TestEnsureLocalImage_Missing(t *testing.T) {
	mgr := newMockImageManager()
	// No images exist, listResult empty
	mgr.listResult = nil

	err := EnsureLocalImage("0.2.0", "2.1.16", false, mgr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	targetImage := LocalImageName("0.2.0", "2.1.16")
	if len(mgr.builtImages) != 1 || mgr.builtImages[0] != targetImage {
		t.Fatalf("expected build of %q, got %v", targetImage, mgr.builtImages)
	}
}

func TestEnsureLocalImage_Mismatch_Rebuild(t *testing.T) {
	mgr := newMockImageManager()
	oldImage := LocalImageName("0.1.0", "2.0.0")
	mgr.existingImages[oldImage] = true
	// List returns the old image as an auto-update image
	mgr.listResult = []string{oldImage}

	err := EnsureLocalImage("0.2.0", "2.1.16", false, mgr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should have built the new image
	targetImage := LocalImageName("0.2.0", "2.1.16")
	if len(mgr.builtImages) != 1 || mgr.builtImages[0] != targetImage {
		t.Fatalf("expected build of %q, got %v", targetImage, mgr.builtImages)
	}
	// Should have removed the old auto-update image (FR-037)
	if len(mgr.removedImages) != 1 || mgr.removedImages[0] != oldImage {
		t.Fatalf("expected removal of %q, got %v", oldImage, mgr.removedImages)
	}
}

func TestEnsureLocalImage_FR037_RemovePreviousAutoUpdate(t *testing.T) {
	mgr := newMockImageManager()
	// Target image doesn't exist yet, but there are old auto-update images
	oldImage1 := "ccbox-local:0.1.0-2.0.0"
	oldImage2 := "ccbox-local:0.1.5-2.0.5"
	mgr.listResult = []string{oldImage1, oldImage2}

	err := EnsureLocalImage("0.2.0", "2.1.16", false, mgr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should have built the new image
	targetImage := LocalImageName("0.2.0", "2.1.16")
	if len(mgr.builtImages) != 1 || mgr.builtImages[0] != targetImage {
		t.Fatalf("expected build of %q, got %v", targetImage, mgr.builtImages)
	}
	// Should have removed all old auto-update images
	if len(mgr.removedImages) != 2 {
		t.Fatalf("expected 2 removals, got %v", mgr.removedImages)
	}
}

func TestEnsureLocalImage_BuildError(t *testing.T) {
	mgr := newMockImageManager()
	mgr.listResult = nil
	mgr.buildErr = fmt.Errorf("build failed")

	err := EnsureLocalImage("0.2.0", "2.1.16", false, mgr)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "build failed") {
		t.Fatalf("expected build error, got %v", err)
	}
}

func TestEnsureLocalImage_DockerfilePassedToBuild(t *testing.T) {
	mgr := newMockImageManager()
	mgr.listResult = nil

	err := EnsureLocalImage("0.2.0", "2.1.16", false, mgr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Verify a build was triggered (dockerfile content is passed to BuildImage)
	if len(mgr.builtImages) != 1 {
		t.Fatalf("expected exactly 1 build, got %d", len(mgr.builtImages))
	}
}

func TestPinnedImageName(t *testing.T) {
	got := PinnedImageName("0.2.0", "2.1.16")
	want := "ccbox-local:pinned-0.2.0-2.1.16"
	if got != want {
		t.Fatalf("PinnedImageName(\"0.2.0\", \"2.1.16\") = %q, want %q", got, want)
	}
}

func TestPinnedImageName_DiffersFromAutoUpdate(t *testing.T) {
	pinned := PinnedImageName("0.2.0", "2.1.16")
	autoUpdate := LocalImageName("0.2.0", "2.1.16")
	if pinned == autoUpdate {
		t.Fatalf("pinned and auto-update image names must differ, both are %q", pinned)
	}
}

func TestEnsureLocalImage_Pinned_SkipsCleanup(t *testing.T) {
	mgr := newMockImageManager()
	// Old auto-update image exists
	oldImage := LocalImageName("0.1.0", "2.0.0")
	mgr.existingImages[oldImage] = true
	mgr.listResult = []string{oldImage}

	// Build a pinned image — should NOT remove the old auto-update image
	err := EnsureLocalImage("0.2.0", "2.1.16", true, mgr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	targetImage := PinnedImageName("0.2.0", "2.1.16")
	if len(mgr.builtImages) != 1 || mgr.builtImages[0] != targetImage {
		t.Fatalf("expected build of %q, got %v", targetImage, mgr.builtImages)
	}
	// Pinned builds must NOT remove old images (FR-037)
	if len(mgr.removedImages) != 0 {
		t.Fatalf("expected no removal for pinned build, got %v", mgr.removedImages)
	}
}

func TestEnsureLocalImage_Pinned_ExistingPinnedImage(t *testing.T) {
	mgr := newMockImageManager()
	targetImage := PinnedImageName("0.2.0", "2.1.16")
	mgr.existingImages[targetImage] = true

	err := EnsureLocalImage("0.2.0", "2.1.16", true, mgr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(mgr.builtImages) != 0 {
		t.Fatalf("expected no build for existing pinned image, got %v", mgr.builtImages)
	}
}

func TestEnsureLocalImage_AutoUpdate_DoesNotRemovePinnedImages(t *testing.T) {
	mgr := newMockImageManager()
	// A pinned image and an old auto-update image both exist
	pinnedImage := "ccbox-local:pinned-0.1.0-2.0.0"
	oldAutoImage := "ccbox-local:0.1.0-2.0.0"
	mgr.listResult = []string{pinnedImage, oldAutoImage}

	// Auto-update build should remove old auto-update but NOT the pinned image
	err := EnsureLocalImage("0.2.0", "2.1.16", false, mgr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should only remove the old auto-update image, not the pinned one
	if len(mgr.removedImages) != 1 || mgr.removedImages[0] != oldAutoImage {
		t.Fatalf("expected removal of only %q, got %v", oldAutoImage, mgr.removedImages)
	}
}

func TestVersionMismatch(t *testing.T) {
	tests := []struct {
		name           string
		tag            string
		wantCCBox      string
		wantClaude     string
		currentCCBox   string
		currentClaude  string
		expectMismatch bool
	}{
		{
			name:           "exact match",
			tag:            "0.2.0-2.1.16",
			currentCCBox:   "0.2.0",
			currentClaude:  "2.1.16",
			expectMismatch: false,
		},
		{
			name:           "ccbox version mismatch",
			tag:            "0.1.0-2.1.16",
			currentCCBox:   "0.2.0",
			currentClaude:  "2.1.16",
			expectMismatch: true,
		},
		{
			name:           "claude version mismatch",
			tag:            "0.2.0-2.0.0",
			currentCCBox:   "0.2.0",
			currentClaude:  "2.1.16",
			expectMismatch: true,
		},
		{
			name:           "both mismatch",
			tag:            "0.1.0-2.0.0",
			currentCCBox:   "0.2.0",
			currentClaude:  "2.1.16",
			expectMismatch: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ImageTagMismatch(tt.tag, tt.currentCCBox, tt.currentClaude)
			if got != tt.expectMismatch {
				t.Fatalf("ImageTagMismatch(%q, %q, %q) = %v, want %v",
					tt.tag, tt.currentCCBox, tt.currentClaude, got, tt.expectMismatch)
			}
		})
	}
}

func TestCleanImages_PreservesLatestAutoUpdate(t *testing.T) {
	mgr := newMockImageManager()
	// Two auto-update images and one pinned image
	mgr.listResult = []string{
		"ccbox-local:0.1.0-2.0.0",
		"ccbox-local:pinned-0.1.0-2.0.0",
		"ccbox-local:0.2.0-2.1.16",
	}

	err := CleanImages(mgr, io.Discard)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Latest auto-update (last non-pinned) should be preserved
	// Old auto-update and pinned should be removed
	if len(mgr.removedImages) != 2 {
		t.Fatalf("expected 2 removals, got %v", mgr.removedImages)
	}
	for _, img := range mgr.removedImages {
		if img == "ccbox-local:0.2.0-2.1.16" {
			t.Fatalf("latest auto-update image should not be removed")
		}
	}
}

func TestCleanImages_RemovesAllOtherImages(t *testing.T) {
	mgr := newMockImageManager()
	mgr.listResult = []string{
		"ccbox-local:0.1.0-2.0.0",
		"ccbox-local:0.1.5-2.0.5",
		"ccbox-local:pinned-0.1.0-2.0.0",
		"ccbox-local:0.2.0-2.1.16",
	}

	err := CleanImages(mgr, io.Discard)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should remove everything except the latest auto-update
	removed := make(map[string]bool)
	for _, img := range mgr.removedImages {
		removed[img] = true
	}
	if !removed["ccbox-local:0.1.0-2.0.0"] {
		t.Fatal("expected old auto-update 0.1.0-2.0.0 to be removed")
	}
	if !removed["ccbox-local:0.1.5-2.0.5"] {
		t.Fatal("expected old auto-update 0.1.5-2.0.5 to be removed")
	}
	if !removed["ccbox-local:pinned-0.1.0-2.0.0"] {
		t.Fatal("expected pinned image to be removed")
	}
	if removed["ccbox-local:0.2.0-2.1.16"] {
		t.Fatal("latest auto-update should NOT be removed")
	}
}

func TestCleanAllImages_RemovesEverything(t *testing.T) {
	mgr := newMockImageManager()
	mgr.listResult = []string{
		"ccbox-local:0.1.0-2.1.0",
		"ccbox-local:0.2.0-2.1.16",
		"ccbox-local:pinned-0.2.0-2.1.16",
	}

	err := CleanAllImages(mgr, io.Discard)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(mgr.removedImages) != 3 {
		t.Fatalf("expected 3 removals, got %d: %v", len(mgr.removedImages), mgr.removedImages)
	}
}

func TestCleanImages_EmptyImageList(t *testing.T) {
	mgr := newMockImageManager()
	mgr.listResult = nil

	err := CleanImages(mgr, io.Discard)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(mgr.removedImages) != 0 {
		t.Fatalf("expected no removals for empty list, got %v", mgr.removedImages)
	}
}
