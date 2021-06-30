package v1alpha1

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBasicAuthConfig_IsInline(t *testing.T) {
	tests := []struct {
		username string
		password string
		expected bool
	}{
		{"", "", false},
		{"uname", "", false},
		{"", "passwd", false},
		{"uname", "passwd", true},
	}
	for _, tc := range tests {
		cfg := BasicAuthConfig{Username: tc.username, Password: tc.password}
		assert.Equalf(t, tc.expected, cfg.IsInline(), "%+v should be %t", cfg, tc.expected)
	}
}

func TestBasicAuthConfig_IsSecret(t *testing.T) {
	tests := []struct {
		name      string
		namespace string
		expected  bool
	}{
		{"", "", false},
		{"secret", "", false},
		{"", "namespace", false},
		{"secret", "namespace", true},
	}
	for _, tc := range tests {
		cfg := BasicAuthConfig{SecretName: tc.name, SecretNamespace: tc.namespace}
		assert.Equalf(t, tc.expected, cfg.IsSecret(), "%+v should be %t", cfg, tc.expected)
	}
}

func TestBasicAuthConfig_Validate(t *testing.T) {
	tests := []struct {
		cfg BasicAuthConfig
		err error
	}{
		{
			BasicAuthConfig{},
			nil,
		},
		{
			BasicAuthConfig{Username: "name", Password: "pass"},
			nil,
		},
		{
			BasicAuthConfig{SecretName: "secret", SecretNamespace: "namespace"},
			nil,
		},
		{
			BasicAuthConfig{Username: "name"},
			errors.New("inline basic auth requires both username and password"),
		},
		{
			BasicAuthConfig{Password: "pass"},
			errors.New("inline basic auth requires both username and password"),
		},
		{
			BasicAuthConfig{SecretName: "secret"},
			errors.New("secret basic auth requires both secret name and namespace"),
		},
		{
			BasicAuthConfig{SecretNamespace: "namespace"},
			errors.New("secret basic auth requires both secret name and namespace")},
		{
			BasicAuthConfig{SecretNamespace: "namespace"},
			errors.New("secret basic auth requires both secret name and namespace"),
		},
		{
			BasicAuthConfig{Username: "name", Password: "pass", SecretName: "secret", SecretNamespace: "namespace"},
			errors.New("basic auth cannot be both inline and secret-based"),
		},
		{
			BasicAuthConfig{Username: "name", SecretName: "secret", SecretNamespace: "namespace"},
			errors.New("inline basic auth requires both username and password"),
		},
		{
			BasicAuthConfig{Password: "pass", SecretName: "secret", SecretNamespace: "namespace"},
			errors.New("inline basic auth requires both username and password"),
		},
		{
			BasicAuthConfig{Username: "name", Password: "pass", SecretName: "secret"},
			errors.New("secret basic auth requires both secret name and namespace"),
		},
		{
			BasicAuthConfig{Username: "name", Password: "pass", SecretNamespace: "namespace"},
			errors.New("secret basic auth requires both secret name and namespace"),
		},
	}
	for _, tc := range tests {
		assert.Equal(t, tc.err, tc.cfg.Validate())
	}
}
