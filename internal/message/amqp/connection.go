package amqp

import (
	"io"

	"github.com/streadway/amqp"
)

var (
	defaultDialer AMQPDialer = amqp.Dial

	defaultDialerAdapter AMQPDialerAdapter = func(url string) (AMQPConnection, error) {
		conn, err := defaultDialer(url)
		if err != nil {
			return nil, err
		}

		return ConnectionAdapter{conn}, nil
	}
)

type AMQPChannel interface {
	io.Closer

	QueueDeclare(name string, durable bool, autoDelete bool, exclusive bool, noWait bool, args amqp.Table) (amqp.Queue, error)
	Publish(exchange string, key string, mandatory bool, immediate bool, msg amqp.Publishing) error
}

type AMQPConnection interface {
	io.Closer

	Channel() (AMQPChannel, error)
	NotifyClose(receiver chan *amqp.Error) chan *amqp.Error
}

type ConnectionAdapter struct {
	*amqp.Connection
}

func (c ConnectionAdapter) Channel() (AMQPChannel, error) {
	return c.Connection.Channel()
}

type AMQPDialer func(url string) (*amqp.Connection, error)

type AMQPDialerAdapter func(url string) (AMQPConnection, error)
