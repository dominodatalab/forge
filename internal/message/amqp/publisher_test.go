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

func TestNewPublisher(t *testing.T) {
	setup := func(fn func(adapter *mockDialerAdapter, conn *mockConnection, channel *mockChannel)) (*mockDialerAdapter, *mockConnection, func()) {
		mockChan := &mockChannel{}
		mockConn := &mockConnection{}
		mockAdapter := &mockDialerAdapter{}

		fn(mockAdapter, mockConn, mockChan)

		origAdapter := defaultDialerAdapter
		origRetryDelay := connectionRetryDelay

		defaultDialerAdapter = mockAdapter.Dial
		connectionRetryDelay = 1 * time.Nanosecond

		reset := func() {
			defaultDialerAdapter = origAdapter
			connectionRetryDelay = origRetryDelay
		}

		return mockAdapter, mockConn, reset
	}

	t.Run("connect", func(t *testing.T) {
		adapter, conn, reset := setup(func(adapter *mockDialerAdapter, conn *mockConnection, channel *mockChannel) {
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

		adapter.AssertExpectations(t)
		conn.AssertExpectations(t)
	})

	t.Run("reconnect", func(t *testing.T) {
		adapter, _, reset := setup(func(adapter *mockDialerAdapter, conn *mockConnection, channel *mockChannel) {
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

		adapter.AssertExpectations(t)
		adapter.AssertNumberOfCalls(t, "Dial", 2)
	})

	t.Run("channel_failure", func(t *testing.T) {
		adapter, conn, reset := setup(func(adapter *mockDialerAdapter, conn *mockConnection, channel *mockChannel) {
			conn.On("Channel").Return(nil, errors.New("test channel failure"))
			adapter.On("Dial", uri).Return(conn, nil)
		})
		defer reset()

		_, err := NewPublisher(uri, queueName, logger)
		assert.Error(t, err)

		adapter.AssertExpectations(t)
		conn.AssertExpectations(t)
	})

	t.Run("retry_limit_failure", func(t *testing.T) {
		adapter, _, reset := setup(func(adapter *mockDialerAdapter, conn *mockConnection, channel *mockChannel) {
			adapter.On("Dial", uri).Return(nil, errors.New("test dial error"))
		})
		defer reset()

		_, err := NewPublisher(uri, queueName, logger)
		assert.Error(t, err)

		adapter.AssertExpectations(t)
		adapter.AssertNumberOfCalls(t, "Dial", connectionRetryLimit)
	})
}

func TestPublisher_Push(t *testing.T) {
	testEvent := "hello"

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

		assert.NoError(t, pub.Push(testEvent))

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

		assert.EqualError(t, pub.Push(testEvent), "failed to publish rabbitmq message: test error")

		mockChan.AssertExpectations(t)
	})

	t.Run("bad_event", func(t *testing.T) {
		badType := make(chan int)
		defer close(badType)
		pub := &publisher{}

		assert.EqualError(t, pub.Push(badType), "cannot marshal rabbitmq event: json: unsupported type: chan int")
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

		assert.EqualError(t, pub.Push(testEvent), "failed to declare rabbitmq queue: test error")

		mockChan.AssertExpectations(t)
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
