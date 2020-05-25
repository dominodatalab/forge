package message

import (
	"errors"
	"fmt"
)

// Options defines the configuration for supported brokers.
type Options struct {
	Broker Broker

	AmqpURI   string
	AmqpQueue string
}

// ValidateOpts enforces broker-specific configuration requirements.
func ValidateOpts(opts *Options) error {
	switch opts.Broker {
	case AmqpBroker:
		if opts.AmqpURI == "" || opts.AmqpQueue == "" {
			return errors.New("amqp broker requires a uri and queue name")
		}
	default:
		return fmt.Errorf("broker %q is invalid (supported brokers: %v)", opts.Broker, SupportedBrokers)
	}

	return nil
}
