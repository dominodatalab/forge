package message

import (
	"fmt"

	"github.com/dominodatalab/forge/internal/message/amqp"
)

// Broker represents a messaging protocol implementation.
type Broker string

// AmqpBroker publishes messages using the AMQP protocol.
const AmqpBroker Broker = "amqp"

// SupportedBrokers defines the list of implemented message publishers.
var SupportedBrokers = []Broker{AmqpBroker}

// Producer defines the operations required by all message producers.
type Publisher interface {
	UnsafePush(message interface{}) error
	Close() error
}

// NewProducer configures a new message producer using the provided options.
func NewPublisher(opts *Options) (Publisher, error) {
	switch opts.Broker {
	case AmqpBroker:
		return amqp.NewPublisher(opts.AmqpURI, opts.AmqpQueue), nil
	default:
		return nil, fmt.Errorf("%v is not supported", opts.Broker)
	}
}
