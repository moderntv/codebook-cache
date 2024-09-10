package codebook

import (
	"context"
	"errors"

	cadre_metrics "github.com/moderntv/cadre/metrics"
	"github.com/nats-io/nats.go"
	"github.com/rs/zerolog"
	"google.golang.org/protobuf/proto"
)

type LoadAllFunc[K comparable, T any] func(ctx context.Context) (entries map[K]*T, err error)

type Params[K comparable, T any] struct {
	Context         context.Context
	Log             zerolog.Logger
	MetricsRegistry *cadre_metrics.Registry
	Invalidations   *Invalidations
	Name            string
	LoadAllFunc     LoadAllFunc[K, T]
	Timeouts        Timeouts
	MemsizeEnabled  bool
}

func (p *Params[K, T]) check() error {
	if p.Context == nil {
		return errors.New("context must be set")
	}

	if p.Name == "" {
		return errors.New("name must be set")
	}

	if p.LoadAllFunc == nil {
		return errors.New("LoadAllFunc must be provided")
	}

	err := p.Timeouts.check()
	if err != nil {
		return err
	}

	if p.Invalidations != nil {
		return p.Invalidations.check()
	}

	return nil
}

type Invalidations struct {
	Nats     *nats.Conn
	Prefix   string
	Messages map[string]proto.Message
}

func (i *Invalidations) check() error {
	if i.Nats == nil {
		return errors.New("no nats connection specified")
	}

	if len(i.Messages) == 0 {
		return errors.New("empty invalidation messages")
	}

	return nil
}
