package message

import (
	"fmt"
	"github.com/go-logr/logr"

	"github.com/dominodatalab/forge/internal/message/amqp"
)

// Broker represents a messaging protocol implementation.
type Broker string

// AmqpBroker publishes messages using the AMQP protocol.
const AmqpBroker Broker = "amqp"

// SupportedBrokers defines the list of implemented message publishers.
var SupportedBrokers = []Broker{AmqpBroker}

// Publisher defines the operations required by all message producers.
type Publisher interface {
	Push(event interface{}) error
	Close() error
}

// NewPublisher configures a new message producer using the provided options.
func NewPublisher(opts *Options, log logr.Logger) (Publisher, error) {
	switch opts.Broker {
	case AmqpBroker:
		return amqp.NewPublisher(opts.AmqpURI, opts.AmqpQueue, log), nil
	default:
		return nil, fmt.Errorf("%v is not supported", opts.Broker)
	}
}
