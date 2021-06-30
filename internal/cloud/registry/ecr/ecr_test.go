package ecr

import (
	"context"
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ecr"
	"github.com/aws/aws-sdk-go-v2/service/ecr/types"
	dockertypes "github.com/docker/docker/api/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dominodatalab/forge/internal/cloud/registry"
)

func TestPatternMatching(t *testing.T) {
	testcases := []struct {
		name      string
		url       string
		expectErr bool
	}{
		{
			name: "america",
			url:  "0123456789012.dkr.ecr.us-west-2.amazonaws.com",
		},
		{
			name: "fips",
			url:  "0123456789012.dkr.ecr-fips.us-gov-east-1.amazonaws.com",
		},
		{
			name: "china",
			url:  "0123456789012.dkr.ecr.cn-north-1.amazonaws.com.cn",
		},
		{
			name:      "no_region",
			url:       "0123456789012.dkr.ecr.amazonaws.com",
			expectErr: true,
		},
		{
			name:      "no_account_id",
			url:       "dkr.ecr.us-east-1.amazonaws.com",
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
	url := "0123456789012.dkr.ecr.af-south-1.amazonaws.com"

	// prevent loading the AWS config
	initOnce.Do(func() {})

	t.Run("success", func(t *testing.T) {
		client = &mockECRClient{
			out: &ecr.GetAuthorizationTokenOutput{
				AuthorizationData: []types.AuthorizationData{
					{
						AuthorizationToken: aws.String("c3RldmUtbzphd2Vzb21l"), // base64 -> "steve-o:awesome"
					},
				},
			},
		}

		actual, err := LoadAuths(ctx, url)
		expected := &dockertypes.AuthConfig{
			Username: "steve-o",
			Password: "awesome",
		}

		require.NoError(t, err)
		assert.Equal(t, expected, actual)
	})

	t.Run("request_error", func(t *testing.T) {
		client = &mockECRClient{
			err: errors.New("api ka-boom"),
		}

		_, err := LoadAuths(ctx, url)
		assert.EqualError(t, err, "failed to get ecr auth token: api ka-boom")
	})

	t.Run("bad_url", func(t *testing.T) {
		client = &mockECRClient{}

		_, err := LoadAuths(ctx, "garbage.url")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid ecr url")
	})
}

type mockECRClient struct {
	out *ecr.GetAuthorizationTokenOutput // mock output
	err error                            // mock error
}

func (m *mockECRClient) GetAuthorizationToken(ctx context.Context, params *ecr.GetAuthorizationTokenInput, optFns ...func(*ecr.Options)) (*ecr.GetAuthorizationTokenOutput, error) {
	return m.out, m.err
}
