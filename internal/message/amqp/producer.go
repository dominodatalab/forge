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

// producer represents a connection to a particular queue
type Queue struct {
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
func NewQueue(uri, queueName string) *Queue {
	q:= Queue{
		name:          queueName,
		logger:        log.New(os.Stdout, "", log.LstdFlags),
		done:          make(chan bool),
	}
	go q.handleReconnect(uri)
	return &q
}

func (q *Queue) handleReconnect(uri string) {
	for {
		q.isConnected = false
		log.Println("Attempting to connect")
		for !q.connect(uri) {
			log.Println("Failed to connect. Retrying...")
			time.Sleep(reconnectDelay)
		}
		select {
		case <-q.done:
			return
		case <-q.notifyClose:
		}
	}
}

// create wrapper around amqp Dial function
func (q *Queue) connect(uri string) bool {

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
		q.name,
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

	q.changeConnection(conn, ch)
	q.isConnected = true
	log.Println("Connected!")
	return true
}

// changeConnection takes a new connection to the queue,
// and updates the channel listeners to reflect this.
func (q *Queue) changeConnection(conn *amqp.Connection, ch *amqp.Channel) {
	q.connection = conn
	q.channel = ch
	q.notifyClose = make(chan *amqp.Error)
	q.notifyConfirm = make(chan amqp.Confirmation)
	q.channel.NotifyClose(q.notifyClose)
	q.channel.NotifyPublish(q.notifyConfirm)
}

// Push will push data onto the queue, and wait for a confirm.
// If no confirms are recieved until within the resendTimeout,
// it continuously resends messages until a confirm is recieved.
// This will block until the server sends a confirm. Errors are
// only returned if the push action itself fails, see UnsafePush.
func (q *Queue) Push(event interface{}) error {
	if !q.isConnected {
		return errors.New("failed to push push: not connected")
	}

	for {
		err := q.UnsafePush(event)
		if err != nil {
			q.logger.Println("Push failed. Retrying...")
			continue
		}
		select {
		case confirm := <-q.notifyConfirm:
			if confirm.Ack {
				q.logger.Println("Push confirmed!")
				return nil
			}
		case <-time.After(resendDelay):
		}
		q.logger.Println("Push didn't confirm. Retrying...")
	}
}

// UnsafePush will push to the queue without checking for
// confirmation. It returns an error if it fails to connect.
// No guarantees are provided for whether the server will
// recieve the message.
func (q *Queue) UnsafePush(event interface{}) error {
	data, err := json.Marshal(event)
	if err != nil {
		return err
	}

	if !q.isConnected {
		return errNotConnected
	}

	return q.channel.Publish(
		"",     // Exchange
		q.name, // Routing key
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
func (q *Queue) Stream() (<-chan amqp.Delivery, error) {
	if !q.isConnected {
		return nil, errNotConnected
	}
	return q.channel.Consume(
		q.name,
		"",    // Consumer
		false, // Auto-Ack
		false, // Exclusive
		false, // No-local
		false, // No-Wait
		nil,   // Args
	)
}

// Close will cleanly shutdown the channel and connection.
func (q *Queue) Close() error {
	if !q.isConnected {
		return errAlreadyClosed
	}
	err := q.channel.Close()
	if err != nil {
		return err
	}
	err = q.connection.Close()
	if err != nil {
		return err
	}
	close(q.done)
	q.isConnected = false
	return nil
}
