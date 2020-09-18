package amqp

import (
	"errors"
	"testing"
	"time"

	"github.com/streadway/amqp"

	"github.com/go-logr/zapr"
	"go.uber.org/zap"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	uri       = "amqp://test-rabbitmq:5672/"
	logger    = zapr.NewLogger(zap.NewNop())
	queueName = "test-queue"
)

type testConnectConfig struct {
	withMocks func(conn *fakeConnection, channel *fakeChannel)
	operation func(expectErr bool)
}

func testConnect(t *testing.T, config testConnectConfig) {
	t.Helper()

	originalDelay := connectionRetryDelay
	originalDialer := defaultDialerAdapter
	defer func() {
		connectionRetryDelay = originalDelay
		defaultDialerAdapter = originalDialer
	}()

	connectionRetryDelay = 1 * time.Nanosecond

	t.Run("connect success", func(t *testing.T) {
		mockChannel := &fakeChannel{}
		mockConn := &fakeConnection{}
		mockConn.On("Channel").Return(mockChannel, nil)

		mockDialerAdapter := fakeDialerAdapter{}
		mockDialerAdapter.On("Dial", uri).Return(mockConn, nil)
		defaultDialerAdapter = mockDialerAdapter.Dial

		config.withMocks(mockConn, mockChannel)
		config.operation(false)

		mockDialerAdapter.AssertExpectations(t)
		mockConn.AssertExpectations(t)
	})

	t.Run("channel failure", func(t *testing.T) {

		mockConn := &fakeConnection{}
		mockConn.On("Channel").Return(nil, errors.New("test channel failure"))

		mockDialerAdapter := fakeDialerAdapter{}
		mockDialerAdapter.On("Dial", uri).Return(mockConn, nil)
		defaultDialerAdapter = mockDialerAdapter.Dial

		config.withMocks(mockConn, nil)
		config.operation(true)

		mockDialerAdapter.AssertExpectations(t)
		mockConn.AssertExpectations(t)
	})

	t.Run("retry limit failure", func(t *testing.T) {
		mock := fakeDialerAdapter{}
		mock.On("Dial", uri).Return(nil, errors.New("test dial error"))
		defaultDialerAdapter = mock.Dial

		config.withMocks(nil, nil)
		config.operation(true)

		mock.AssertExpectations(t)
		mock.AssertNumberOfCalls(t, "Dial", connectionRetryLimit)
	})

	t.Run("reconnect success", func(t *testing.T) {
		mockChannel := &fakeChannel{}
		mockConn := &fakeConnection{}
		mockConn.On("Channel").Return(mockChannel, nil)

		mockDialerAdapter := fakeDialerAdapter{}
		mockDialerAdapter.On("Dial", uri).Return(nil, errors.New("test dial error")).Once()
		mockDialerAdapter.On("Dial", uri).Return(mockConn, nil).Once()

		defaultDialerAdapter = mockDialerAdapter.Dial

		config.withMocks(mockConn, mockChannel)
		config.operation(false)

		mockDialerAdapter.AssertExpectations(t)
		mockDialerAdapter.AssertNumberOfCalls(t, "Dial", 2)
	})
}

func TestNewPublisher(t *testing.T) {
	/*
		1. Create a new publisher successfully
		2. Create a new publisher and fail

		connect:
		success states:
		successful dial
		retry success

		failure states:
		retry limit failure / unsuccessful dial
		connection success channel failure

	*/

	config := testConnectConfig{
		operation: func(expectErr bool) {
			actual, err := NewPublisher(uri, queueName, logger)
			if expectErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.NotNil(t, actual.conn)
			assert.NotNil(t, actual.channel)
			assert.Equal(t, queueName, actual.queueName)
			assert.Equal(t, uri, actual.uri)
		},
	}
	testConnect(t, config)
}

