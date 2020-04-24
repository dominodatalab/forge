package message

import (
	"fmt"

	"github.com/dominodatalab/forge/pkg/message/amqp"
)

const AmqpBroker Broker = "amqp"

var SupportedBrokers = []Broker{AmqpBroker}

type Broker string

type Producer interface {
	Publish(message interface{}) error
	Close() error
}

func NewProducer(opts *Options) (Producer, error) {
	switch opts.Broker {
	case AmqpBroker:
		return amqp.NewProducer(opts.AmqpURI, opts.AmqpQueue)
	default:
		return nil, fmt.Errorf("%v is not supported", opts.Broker)
	}
}
