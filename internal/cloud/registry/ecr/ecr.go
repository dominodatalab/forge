package ecr

import (
	"context"
	"regexp"
	"sync"

	"github.com/aws/aws-sdk-go-v2/aws/external"
	"github.com/aws/aws-sdk-go-v2/service/ecr"
	"github.com/aws/aws-sdk-go-v2/service/ecr/ecriface"
	"github.com/docker/docker/api/types"
	"github.com/pkg/errors"

	"github.com/dominodatalab/forge/internal/cloud/registry"
)

var (
	urlRegex = regexp.MustCompile(`^\d{12}\.dkr\.ecr\.[a-z0-9-]+\.amazonaws.com/.+$`)

	client   ecriface.ClientAPI
	initOnce sync.Once
)

// LoadAuths will read the local AWS config once and use it to request ECR authorization data.
func LoadAuths(ctx context.Context, url string) (*types.AuthConfig, error) {
	var iErr error
	initOnce.Do(func() {
		config, err := external.LoadDefaultAWSConfig()
		if err != nil {
			iErr = errors.Wrap(err, "cannot load aws config")
			return
		}
		client = ecr.New(config)
	})
	if iErr != nil {
		return nil, iErr
	}
	return authenticate(ctx, url)
}

func authenticate(ctx context.Context, url string) (*types.AuthConfig, error) {
	// TODO: extract AWS account ID from url and add to token input
	input := &ecr.GetAuthorizationTokenInput{}
	req := client.GetAuthorizationTokenRequest(input)

	resp, err := req.Send(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get ecr auth token")
	}
	if len(resp.AuthorizationData) != 1 {
		return nil, errors.Wrapf(err, "invalid ecr authorization data: %v", resp.AuthorizationData)
	}
	data := resp.AuthorizationData[0]

	return &types.AuthConfig{
		Auth: *data.AuthorizationToken,
	}, nil
}

func init() {
	registry.DefaultURLMux().RegisterLoader(urlRegex, LoadAuths)
}
