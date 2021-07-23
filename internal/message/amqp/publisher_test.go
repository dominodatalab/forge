package amqp

import (
	"errors"
	"testing"
	"time"

	"github.com/go-logr/zapr"
	"github.com/streadway/amqp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

var (
	uri       = "amqp://test-rabbitmq:5672/"
	logger    = zapr.NewLogger(zap.NewNop())
	queueName = "test-queue"
)

type connectFixture struct {
	adapter    *mockDialerAdapter
	connection *mockConnection
	channel    *mockChannel
}

func setupConnect(fn func(adapter *mockDialerAdapter, conn *mockConnection, channel *mockChannel)) (*connectFixture, func()) {
	mockChan := &mockChannel{}
	mockConn := &mockConnection{}
	mockAdapter := &mockDialerAdapter{}

	fn(mockAdapter, mockConn, mockChan)

	origAdapter := defaultDialerAdapter
	origRetryDelay := connectionRetryDelay

	defaultDialerAdapter = mockAdapter.Dial
	connectionRetryDelay = 1 * time.Nanosecond

	fixture := &connectFixture{
		adapter:    mockAdapter,
		connection: mockConn,
		channel:    mockChan,
	}
	reset := func() {
		defaultDialerAdapter = origAdapter
		connectionRetryDelay = origRetryDelay
	}
	return fixture, reset
}

func TestNewPublisher(t *testing.T) {
	t.Run("connect", func(t *testing.T) {
		f, reset := setupConnect(func(adapter *mockDialerAdapter, conn *mockConnection, channel *mockChannel) {
			conn.On("Channel").Return(channel, nil)
			adapter.On("Dial", uri).Return(conn, nil)
		})
		defer reset()

		actual, err := NewPublisher(uri, queueName, logger)
		require.NoError(t, err)
		assert.NotNil(t, actual.conn)
		assert.NotNil(t, actual.channel)
		assert.Equal(t, queueName, actual.queueName)
		assert.Equal(t, uri, actual.uri)

		f.adapter.AssertExpectations(t)
		f.connection.AssertExpectations(t)
	})

	t.Run("reconnect", func(t *testing.T) {
		f, reset := setupConnect(func(adapter *mockDialerAdapter, conn *mockConnection, channel *mockChannel) {
			conn.On("Channel").Return(channel, nil)
			adapter.On("Dial", uri).Return(nil, errors.New("test dial error")).Once()
			adapter.On("Dial", uri).Return(conn, nil).Once()
		})
		defer reset()

		actual, err := NewPublisher(uri, queueName, logger)
		require.NoError(t, err)
		assert.NotNil(t, actual.conn)
		assert.NotNil(t, actual.channel)
		assert.Equal(t, queueName, actual.queueName)
		assert.Equal(t, uri, actual.uri)

		f.adapter.AssertExpectations(t)
		f.adapter.AssertNumberOfCalls(t, "Dial", 2)
		f.connection.AssertExpectations(t)
	})

	t.Run("channel_failure", func(t *testing.T) {
		f, reset := setupConnect(func(adapter *mockDialerAdapter, conn *mockConnection, channel *mockChannel) {
			conn.On("Channel").Return(nil, errors.New("test channel failure"))
			adapter.On("Dial", uri).Return(conn, nil)
		})
		defer reset()

		_, err := NewPublisher(uri, queueName, logger)
		assert.Error(t, err)

		f.adapter.AssertExpectations(t)
		f.connection.AssertExpectations(t)
	})

	t.Run("retry_limit_failure", func(t *testing.T) {
		f, reset := setupConnect(func(adapter *mockDialerAdapter, conn *mockConnection, channel *mockChannel) {
			adapter.On("Dial", uri).Return(nil, errors.New("test dial error"))
		})
		defer reset()

		_, err := NewPublisher(uri, queueName, logger)
		assert.Error(t, err)

		f.adapter.AssertExpectations(t)
		f.adapter.AssertNumberOfCalls(t, "Dial", connectionRetryLimit)
	})
}

func TestPublisher_Push(t *testing.T) {
	testMessage := "hello"

	t.Run("success", func(t *testing.T) {
		mockChan := &mockChannel{}

		mockChan.On("QueueDeclare", queueName, true, false, false, false, amqp.Table{
			"x-single-active-consumer": true,
		}).Return(amqp.Queue{Name: queueName}, nil)

		mockChan.On("Publish", "", queueName, true, false, amqp.Publishing{
			ContentType: "application/json",
			Body:        []byte(`"hello"`),
		}).Return(nil)

		pub := &publisher{
			channel:   mockChan,
			queueName: queueName,
		}

		assert.NoError(t, pub.Push(testMessage))

		mockChan.AssertExpectations(t)
	})

	t.Run("publish_failure", func(t *testing.T) {
		mockChan := &mockChannel{}

		mockChan.On("QueueDeclare", queueName, true, false, false, false, amqp.Table{
			"x-single-active-consumer": true,
		}).Return(amqp.Queue{Name: queueName}, nil)

		mockChan.On("Publish", "", queueName, true, false, amqp.Publishing{
			ContentType: "application/json",
			Body:        []byte(`"hello"`),
		}).Return(errors.New("test error"))

		pub := &publisher{
			channel:   mockChan,
			queueName: queueName,
		}

		assert.EqualError(t, pub.Push(testMessage), "failed to publish rabbitmq message: test error")

		mockChan.AssertExpectations(t)
	})

	t.Run("queue_declare_failure", func(t *testing.T) {
		mockChan := &mockChannel{}

		mockChan.On("QueueDeclare", queueName, true, false, false, false, amqp.Table{
			"x-single-active-consumer": true,
		}).Return(amqp.Queue{}, errors.New("test error"))

		pub := &publisher{
			channel:   mockChan,
			queueName: queueName,
		}

		assert.EqualError(t, pub.Push(testMessage), "failed to declare rabbitmq queue: test error")

		mockChan.AssertExpectations(t)
	})

	t.Run("bad_input", func(t *testing.T) {
		badType := make(chan int)
		defer close(badType)
		pub := &publisher{}

		assert.EqualError(t, pub.Push(badType), "cannot marshal rabbitmq event: json: unsupported type: chan int")
	})

	t.Run("connection_closed", func(t *testing.T) {
		f, reset := setupConnect(func(adapter *mockDialerAdapter, conn *mockConnection, channel *mockChannel) {
			channel.On("QueueDeclare", queueName, true, false, false, false, amqp.Table{
				"x-single-active-consumer": true,
			}).Return(amqp.Queue{Name: queueName}, nil)

			channel.On("Publish", "", queueName, true, false, amqp.Publishing{
				ContentType: "application/json",
				Body:        []byte(`"hello"`),
			}).Return(nil)

			conn.On("Channel").Return(channel, nil)
			adapter.On("Dial", uri).Return(conn, nil)
		})
		defer reset()

		pub := &publisher{
			uri:       uri,
			log:       logger,
			queueName: queueName,
			err:       make(chan error, 1),
		}

		pub.err <- errors.New("dang, conn be broke")

		assert.NoError(t, pub.Push(testMessage))

		f.adapter.AssertExpectations(t)
		f.connection.AssertExpectations(t)
		f.channel.AssertExpectations(t)
	})

	t.Run("connection_closed_retry", func(t *testing.T) {
		f, reset := setupConnect(func(adapter *mockDialerAdapter, conn *mockConnection, channel *mockChannel) {
			channel.On("QueueDeclare", queueName, true, false, false, false, amqp.Table{
				"x-single-active-consumer": true,
			}).Return(amqp.Queue{Name: queueName}, nil)

			channel.On("Publish", "", queueName, true, false, amqp.Publishing{
				ContentType: "application/json",
				Body:        []byte(`"hello"`),
			}).Return(nil)

			conn.On("Channel").Return(channel, nil)

			adapter.On("Dial", uri).Return(nil, errors.New("test dial error")).Once()
			adapter.On("Dial", uri).Return(conn, nil).Once()
		})
		defer reset()

		pub := &publisher{
			uri:       uri,
			log:       logger,
			queueName: queueName,
			err:       make(chan error, 1),
		}

		pub.err <- errors.New("dang, conn be broke")

		assert.NoError(t, pub.Push(testMessage))

		f.adapter.AssertExpectations(t)
		f.connection.AssertExpectations(t)
		f.channel.AssertExpectations(t)
	})

	t.Run("connection_closed_retry_failure", func(t *testing.T) {
		f, reset := setupConnect(func(adapter *mockDialerAdapter, conn *mockConnection, channel *mockChannel) {
			adapter.On("Dial", uri).Return(nil, errors.New("test dial error"))
		})
		defer reset()

		pub := &publisher{
			uri:       uri,
			log:       logger,
			queueName: queueName,
			err:       make(chan error, 1),
		}

		pub.err <- errors.New("dang, conn be broke")

		assert.Error(t, pub.Push(testMessage))

		f.adapter.AssertExpectations(t)
	})
}

func TestPublisher_Close(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mockConn := &mockConnection{}
		mockConn.On("Close").Return(nil)
		pub := &publisher{
			conn: mockConn,
		}

		assert.NoError(t, pub.Close())
		mockConn.AssertExpectations(t)
	})

	t.Run("failure", func(t *testing.T) {
		mockConn := &mockConnection{}
		mockConn.On("Close").Return(errors.New("test failed to close connection"))
		pub := &publisher{
			conn: mockConn,
		}

		assert.EqualError(t, pub.Close(), "test failed to close connection")
		mockConn.AssertExpectations(t)
	})

	t.Run("no_connection", func(t *testing.T) {
		assert.NoError(t, (&publisher{}).Close())
	})
}
