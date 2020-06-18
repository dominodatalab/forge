package bkimage

import (
	"github.com/docker/distribution/reference"
	"github.com/pkg/errors"
)

func parseImageName(image string) (string, error) {
	// parse the image name and tag
	named, err := reference.ParseNormalizedNamed(image)
	if err != nil {
		return "", errors.Wrapf(err, "parsing image name %q failed", image)
	}

	// Add "latest" tag if tag is missing.
	named = reference.TagNameOnly(named)
	return named.String(), nil
}
