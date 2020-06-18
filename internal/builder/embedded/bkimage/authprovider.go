package bkimage

import (
	"context"

	"github.com/moby/buildkit/session"
	"github.com/moby/buildkit/session/auth"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
)

type dynamicAuthProvider struct {
	credFn CredentialsFn
}

func (ap *dynamicAuthProvider) Register(server *grpc.Server) {
	auth.RegisterAuthServer(server, ap)
}

func (ap *dynamicAuthProvider) Credentials(ctx context.Context, req *auth.CredentialsRequest) (*auth.CredentialsResponse, error) {
	if req.Host == "registry-1.docker.io" {
		req.Host = "https://index.docker.io/v1/"
	}

	username, password, err := ap.credFn(req.Host)
	if err != nil {
		return nil, errors.Wrapf(err, "credential fetch failed for %q", req.Host)
	}

	return &auth.CredentialsResponse{
		Username: username,
		Secret:   password,
	}, nil
}

func NewDynamicAuthProvider(fn CredentialsFn) session.Attachable {
	return &dynamicAuthProvider{
		credFn: fn,
	}
}
