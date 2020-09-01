package cloud

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRetrieveRegistryAuthorization(t *testing.T) {
	out, err := RetrieveRegistryAuthorization(context.TODO(), "977170443939.dkr.ecr.us-west-2.amazonaws.com/forge/credential-test")
	require.NoError(t, err)

	bs, err := json.MarshalIndent(out, "", " ")
	require.NoError(t, err)

	fmt.Println(string(bs))
}
