package message

import (
	"errors"
	"fmt"
)

type Options struct {
	Broker Broker

	AmqpURI   string
	AmqpQueue string
}

func ValidationOpts(opts *Options) error {
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
