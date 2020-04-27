package amqp

import (
	"encoding/json"

	"github.com/streadway/amqp"
)

type producer struct {
	conn      *amqp.Connection
	queueName string
}

func NewProducer(uri, queueName string) (*producer, error) {
	conn, err := amqp.Dial(uri)
	if err != nil {
		return nil, err
	}

	return &producer{
		conn:      conn,
		queueName: queueName,
	}, nil
}

func (p *producer) Publish(event interface{}) error {
	data, err := json.Marshal(event)
	if err != nil {
		return err
	}

	ch, err := p.conn.Channel()
	if err != nil {
		return err
	}
	defer ch.Close()

	q, err := ch.QueueDeclare(
		p.queueName,
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return err
	}

	return ch.Publish(
		"",
		q.Name,
		true,
		false,
		amqp.Publishing{
			ContentType: "application/json",
			Body:        data,
		},
	)
}

func (p *producer) Close() error {
	return p.conn.Close()
}
