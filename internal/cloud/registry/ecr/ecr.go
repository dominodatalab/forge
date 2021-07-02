package ecr

import (
	"context"
	"encoding/base64"
	"fmt"
	"regexp"
	"strings"
	"sync"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ecr"
	"github.com/docker/docker/api/types"
	"github.com/pkg/errors"

	"github.com/dominodatalab/forge/internal/cloud/registry"
)

type ecrClient interface {
	GetAuthorizationToken(ctx context.Context, params *ecr.GetAuthorizationTokenInput, optFns ...func(*ecr.Options)) (*ecr.GetAuthorizationTokenOutput, error)
}

var (
	urlRegex = regexp.MustCompile(`^(?P<aws_account_id>[a-zA-Z0-9][a-zA-Z0-9-_]*)\.dkr\.ecr(-fips)?\.([a-zA-Z0-9][a-zA-Z0-9-_]*)\.amazonaws\.com(\.cn)?`)

	client   ecrClient
	initOnce sync.Once
)

// LoadAuths will read the local AWS config once and use it to request ECR authorization data.
func LoadAuths(ctx context.Context, url string) (*types.AuthConfig, error) {
	var iErr error
	initOnce.Do(func() {
		config, err := config.LoadDefaultConfig(ctx)
		if err != nil {
			iErr = errors.Wrap(err, "cannot load aws config")
			return
		}
		client = ecr.NewFromConfig(config)
	})
	if iErr != nil {
		return nil, iErr
	}
	return authenticate(ctx, url)
}

func authenticate(ctx context.Context, url string) (*types.AuthConfig, error) {
	match := urlRegex.FindStringSubmatch(url)
	if match == nil {
		return nil, fmt.Errorf("invalid ecr url: %q should match %v", url, urlRegex)
	}
	input := &ecr.GetAuthorizationTokenInput{}

	resp, err := client.GetAuthorizationToken(ctx, input)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get ecr auth token")
	}
	if len(resp.AuthorizationData) != 1 {
		return nil, errors.Wrapf(err, "expected a single ecr authorization token: %v", resp.AuthorizationData)
	}
	authToken := aws.ToString(resp.AuthorizationData[0].AuthorizationToken)

	username, password, err := decodeAuth(authToken)
	if err != nil {
		return nil, errors.Wrap(err, "invalid ecr authorization token")
	}

	return &types.AuthConfig{
		Username: username,
		Password: password,
	}, nil
}

func decodeAuth(auth string) (string, string, error) {
	if auth == "" {
		return "", "", errors.New("docker auth token cannot be blank")
	}

	decoded, err := base64.StdEncoding.DecodeString(auth)
	if err != nil {
		return "", "", errors.Wrap(err, "failed to decode docker auth token")
	}

	creds := strings.SplitN(string(decoded), ":", 2)
	if len(creds) != 2 {
		return "", "", fmt.Errorf("invalid docker auth token: %q", creds)
	}
	return creds[0], creds[1], nil
}

func init() {
	registry.DefaultURLMux().RegisterLoader(urlRegex, LoadAuths)
}
