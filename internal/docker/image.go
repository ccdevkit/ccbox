package docker

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/ccdevkit/ccbox/internal/constants"
)

// ImageManager abstracts Docker image operations for testability.
type ImageManager interface {
	ImageExists(imageName string) (bool, error)
	BuildImage(imageName string, dockerfile string, context string) error
	RemoveImage(imageName string) error
	ListImages(prefix string) ([]string, error)
}

// EnsureLocalImage ensures the correct local Docker image exists.
// It checks for the target image, builds it if missing, and on rebuild
// removes any previous auto-update images (FR-037).
// When pinned is true, a pinned image name is used and old images are
// NOT auto-removed (FR-037: only auto-update images are auto-cleaned).
func EnsureLocalImage(ccboxVersion, claudeVersion string, pinned bool, mgr ImageManager) error {
	targetImage := LocalImageName(ccboxVersion, claudeVersion)
	if pinned {
		targetImage = PinnedImageName(ccboxVersion, claudeVersion)
	}

	exists, err := mgr.ImageExists(targetImage)
	if err != nil {
		return fmt.Errorf("checking image %s: %w", targetImage, err)
	}
	if exists {
		return nil
	}

	status := statusWriter(os.Stderr)

	// Remove old auto-update images before building, but only for auto-update builds.
	// Pinned builds skip cleanup (FR-037). Auto-update cleanup skips pinned images.
	if !pinned {
		oldImages, err := mgr.ListImages(constants.ImageNamePrefix)
		if err != nil {
			return fmt.Errorf("listing images: %w", err)
		}
		for _, img := range oldImages {
			if img != targetImage && !isPinnedImage(img) {
				fmt.Fprintf(status, "Removing old image %s...\n", img)
				if rmErr := mgr.RemoveImage(img); rmErr != nil {
					return fmt.Errorf("removing old image %s: %w", img, rmErr)
				}
			}
		}
	}

	fmt.Fprintf(status, "Building image %s (this may take a moment)...\n", targetImage)
	dockerfile := generateDockerfile(ccboxVersion, claudeVersion)
	if err := mgr.BuildImage(targetImage, dockerfile, "."); err != nil {
		return fmt.Errorf("building image %s: %w", targetImage, err)
	}
	fmt.Fprintf(status, "Image %s ready.\n", targetImage)

	return nil
}

// pinnedTagPrefix is the tag prefix that distinguishes pinned images from auto-update images.
const pinnedTagPrefix = "pinned-"

// PinnedImageName returns the Docker image name for a pinned (--use) image.
// Format: ccbox-local:pinned-{ccboxVersion}-{claudeVersion}
func PinnedImageName(ccboxVersion, claudeVersion string) string {
	return fmt.Sprintf("%s:%s%s-%s", constants.ImageNamePrefix, pinnedTagPrefix, ccboxVersion, claudeVersion)
}

// isPinnedImage returns true if the image name has a pinned tag prefix.
func isPinnedImage(imageName string) bool {
	parts := strings.SplitN(imageName, ":", 2)
	if len(parts) != 2 {
		return false
	}
	return strings.HasPrefix(parts[1], pinnedTagPrefix)
}

// generateDockerfile returns the Dockerfile content for building a local ccbox image.
// The base image contains OS, system packages, and ccbox daemons.
// The local image layers Claude Code installation on top (Note I5).
func generateDockerfile(ccboxVersion, claudeVersion string) string {
	return fmt.Sprintf(`FROM %s:%s
LABEL ccbox.version="%s"
LABEL ccbox.claude-version="%s"
USER root
RUN curl -fsSL https://claude.ai/install.sh | bash -s -- %s
RUN cp /root/.local/bin/claude /home/claude/.local/bin/claude \
    && chown claude:claude /home/claude/.local/bin/claude
`, constants.BaseImageRegistry, ccboxVersion, ccboxVersion, claudeVersion, claudeVersion)
}

