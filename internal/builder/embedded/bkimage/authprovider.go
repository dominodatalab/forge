package bkimage

import (
	"context"
	"crypto/ed25519"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"

	authutil "github.com/containerd/containerd/remotes/docker/auth"
	remoteserrors "github.com/containerd/containerd/remotes/errors"
	"github.com/moby/buildkit/session"
	"github.com/moby/buildkit/session/auth"
	"github.com/pkg/errors"
	"golang.org/x/crypto/nacl/sign"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// copied from https://github.com/moby/buildkit/blob/4e69662758446c7dc0e6de2bc1f7973d03bacbed/session/auth/authprovider/authprovider.go

type tokenSeeds struct {
	mu sync.Mutex
	m  map[string]seed
}

type seed struct {
	Seed []byte
}
type dynamicAuthProvider struct {
	credFn CredentialsFn
	seeds  *tokenSeeds
}

func NewDynamicAuthProvider(fn CredentialsFn) session.Attachable {
	return &dynamicAuthProvider{
		credFn: fn,
		seeds:  &tokenSeeds{},
	}
}

func (ap *dynamicAuthProvider) Register(server *grpc.Server) {
	auth.RegisterAuthServer(server, ap)
}

func (ap *dynamicAuthProvider) Credentials(ctx context.Context, req *auth.CredentialsRequest) (*auth.CredentialsResponse, error) {
	return ap.credentials(req.Host)
}

func (ap *dynamicAuthProvider) credentials(host string) (*auth.CredentialsResponse, error) {
	if host == "registry-1.docker.io" {
		host = "https://index.docker.io/v1/"
	}

	username, password, err := ap.credFn(host)
	if err != nil {
		return nil, errors.Wrapf(err, "credential fetch failed for %q", host)
	}

	return &auth.CredentialsResponse{
		Username: username,
		Secret:   password,
	}, nil
}

func (ap *dynamicAuthProvider) FetchToken(ctx context.Context, req *auth.FetchTokenRequest) (*auth.FetchTokenResponse, error) {
	creds, err := ap.credentials(req.Host)
	if err != nil {
		return nil, err
	}

	to := authutil.TokenOptions{
		Realm:    req.Realm,
		Service:  req.Service,
		Scopes:   req.Scopes,
		Username: creds.Username,
		Secret:   creds.Secret,
	}

	if creds.Secret != "" {
		defer func() {
			err = errors.Wrap(err, "failed to fetch oauth token")
		}()
		// try GET first because Docker Hub does not support POST
		// switch once support has landed
		resp, err := authutil.FetchToken(ctx, http.DefaultClient, nil, to)
		if err != nil {
			var errStatus remoteserrors.ErrUnexpectedStatus
			if errors.As(err, &errStatus) {
				// retry with POST request
				// As of September 2017, GCR is known to return 404.
				// As of February 2018, JFrog Artifactory is known to return 401.
				if (errStatus.StatusCode == 405 && to.Username != "") || errStatus.StatusCode == 404 || errStatus.StatusCode == 401 {
					resp, err := authutil.FetchTokenWithOAuth(ctx, http.DefaultClient, nil, "buildkit-client", to)
					if err != nil {
						return nil, err
					}

					return toTokenResponse(resp.AccessToken, resp.IssuedAt, resp.ExpiresIn), nil
				}
			}
			return nil, err
		}
		return toTokenResponse(resp.Token, resp.IssuedAt, resp.ExpiresIn), nil
	}
	// do request anonymously
	resp, err := authutil.FetchToken(ctx, http.DefaultClient, nil, to)
	if err != nil {
		return nil, errors.Wrap(err, "failed to fetch anonymous token")
	}
	return toTokenResponse(resp.Token, resp.IssuedAt, resp.ExpiresIn), nil
}

func (ap *dynamicAuthProvider) GetTokenAuthority(ctx context.Context, req *auth.GetTokenAuthorityRequest) (*auth.GetTokenAuthorityResponse, error) {
	key, err := ap.getAuthorityKey(req.Host, req.Salt)
	if err != nil {
		return nil, err
	}

	return &auth.GetTokenAuthorityResponse{PublicKey: key[32:]}, nil
}

func (ap *dynamicAuthProvider) VerifyTokenAuthority(ctx context.Context, req *auth.VerifyTokenAuthorityRequest) (*auth.VerifyTokenAuthorityResponse, error) {
	key, err := ap.getAuthorityKey(req.Host, req.Salt)
	if err != nil {
		return nil, err
	}

	priv := new([64]byte)
	copy((*priv)[:], key)

	return &auth.VerifyTokenAuthorityResponse{Signed: sign.Sign(nil, req.Payload, priv)}, nil
}

func (ap *dynamicAuthProvider) getAuthorityKey(host string, salt []byte) (ed25519.PrivateKey, error) {
	if v, err := strconv.ParseBool(os.Getenv("BUILDKIT_NO_CLIENT_TOKEN")); err == nil && v {
		return nil, status.Errorf(codes.Unavailable, "client side tokens disabled")
	}

	creds, err := ap.credentials(host)
	if err != nil {
		return nil, err
	}
	seed, err := ap.seeds.getSeed(host)
	if err != nil {
		return nil, err
	}

	mac := hmac.New(sha256.New, salt)
	if creds.Secret != "" {
		mac.Write(seed)
	}

	sum := mac.Sum(nil)

	return ed25519.NewKeyFromSeed(sum[:ed25519.SeedSize]), nil
}

func (ts *tokenSeeds) getSeed(host string) ([]byte, error) {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	if ts.m == nil {
		ts.m = map[string]seed{}
	}

	v, ok := ts.m[host]
	if !ok {
		b := make([]byte, 16)
		if _, err := rand.Read(b); err != nil {
			return nil, err
		}
		v = seed{Seed: b}
	}

	ts.m[host] = v

	return v.Seed, nil
}

func toTokenResponse(token string, issuedAt time.Time, expires int) *auth.FetchTokenResponse {
	resp := &auth.FetchTokenResponse{
		Token:     token,
		ExpiresIn: int64(expires),
	}
	if !issuedAt.IsZero() {
		resp.IssuedAt = issuedAt.Unix()
	}
	return resp
}
