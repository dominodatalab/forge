package message

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidationOpts(t *testing.T) {
	for _, name := range []string{"", "kafka", "AMQP"} {
		t.Run("invalid_broker_"+name, func(t *testing.T) {
			err := ValidateOpts(&Options{Broker: Broker(name)})

			require.Error(t, err)
			assert.Contains(t, err.Error(), fmt.Sprintf("broker %q is invalid", name))
		})
	}

	tests := []struct {
		name  string
		uri   string
		queue string
		valid bool
	}{
		{"missing_all", "", "", false},
		{"missing_uri", "", "queue-name", false},
		{"missing_queue", "amqp://user:pass@host", "", false},
		{"valid_opts", "amqp://user:pass@host", "queue-name", true},
	}
	for _, tc := range tests {
		t.Run("amqp_"+tc.name, func(t *testing.T) {
			opts := &Options{
				Broker:    AmqpBroker,
				AmqpURI:   tc.uri,
				AmqpQueue: tc.queue,
			}
			err := ValidateOpts(opts)

			if tc.valid {
				assert.NoError(t, err)
				return
			}
			require.Error(t, err)
			assert.Equal(t, "amqp broker requires a uri and queue name", err.Error())
		})
	}
}