// LocalImageName returns the full Docker image name for a local ccbox image.
// Format: ccbox-local:{ccboxVersion}-{claudeVersion}
func LocalImageName(ccboxVersion, claudeVersion string) string {
	return fmt.Sprintf("%s:%s-%s", constants.ImageNamePrefix, ccboxVersion, claudeVersion)
}

// ParseImageTag extracts the ccbox and claude versions from an image tag.
// The tag format is "{ccboxVersion}-{claudeVersion}" where ccboxVersion
// is the portion before the first hyphen and claudeVersion is the remainder.
func ParseImageTag(tag string) (ccboxVersion, claudeVersion string, err error) {
	idx := strings.Index(tag, "-")
	if idx < 0 {
		return "", "", fmt.Errorf("invalid image tag %q: missing hyphen separator", tag)
	}
	ccboxVersion = tag[:idx]
	claudeVersion = tag[idx+1:]
	if ccboxVersion == "" || claudeVersion == "" {
		return "", "", fmt.Errorf("invalid image tag %q: empty version component", tag)
	}
	return ccboxVersion, claudeVersion, nil
}

// CleanAllImages removes all ccbox-managed Docker images unconditionally.
func CleanAllImages(mgr ImageManager, w io.Writer) error {
	images, err := mgr.ListImages(constants.ImageNamePrefix)
	if err != nil {
		return fmt.Errorf("listing images: %w", err)
	}
	if len(images) == 0 {
		fmt.Fprintln(w, "Nothing to remove.")
		return nil
	}
	for _, img := range images {
		fmt.Fprintf(w, "Removing %s...\n", img)
		if rmErr := mgr.RemoveImage(img); rmErr != nil {
			return fmt.Errorf("removing image %s: %w", img, rmErr)
		}
	}
	fmt.Fprintf(w, "Removed %d image(s).\n", len(images))
	return nil
}

// CleanImages removes all ccbox-managed Docker images except the latest
// auto-update image (FR-038). The latest auto-update image is the last
// non-pinned image in the list returned by ListImages.
func CleanImages(mgr ImageManager, w io.Writer) error {
	images, err := mgr.ListImages(constants.ImageNamePrefix)
	if err != nil {
		return fmt.Errorf("listing images: %w", err)
	}

	// Find the latest auto-update image (last non-pinned in list).
	latestAutoUpdate := ""
	for _, img := range images {
		if !isPinnedImage(img) {
			latestAutoUpdate = img
		}
	}

	// Remove everything except the latest auto-update image.
	removed := 0
	for _, img := range images {
		if img == latestAutoUpdate {
			fmt.Fprintf(w, "Keeping %s (latest)\n", img)
			continue
		}
		fmt.Fprintf(w, "Removing %s...\n", img)
		if rmErr := mgr.RemoveImage(img); rmErr != nil {
			return fmt.Errorf("removing image %s: %w", img, rmErr)
		}
		removed++
	}

	if removed == 0 {
		fmt.Fprintln(w, "Nothing to remove.")
	} else {
		fmt.Fprintf(w, "Removed %d image(s).\n", removed)
	}

	return nil
}

// statusWriter wraps an io.Writer with a "ccbox: " prefix for status messages.
func statusWriter(w io.Writer) io.Writer {
	return &prefixWriter{w: w, prefix: "ccbox: "}
}

type prefixWriter struct {
	w      io.Writer
	prefix string
}

func (pw *prefixWriter) Write(p []byte) (int, error) {
	// Prepend prefix to the output.
	_, err := fmt.Fprintf(pw.w, "%s%s", pw.prefix, p)
	if err != nil {
		return 0, err
	}
	return len(p), nil
}

// ImageTagMismatch reports whether the given image tag does not match
// the expected ccbox and claude versions.
func ImageTagMismatch(tag, currentCCBox, currentClaude string) bool {
	ccbox, claude, err := ParseImageTag(tag)
	if err != nil {
		return true
	}
	return ccbox != currentCCBox || claude != currentClaude
}
