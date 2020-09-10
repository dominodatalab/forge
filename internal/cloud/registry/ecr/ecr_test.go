package ecr

import (
	"context"
	"errors"
	"net/http"
	"sync"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/external"
	"github.com/aws/aws-sdk-go-v2/service/ecr"
	"github.com/aws/aws-sdk-go-v2/service/ecr/ecriface"
	dockertypes "github.com/docker/docker/api/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/utils/pointer"

	"github.com/dominodatalab/forge/internal/cloud/registry"
)

func TestPatternMatching(t *testing.T) {
	testcases := []struct{
		name string
		url string
		expectErr bool
	}{
		{
			name: "america",
			url: "0123456789012.dkr.ecr.us-west-2.amazonaws.com",
		},
		{
			name: "fips",
			url: "0123456789012.dkr.ecr-fips.us-gov-east-1.amazonaws.com",
		},
		{
			name: "china",
			url: "0123456789012.dkr.ecr.cn-north-1.amazonaws.com.cn",
		},
		{
			name: "no_region",
			url: "0123456789012.dkr.ecr.amazonaws.com",
			expectErr: true,
		},
		{
			name: "no_account_id",
			url: "dkr.ecr.us-east-1.amazonaws.com",
			expectErr: true,
		},
	}
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			loader, err := registry.DefaultURLMux().FromString(tc.url)

			if tc.expectErr {
				assert.Error(t, err, "expected %q to return err", tc.url)
				return
			}

			assert.NotNil(t, loader, "expected %q to return ECR loader", tc.url)
		})
	}
}

func TestLoadAuths(t *testing.T) {
	ctx := context.Background()
	url := "ignored"

	t.Run("success", func(t *testing.T) {
		client = &mockECRClient{
			out: &ecr.GetAuthorizationTokenOutput{
				AuthorizationData: []ecr.AuthorizationData{
					{
						ProxyEndpoint:      pointer.StringPtr("https://123456789012.dkr.ecr.us-west-2.amazonaws.com"),
						AuthorizationToken: pointer.StringPtr("c3RldmUtbwo="), // base64 -> "steve-o"
					},
				},
			},
		}
		initOnce.Do(func() {})

		actual, err := LoadAuths(ctx, url)
		expected := &dockertypes.AuthConfig{
			Auth: "c3RldmUtbwo=",
		}

		require.NoError(t, err)
		assert.Equal(t, expected, actual)
	})

	t.Run("request_error", func(t *testing.T) {
		client = &mockECRClient{
			err: errors.New("api ka-boom"),
		}
		initOnce.Do(func() {})

		_, err := LoadAuths(ctx, url)
		assert.EqualError(t, err, "failed to get ecr auth token: api ka-boom")
	})

	t.Run("resolve_failure", func(t *testing.T) {
		initOnce = sync.Once{}

		badResolver := func(cfg *aws.Config, configs external.Configs) error {
			return errors.New("resolve error")
		}
		external.DefaultAWSConfigResolvers = append(
			[]external.AWSConfigResolver{badResolver},
			external.DefaultAWSConfigResolvers...,
		)
		defer func() {
			external.DefaultAWSConfigResolvers = external.DefaultAWSConfigResolvers[1:]
		}()

		out, err := LoadAuths(ctx, url)

		require.Nil(t, out)
		assert.EqualError(t, err, "cannot load aws config: resolve error")
	})
}

type mockECRClient struct {
	ecriface.ClientAPI
	out *ecr.GetAuthorizationTokenOutput
	err error
}

func (m *mockECRClient) GetAuthorizationTokenRequest(input *ecr.GetAuthorizationTokenInput) ecr.GetAuthorizationTokenRequest {
	mockReq := &aws.Request{
		HTTPRequest:  &http.Request{},
		HTTPResponse: &http.Response{},
		Retryer:      aws.NoOpRetryer{},
		Data:         m.out,
		Error:        m.err,
	}
	return ecr.GetAuthorizationTokenRequest{
		Request: mockReq,
	}
}
