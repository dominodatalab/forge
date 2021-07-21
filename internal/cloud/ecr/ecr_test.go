package ecr

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ecr"
	"github.com/aws/aws-sdk-go-v2/service/ecr/types"
	dockertypes "github.com/docker/docker/api/types"
	"github.com/dominodatalab/forge/internal/cloud"
	"github.com/go-logr/logr"
	"github.com/go-logr/zapr"
	"go.uber.org/zap"
)

func TestRegister(t *testing.T) {
	registry := &cloud.Registry{}

	err := Register(newLogger(t), registry)
	if err != nil {
		t.Error(err)
	}
}

func TestPatternMatching(t *testing.T) {
	testcases := []struct {
		name  string
		url   string
		match bool
	}{
		{
			name:  "america",
			url:   "0123456789012.dkr.ecr.us-west-2.amazonaws.com",
			match: true,
		},
		{
			name:  "fips",
			url:   "0123456789012.dkr.ecr-fips.us-gov-east-1.amazonaws.com",
			match: true,
		},
		{
			name:  "china",
			url:   "0123456789012.dkr.ecr.cn-north-1.amazonaws.com.cn",
			match: true,
		},
		{
			name: "no_region",
			url:  "0123456789012.dkr.ecr.amazonaws.com",
		},
		{
			name: "no_account_id",
			url:  "dkr.ecr.us-east-1.amazonaws.com",
		},
	}
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			actual := urlRegex.MatchString(tc.url)

			if actual != tc.match {
				t.Errorf("wrong match: got %v", actual)
			}
		})
	}
}

func TestAuthenticate(t *testing.T) {
	ctx := context.Background()
	url := "0123456789012.dkr.ecr.af-south-1.amazonaws.com"

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

		actual, err := authenticate(ctx, url)
		expected := &dockertypes.AuthConfig{
			Username: "steve-o",
			Password: "awesome",
		}

		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(expected, actual) {
			t.Fatalf("wrong auth: got %v, want %v", actual, expected)
		}
	})

	t.Run("request_error", func(t *testing.T) {
		expected := errors.New("api ka-boom")
		client = &mockECRClient{
			err: expected,
		}

		_, err := authenticate(ctx, url)
		if !errors.Is(err, expected) {
			t.Fatalf("wrong err: %v", err)
		}
	})

	t.Run("bad_url", func(t *testing.T) {
		client = &mockECRClient{}

		_, err := authenticate(ctx, "garbage.url")
		if err == nil {
			t.Fatalf("expected err")
		}
		if !strings.Contains(err.Error(), "invalid ecr url") {
			t.Fatalf("wrong err: %v", err)
		}
	})
}

type mockECRClient struct {
	out *ecr.GetAuthorizationTokenOutput // mock output
	err error                            // mock error
}

func (m *mockECRClient) GetAuthorizationToken(ctx context.Context, params *ecr.GetAuthorizationTokenInput, optFns ...func(*ecr.Options)) (*ecr.GetAuthorizationTokenOutput, error) {
	return m.out, m.err
}

func newLogger(t *testing.T) logr.Logger {
	zlog, err := zap.NewDevelopment()
	if err != nil {
		t.Fatal(err)
	}
	return zapr.NewLogger(zlog)
}
