package amqp

import (
	"io"

	"github.com/streadway/amqp"
)

var defaultDialerAdapter DialerAdapter = func(url string) (Connection, error) {
	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, err
	}

	return ConnectionAdapter{conn}, nil
}

type DialerAdapter func(url string) (Connection, error)

type Connection interface {
	io.Closer

	Channel() (Channel, error)
	NotifyClose(receiver chan *amqp.Error) chan *amqp.Error
}

type ConnectionAdapter struct {
	*amqp.Connection
}

func (c ConnectionAdapter) Channel() (Channel, error) {
	return c.Connection.Channel()
}

type Channel interface {
	QueueDeclare(name string, durable bool, autoDelete bool, exclusive bool, noWait bool, args amqp.Table) (amqp.Queue, error)
	Publish(exchange string, key string, mandatory bool, immediate bool, msg amqp.Publishing) error
}
