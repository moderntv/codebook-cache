package invalidation

import (
	"context"
	"testing"
	"time"

	natstest "github.com/nats-io/nats-server/v2/test"
	"github.com/nats-io/nats.go"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/emptypb"
)

func connect(t *testing.T, url string) *nats.Conn {
	connection, err := nats.Connect(url)
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(connection.Close)

	return connection
}

func TestSubscribeEndsWithContext(t *testing.T) {
	server := natstest.RunRandClientPortServer()
	t.Cleanup(server.Shutdown)

	subscriber := connect(t, server.ClientURL())
	publisher := NewNatsHelper(zerolog.Nop(), connect(t, server.ClientURL()), "test.")

	ctx, cancel := context.WithCancel(context.Background())
	received := make(chan struct{}, 1)
	NewNatsHelper(zerolog.Nop(), subscriber, "test.").Subscribe(
		ctx,
		"subject",
		&emptypb.Empty{},
		func(proto.Message) {
			received <- struct{}{}
		},
	)
	assert.Equal(t, 1, subscriber.NumSubscriptions())

	assert.NoError(t, publisher.Publish("subject", &emptypb.Empty{}))
	select {
	case <-received:
	case <-time.After(5 * time.Second):
		t.Fatal("message not received")
	}

	cancel()
	assert.Eventually(t, func() bool {
		return subscriber.NumSubscriptions() == 0
	}, 5*time.Second, 10*time.Millisecond)
}

func TestSubscribeWithDoneContext(t *testing.T) {
	server := natstest.RunRandClientPortServer()
	t.Cleanup(server.Shutdown)

	subscriber := connect(t, server.ClientURL())

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	NewNatsHelper(zerolog.Nop(), subscriber, "test.").Subscribe(
		ctx,
		"subject",
		&emptypb.Empty{},
		func(proto.Message) {},
	)

	assert.Equal(t, 0, subscriber.NumSubscriptions())
}
