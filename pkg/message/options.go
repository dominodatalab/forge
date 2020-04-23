package message

import "fmt"

type Options struct {
	Broker Broker

	AmqpURI   string
	AmqpQueue string
}

func ValidationOpts(opts *Options) error {
	name := opts.Broker

	ok := false
	for _, broker := range SupportedBrokers {
		if name == broker {
			ok = true
		}
	}
	if !ok {
		return fmt.Errorf("broker %q is invalid (supported brokers: %v)", name, SupportedBrokers)
	}

	switch opts.Broker {
	case "amqp":
		if opts.AmqpURI == "" || opts.AmqpQueue == "" {
			return fmt.Errorf("broker %q requires a uri and queue name", "amqp")
		}
	}

	return nil
}
