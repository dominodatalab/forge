package gcr

/*
	Important implementation information:

	https://cloud.google.com/artifact-registry/docs/docker/authentication
	https://github.com/google/go-containerregistry/blob/v0.1.2/pkg/v1/google/auth.go
*/

import (
	"context"
	"regexp"

	"github.com/docker/docker/api/types"
	//dockertypes "github.com/docker/docker/api/types"
	//"golang.org/x/oauth2/google"
)

var urlRegex = regexp.MustCompile(`^gcr\.io$`)

const scope = "https://www.googleapis.com/auth/cloud-platform"

func LoadAuths(ctx context.Context, url string) (*types.AuthConfig, error) {
	//ts, err := google.DefaultTokenSource(ctx, scope)
	//if err != nil {
	//	return nil, err
	//}
	//token, err := ts.Token()
	//if err != nil {
	//	panic(err)
	//}
	//if !token.Valid() {
	//	panic("invalid token")
	//}
	//
	//ac := dockertypes.AuthConfig{
	//	Username: "oauth2accesstoken",
	//	Password: token.AccessToken,
	//}

	return nil, nil
}

//func init() {
//	registry.DefaultURLMux().RegisterLoader(urlRegex, LoadAuths)
//}
