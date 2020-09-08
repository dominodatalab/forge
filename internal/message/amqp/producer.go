package amqp

import (
	"encoding/json"
	"errors"
	"log"
	"os"
	"time"

	"github.com/streadway/amqp"
)

const (
	// When reconnecting to the server after connection failure
	reconnectDelay = 5 * time.Second

	// When resending messages the server didn't confirm
	resendDelay = 5 * time.Second
)

var (
	errNotConnected  = errors.New("not connected to the queue")
	errNotConfirmed  = errors.New("message not confirmed")
	errAlreadyClosed = errors.New("already closed: not connected to the queue")
)

// Publisher represents a connection to a particular queue
type Publisher struct {
	name          string
	logger        *log.Logger
	connection    *amqp.Connection
	channel       *amqp.Channel
	done          chan bool
	isConnected   bool
	notifyClose   chan *amqp.Error
	notifyConfirm chan amqp.Confirmation
}

// create a new producer object and automatically try to connect to the queue
func NewPublisher(uri, queueName string) *Publisher {
	p := Publisher{
		name:          queueName,
		logger:        log.New(os.Stdout, "", log.LstdFlags),
		done:          make(chan bool),
	}
	go p.handleReconnect(uri)
	return &p
}

func (p Publisher) handleReconnect(uri string) {
	for {
		p.isConnected = false
		log.Println("Attempting to connect")
		for !p.connect(uri) {
			log.Println("Failed to connect. Retrying...")
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

	ch.Confirm(false)

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
	log.Println("Connected!")
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

// Push will push data onto the queue, and wait for a confirm.
// If no confirms are received until within the resendTimeout,
// it continuously resends messages until a confirm is recieved.
// This will block until the server sends a confirm. Errors are
// only returned if the push action itself fails, see UnsafePush.
func (p *Publisher) Push(event interface{}) error {
	if !p.isConnected {
		return errors.New("failed to push push: not connected")
	}

	for {
		err := p.UnsafePush(event)
		if err != nil {
			p.logger.Println("Push failed. Retrying...")
			continue
		}
		select {
		case confirm := <-p.notifyConfirm:
			if confirm.Ack {
				p.logger.Println("Push confirmed!")
				return nil
			}
		case <-time.After(resendDelay):
		}
		p.logger.Println("Push didn't confirm. Retrying...")
	}
}

// UnsafePush will push to the queue without checking for
// confirmation. It returns an error if it fails to connect.
func (p *Publisher) UnsafePush(event interface{}) error {
	data, err := json.Marshal(event)
	if err != nil {
		return err
	}

	if !p.isConnected {
		return errNotConnected
	}

	return p.channel.Publish(
		"",     // Exchange
		p.name, // Routing key
		false,  // Mandatory
		false,  // Immediate
		amqp.Publishing{
			ContentType: "application/json",
			Body:        data,
		},
	)
}

// Stream will continuously put queue items on the channel.
// It is required to call delivery.Ack when it has been
// successfully processed, or delivery.Nack when it fails.
// Ignoring this will cause data to build up on the server.
func (p *Publisher) Stream() (<-chan amqp.Delivery, error) {
	if !p.isConnected {
		return nil, errNotConnected
	}
	return p.channel.Consume(
		p.name,
		"",    // Consumer
		false, // Auto-Ack
		false, // Exclusive
		false, // No-local
		false, // No-Wait
		nil,   // Args
	)
}

// Close will cleanly shutdown the channel and connection.
func (p *Publisher) Close() error {
	if !p.isConnected {
		return errAlreadyClosed
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
