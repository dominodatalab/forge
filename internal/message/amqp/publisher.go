package amqp

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	"github.com/streadway/amqp"
)

const (
	connectionRetryLimit = 5

	queueDurable    = true
	queueAutoDelete = false
	queueExclusive  = false
	queueNoWait     = false

	amqpExchange     = ""
	publishMandatory = true
	publishImmediate = false
)

var (
	connectionRetryDelay = 5 * time.Second
	queueArgs            = amqp.Table{
		"x-single-active-consumer": true,
	}
)

type publisher struct {
	log       logr.Logger
	uri       string
	queueName string

	conn    Connection
	channel Channel
	err     chan error
}

func NewPublisher(uri, name string, logger logr.Logger) (*publisher, error) {
	p := &publisher{
		log:       logger.WithName("MessagePublisher"),
		uri:       uri,
		queueName: name,
		err:       make(chan error),
	}

	if err := p.connect(); err != nil {
		return nil, err
	}
	return p, nil
}

func (p *publisher) Push(event interface{}) error {
	select {
	case <-p.err:
		p.log.Info("attempting to reconnect to rabbitmq", "uri", p.uri)

		if err := p.connect(); err != nil {
			return errors.Wrap(err, "could not reconnect to rabbitmq")
		}
	default:
	}

	data, err := json.Marshal(event)
	if err != nil {
		return errors.Wrap(err, "cannot marshal rabbitmq event")
	}

	q, err := p.channel.QueueDeclare(
		p.queueName,
		queueDurable,
		queueAutoDelete,
		queueExclusive,
		queueNoWait,
		queueArgs,
	)
	if err != nil {
		return errors.Wrap(err, "failed to declare rabbitmq queue")
	}

	message := amqp.Publishing{
		ContentType: "application/json",
		Body:        data,
	}
	err = p.channel.Publish(amqpExchange, q.Name, publishMandatory, publishImmediate, message)
	return errors.Wrap(err, "failed to publish rabbitmq message")
}

func (p *publisher) Close() error {
	if p.conn != nil {
		return p.conn.Close()
	}
	return nil
}

func (p *publisher) connect() error {
	ticker := time.NewTicker(connectionRetryDelay)
	defer ticker.Stop()

	for counter := 0; counter < connectionRetryLimit; <-ticker.C {
		var err error

		p.conn, err = defaultDialerAdapter(p.uri)
		if err != nil {
			p.log.Error(err, "cannot dial rabbitmq", "uri", p.uri, "attempt", counter+1)

			counter++
			continue
		}

		go func() {
			closed := make(chan *amqp.Error, 1)
			p.conn.NotifyClose(closed)

			reason, ok := <-closed
			if ok {
				p.log.Error(reason, "rabbitmq connection closed, registering err signal")
				p.err <- reason
			}
		}()

		p.channel, err = p.conn.Channel()
		return errors.Wrapf(err, "failed to create rabbitmq channel to %q", p.uri)
	}

	return fmt.Errorf("rabbitmq connection retry limit reached: %d", connectionRetryLimit)
}
