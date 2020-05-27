package bkimage

import (
	"fmt"

	"github.com/docker/distribution/reference"
)

func parseImageName(image string) (string, error) {
	// parse the image name and tag
	named, err := reference.ParseNormalizedNamed(image)
	if err != nil {
		return "", fmt.Errorf("parsing image name %q failed: %w", image, err)
	}

	// Add "latest" tag if tag is missing.
	named = reference.TagNameOnly(named)
	return named.String(), nil
}
