package invalidation

import (
	"context"
	"fmt"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/rs/zerolog"
	"google.golang.org/protobuf/proto"
)

type NatsHelper struct {
	log        zerolog.Logger
	connection *nats.Conn
	prefix     string
}

type resetter interface {
	Reset()
}

func NewNatsHelper(
	log zerolog.Logger,
	connection *nats.Conn,
	prefix string,
) (h *NatsHelper) {
	h = &NatsHelper{
		log:        log,
		connection: connection,
		prefix:     prefix,
	}
	return
}

// Publish broadcasts NATS message.
// Returns error when message was not broadcasted. Otherwise
// returns nil.
func (h *NatsHelper) Publish(subject string, data proto.Message) (err error) {
	var d []byte
	d, err = proto.Marshal(data)
	if err != nil {
		err = fmt.Errorf("cannot marshal data: %w", err)
		return err
	}

	err = h.connection.Publish(h.prefix+subject, d)
	if err != nil {
		err = fmt.Errorf("cannot publish message: %w", err)
		return
	}

	return
}

// Subscribe receives messages from NATS and with each message
// calls `cb` function until ctx is done.
// When Subscribe fails, function automatically tries to subscribe again
// after 31 seconds until it succeeds or ctx is done.
func (h *NatsHelper) Subscribe(ctx context.Context, subject string, protoMsg proto.Message, cb func(proto.Message)) {
	if ctx.Err() != nil {
		return
	}

	subs, err := h.connection.Subscribe(h.prefix+subject, func(natsMsg *nats.Msg) {
		msgReset, ok := protoMsg.(resetter)
		if ok {
			msgReset.Reset()
		}

		msgData := natsMsg.Data
		err := proto.Unmarshal(msgData, protoMsg)
		if err != nil {
			h.log.Warn().
				Err(err).
				Str("subject", subject).
				Msg("cannot unmarshal proto message")
			return
		}

		response := proto.Clone(protoMsg)
		h.log.Trace().
			Str("subject", subject).
			Msg("invalidation received")
		cb(response)
	})
	if err != nil {
		h.log.Error().
			Err(err).
			Str("subject", subject).
			Msg("cannot subscribe to NATS server")
		_ = subs.Unsubscribe() // ignore error
		// try to subscribe again after 31 sec
		time.AfterFunc(31*time.Second, func() {
			h.Subscribe(ctx, subject, protoMsg, cb)
		})
		return
	}

	context.AfterFunc(ctx, func() {
		_ = subs.Unsubscribe()
	})
}
