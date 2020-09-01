package types

import (
	"context"
	"fmt"

	"github.com/docker/docker/api/types"
)

// AuthConfigs map a server url to a Docker authorization config.
type AuthConfigs map[string]types.AuthConfig

// AuthLoader is function that must be implemented by every cloud-specific registry authorization provider.
type AuthLoader func(ctx context.Context, url string) (AuthConfigs, error)

type NonCloudURLError struct {
	url string
}

func (e *NonCloudURLError) Error() string {
	return fmt.Sprintf("non-cloud url: %q", e.url)
}

func IsNotCloudURL(err error) bool {
	_, ok := err.(*NonCloudURLError)
	return ok
}
