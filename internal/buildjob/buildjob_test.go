package buildjob

import (
	"context"
	"fmt"
	"testing"

	"github.com/dominodatalab/forge/api/forge/v1alpha1"
	"github.com/dominodatalab/forge/internal/config"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	testK8sClient "k8s.io/client-go/kubernetes/fake"
)

func TestBuildRegistryConfigs(t *testing.T) {
	noAuthHost := "noauth-test.com"

	noAuthRegistryConfig := v1alpha1.Registry{
		Server: noAuthHost,
		BasicAuth: v1alpha1.BasicAuthConfig{
			Username: "",
			Password: "",
		},
	}

	noAuthRegistry := config.Registry{
		Host:   noAuthHost,
		NonSSL: false,
	}

	inlineUsername := "marge"
	inlinePassword := "simpson"
	inlineHost := "inline-test.com"

	inlineRegistryConfig := v1alpha1.Registry{
		Server: inlineHost,
		BasicAuth: v1alpha1.BasicAuthConfig{
			Username: inlineUsername,
			Password: inlinePassword,
		},
	}

	inlineRegistry := config.Registry{
		Host:     inlineHost,
		Username: inlineUsername,
		Password: inlinePassword,
		NonSSL:   false,
	}

	secretUsername1 := "johndoe"
	secretPassword1 := "anothersecurepassword"
	secretHost1 := "secret-test.com"

	secretRegistry1 := config.Registry{
		Host:     secretHost1,
		Username: secretUsername1,
		Password: secretPassword1,
		NonSSL:   false,
	}

	secretUsername2 := "janedoe"
	secretPassword2 := "yetanothersecurepassword"
	secretHost2 := inlineHost // intentional conflict with inline auth host

	secretRegistry2 := config.Registry{
		Host:     secretHost2,
		Username: secretUsername2,
		Password: secretPassword2,
		NonSSL:   false,
	}

	singleRegistrySecretName := "SecretWithSingleRegistry"
	singleRegistrySecretNamespace := "SomeNamespace"
	singleRegistrySecretContent := []byte(
		fmt.Sprintf(`{"auths":{"%v":{"username":"%v","password":"%v"}}}`, secretHost1, secretUsername1, secretPassword1))

	singleRegistrySecretRegistryConfig := v1alpha1.Registry{
		Server: secretHost1,
		BasicAuth: v1alpha1.BasicAuthConfig{
			SecretName:      singleRegistrySecretName,
			SecretNamespace: singleRegistrySecretNamespace,
		},
	}

	twoRegistriesSecretName := "SecretWithTwoRegistries"
	twoRegistriesSecretNamespace := "SomeNamespace"
	twoRegistriesSecretContent := []byte(
		fmt.Sprintf(`{"auths":{"%v":{"username":"%v","password":"%v"}, "%v":{"username":"%v","password":"%v"}}}`,
			secretHost1, secretUsername1, secretPassword1,
			secretHost2, secretUsername2, secretPassword2,
		))

	twoRegistriesSecretRegistryConfig := v1alpha1.Registry{
		Server: secretHost1,
		BasicAuth: v1alpha1.BasicAuthConfig{
			SecretName:      twoRegistriesSecretName,
			SecretNamespace: twoRegistriesSecretNamespace,
		},
	}

	clientWithSingleRegistrySecret := testK8sClient.NewSimpleClientset(
		&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      singleRegistrySecretName,
				Namespace: singleRegistrySecretNamespace,
			},
			Type: corev1.SecretTypeDockerConfigJson,
			Data: map[string][]byte{
				corev1.DockerConfigJsonKey: singleRegistrySecretContent,
			},
		},
	)

	clientWithTwoRegistriesSecret := testK8sClient.NewSimpleClientset(
		&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      twoRegistriesSecretName,
				Namespace: twoRegistriesSecretNamespace,
			},
			Type: corev1.SecretTypeDockerConfigJson,
			Data: map[string][]byte{
				corev1.DockerConfigJsonKey: twoRegistriesSecretContent,
			},
		},
	)

	ctx := context.Background()
	log := NewLogger()
	job := Job{
		log: log,
	}

	t.Run("invalid_basic_auth_both_inline_and_secret", func(t *testing.T) {
		apiRegs := []v1alpha1.Registry{
			{
				Server: "test.com",
				BasicAuth: v1alpha1.BasicAuthConfig{
					Username:        "this",
					Password:        "should",
					SecretName:      "never",
					SecretNamespace: "happen",
				},
			},
		}
		_, err := job.buildRegistryConfigs(ctx, apiRegs)
		assert.Error(t, err)
	})

	t.Run("no_auth_basic_auth", func(t *testing.T) {
		apiRegs := []v1alpha1.Registry{noAuthRegistryConfig}
		registries, err := job.buildRegistryConfigs(ctx, apiRegs)

		assert.NoError(t, err)
		assert.ElementsMatch(t, registries, []config.Registry{noAuthRegistry})
	})

	t.Run("single_basic_auth_from_inline", func(t *testing.T) {
		apiRegs := []v1alpha1.Registry{inlineRegistryConfig}
		registries, err := job.buildRegistryConfigs(ctx, apiRegs)

		assert.NoError(t, err)
		assert.ElementsMatch(t, registries, []config.Registry{inlineRegistry})
	})

	t.Run("single_basic_auth_from_inline_with_nonssl", func(t *testing.T) {
		inlineRegistryConfigWithNonSSl := inlineRegistryConfig
		inlineRegistryConfigWithNonSSl.NonSSL = true

		inlineRegistryWithNonSSl := inlineRegistry
		inlineRegistryWithNonSSl.NonSSL = true

		apiRegs := []v1alpha1.Registry{inlineRegistryConfigWithNonSSl}
		registries, err := job.buildRegistryConfigs(ctx, apiRegs)

		assert.NoError(t, err)
		assert.ElementsMatch(t, registries, []config.Registry{inlineRegistryWithNonSSl})
	})

	t.Run("single_basic_auth_from_secret", func(t *testing.T) {
		job.clientk8s = clientWithSingleRegistrySecret

		apiRegs := []v1alpha1.Registry{singleRegistrySecretRegistryConfig}
		registries, err := job.buildRegistryConfigs(ctx, apiRegs)

		assert.NoError(t, err)
		assert.ElementsMatch(t, registries, []config.Registry{secretRegistry1})
	})

	t.Run("load_implicit_basic_auth_from_secret", func(t *testing.T) {
		job.clientk8s = clientWithTwoRegistriesSecret

		apiRegs := []v1alpha1.Registry{twoRegistriesSecretRegistryConfig}
		registries, err := job.buildRegistryConfigs(ctx, apiRegs)

		assert.NoError(t, err)
		assert.ElementsMatch(t, registries, []config.Registry{secretRegistry1, secretRegistry2})
	})

	t.Run("override_implicit_basic_auth_from_secret", func(t *testing.T) {
		job.clientk8s = clientWithTwoRegistriesSecret

		apiRegs := []v1alpha1.Registry{twoRegistriesSecretRegistryConfig, inlineRegistryConfig}
		registries, err := job.buildRegistryConfigs(ctx, apiRegs)

		assert.NoError(t, err)
		assert.ElementsMatch(t, registries, []config.Registry{secretRegistry1, inlineRegistry})

		// Test order irrelevant to overriding
		apiRegsReverseOrder := []v1alpha1.Registry{inlineRegistryConfig, twoRegistriesSecretRegistryConfig}
		registriesReverseOrder, err := job.buildRegistryConfigs(ctx, apiRegsReverseOrder)

		assert.NoError(t, err)
		assert.ElementsMatch(t, registriesReverseOrder, []config.Registry{secretRegistry1, inlineRegistry})
	})

	t.Run("duplicate_explicit_basic_auth_last_wins", func(t *testing.T) {
		job.clientk8s = clientWithTwoRegistriesSecret

		conflictingHostsRegistryConfig := twoRegistriesSecretRegistryConfig
		conflictingHostsRegistryConfig.Server = secretHost2 // conflicts with inline host

		// Secret first, later explicit load from inline overrides
		apiRegs := []v1alpha1.Registry{conflictingHostsRegistryConfig, inlineRegistryConfig}
		registries, err := job.buildRegistryConfigs(ctx, apiRegs)
		assert.NoError(t, err)
		assert.ElementsMatch(t, registries, []config.Registry{secretRegistry1, inlineRegistry})

		// Inline first, later explicit load from secret overrides
		apiRegsReverseOrder := []v1alpha1.Registry{inlineRegistryConfig, conflictingHostsRegistryConfig}
		registriesReverseOrder, err := job.buildRegistryConfigs(ctx, apiRegsReverseOrder)
		assert.NoError(t, err)
		assert.ElementsMatch(t, registriesReverseOrder, []config.Registry{secretRegistry1, secretRegistry2})
	})
}
