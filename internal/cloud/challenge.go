package cloud

import (
	"context"
	"fmt"
	"strings"

	"github.com/Azure/azure-sdk-for-go/services/preview/containerregistry/runtime/2019-08-15-preview/containerregistry"
)

type AuthDirective struct {
	Service string
	Realm   string
}

func ChallengeLoginServer(ctx context.Context, loginServerURL string) (*AuthDirective, error) {
	v2Support := containerregistry.NewV2SupportClient(loginServerURL)
	challenge, err := v2Support.Check(ctx)
	// A 401 will also return an error so just check first
	if !challenge.IsHTTPStatus(401) {
		if err != nil {
			return nil, err
		}

		defer challenge.Body.Close()
		return nil, fmt.Errorf("registry did not issue a valid AAD challenge, status: %d", challenge.StatusCode)
	}

	//Www-Authenticate: Bearer realm="https://xxx.azurecr.io/oauth2/token",service="xxx.azurecr.io"
	authHeader, ok := challenge.Header["Www-Authenticate"]
	if !ok {
		return nil, fmt.Errorf("challenge response does not contain header 'Www-Authenticate'")
	}

	if len(authHeader) != 1 {
		return nil, fmt.Errorf("registry did not issue a valid challenge, authenticate header [%s]",
			strings.Join(authHeader, ", "))
	}

	authSections := strings.SplitN(authHeader[0], " ", 2)
	if !strings.EqualFold("Bearer", authSections[0]) {
		return nil, fmt.Errorf("Www-Authenticate: expected realm: Bearer, actual: %s", authSections[0])
	}

	authParams := map[string]string{}
	params := strings.Split(authSections[1], ",")
	for _, p := range params {
		parts := strings.SplitN(strings.TrimSpace(p), "=", 2)
		authParams[parts[0]] = strings.Trim(parts[1], `"`)
	}

	// verify headers
	if authParams["realm"] == "" {
		return nil, fmt.Errorf("Www-Authenticate: missing header \"realm\"")
	}

	return &AuthDirective{
		Service: authParams["service"], // optional
		Realm:   authParams["realm"],
	}, nil
}
