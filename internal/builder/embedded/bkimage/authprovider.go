package bkimage

import (
	"context"
	"fmt"

	"github.com/moby/buildkit/session"
	"github.com/moby/buildkit/session/auth"
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
		return nil, fmt.Errorf("credential fetch failed for %q: %w", req.Host, err)
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
