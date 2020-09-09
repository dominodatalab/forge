package amqp

import (
	"encoding/json"
	"errors"
	"github.com/go-logr/logr"
	"time"

	"github.com/streadway/amqp"
)

const (
	reconnectDelay = 5 * time.Second
	resendDelay = 5 * time.Second
)

// Publisher represents a connection to a particular queue
type Publisher struct {
	name          string
	logger        logr.Logger
	connection    *amqp.Connection
	channel       *amqp.Channel
	done          chan bool
	isConnected   bool
	notifyClose   chan *amqp.Error
	notifyConfirm chan amqp.Confirmation
}

// create a new publisher object and try to connect to the queue
func NewPublisher(uri, queueName string, log logr.Logger) *Publisher {
	p := Publisher{
		name:          queueName,
		logger:        log,
		done:          make(chan bool),
	}
	go p.handleReconnect(uri)
	return &p
}

func (p Publisher) handleReconnect(uri string) {
	for {
		p.isConnected = false
		p.logger.Info("Attempting to connect")
		for !p.connect(uri) {
			p.logger.Info("Failed to connect. Retrying.")
			time.Sleep(reconnectDelay)
		}
		select {
		case <-p.done:
			return
		case <-p.notifyClose:
		}
	}
}

// wrapper around amqp Dial function
func (p *Publisher) connect(uri string) bool {

	// try and connect to the queue
	conn, err := amqp.Dial(uri)
	if err != nil {
		return false
	}

	ch, err := conn.Channel()
	if err != nil {
		return false
	}

	err = ch.Confirm(false)
	if err != nil {
		return false
	}

	_, err = ch.QueueDeclare(
		p.name,
		true,
		false,
		false,
		false,
		amqp.Table {
			"x-single-active-consumer": true,
		},
	)

	if err != nil {
		return false
	}

	p.changeConnection(conn, ch)
	p.isConnected = true
	p.logger.Info("Connected!")
	return true
}

// changeConnection takes a new connection to the queue,
// and updates the channel listeners to reflect this.
func (p *Publisher) changeConnection(conn *amqp.Connection, ch *amqp.Channel) {
	p.connection = conn
	p.channel = ch
	p.notifyClose = make(chan *amqp.Error)
	p.notifyConfirm = make(chan amqp.Confirmation)
	p.channel.NotifyClose(p.notifyClose)
	p.channel.NotifyPublish(p.notifyConfirm)
}

// will push data onto the queue and wait for a confirm.
// it continuously resends messages until a confirm is received.
func (p *Publisher) Push(event interface{}) error {
	if !p.isConnected {
		return errors.New("failed to push push: not connected")
	}

	for {
		err := p.UnsafePush(event)
		if err != nil {
			p.logger.Info("Push failed. Retrying...")
			continue
		}
		select {
		case confirm := <-p.notifyConfirm:
			if confirm.Ack {
				p.logger.Info("Push confirmed!")
				return nil
			}
		case <-time.After(resendDelay):
		}
		p.logger.Info("Push didn't confirm. Retrying...")
	}
}

// will push to the queue without checking for
// confirmation. It returns an error if it fails to connect.
func (p *Publisher) UnsafePush(event interface{}) error {
	data, err := json.Marshal(event)
	if err != nil {
		return err
	}

	if !p.isConnected {
		return errors.New("not connected to the queue")
	}

	return p.channel.Publish(
		"",
		p.name,
		false,
		false,
		amqp.Publishing{
			ContentType: "application/json",
			Body:        data,
		},
	)
}

// Close will cleanly shutdown the channel and connection.
func (p *Publisher) Close() error {
	if !p.isConnected {
		return errors.New("already closed: not connected to the queue")
	}
	err := p.channel.Close()
	if err != nil {
		return err
	}
	err = p.connection.Close()
	if err != nil {
		return err
	}
	close(p.done)
	p.isConnected = false
	return nil
}