func TestPublisher_Close(t *testing.T) {
	/*
		1. close successfully
		2. close unsuccessfully
	*/

	t.Run("success", func(t *testing.T) {
		fakeConn := &fakeConnection{}
		fakeConn.On("Close").Return(nil)
		p := &publisher{
			conn: fakeConn,
		}

		assert.NoError(t, p.Close())
		fakeConn.AssertExpectations(t)
	})

	t.Run("no_connection", func(t *testing.T) {
		p := &publisher{}

		assert.NoError(t, p.Close())
	})

	t.Run("failure", func(t *testing.T) {
		fakeConn := &fakeConnection{}
		fakeConn.On("Close").Return(errors.New("test failed to close connection"))
		p := &publisher{
			conn: fakeConn,
		}

		assert.EqualError(t, p.Close(), "test failed to close connection")
		fakeConn.AssertExpectations(t)
	})
}

func TestPublisher_Push(t *testing.T) {

	var (
		event           = ""
		queueDurable    = true
		queueAutoDelete = false
		queueExclusive  = false
		queueNoWait     = false
	)

	t.Run("reconnect connection failure", func(t *testing.T) {
		//mockConn := &fakeConnection{}
		//mockChan := &fakeChannel{}
		//mockChan.On("QueueDeclare", queueName, queueDurable, queueAutoDelete, queueExclusive, queueNoWait, queueArgs).Return(
		//	amqp.Queue{}, nil)
		//mockConn.On("Channel").Return(mockChan, nil)

		p := &publisher{
			log:       logger,
			uri:       uri,
			queueName: queueName,
			err:       make(chan error, 1),
		}
		go func() {
			p.err <- errors.New("test failed")
		}()

		config := testConnectConfig{
			withMocks: func(conn *fakeConnection, channel *fakeChannel) {
				if channel != nil {
					channel.On(
						"QueueDeclare",
						queueName,
						queueDurable,
						queueAutoDelete,
						queueExclusive,
						queueNoWait,
						queueArgs,
					).Return(amqp.Queue{}, nil)
					channel.On("Publish", "", "", true, false, amqp.Publishing{ContentType: "application/json", Body: []byte(`""`)}).Return(nil)
				}
			},
			operation: func(expectErr bool) {
				err := p.Push(event)
				if expectErr {
					assert.Error(t, err)
					return
				}

				require.NoError(t, err)
			},
		}
		testConnect(t, config)
	})

	t.Run("failed to marshal rmq event", func(t *testing.T) {
		fakeConn := &fakeConnection{}
		p := &publisher{
			conn: fakeConn,
		}

		event := make(chan bool)

		err := p.Push(event)
		assert.Error(t, err)
	})

	t.Run("failed to declare queue", func(t *testing.T) {
		mockConn := &fakeConnection{}
		mockChan := &fakeChannel{}
		mockChan.On("QueueDeclare", queueName, queueDurable, queueAutoDelete, queueExclusive, queueNoWait, queueArgs).Return(
			nil, errors.New("failed to create queue"))
		mockConn.On("Channel").Return(
			&fakeChannel{}, nil)
		p := &publisher{
			conn: mockConn,
		}

		err := p.Push(event)
		assert.NoError(t, err)
	})

	t.Run("failed to publish", func(t *testing.T) {
		mockConn := &fakeConnection{}
		mockChan := &fakeChannel{}
		mockChan.On("QueueDeclare", queueName, queueDurable, queueAutoDelete, queueExclusive, queueNoWait, queueArgs).Return(
			nil, errors.New("failed to create queue"))
		mockConn.On("Channel").Return(mockChan, nil)

		p := &publisher{
			conn: mockConn,
		}

		err := p.Push(event)
		assert.NoError(t, err)
	})

	t.Run("success", func(t *testing.T) {
		mockConn := &fakeConnection{}
		mockConn.On("Channel").Return(&fakeChannel{}, nil)
		p := &publisher{
			conn: mockConn,
		}

		assert.NoError(t, p.Push(event))
	})
}
