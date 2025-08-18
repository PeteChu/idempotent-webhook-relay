package queue

import (
	"errors"
	"fmt"

	"github.com/rabbitmq/amqp091-go"
)

type Queue struct {
	amqp091.Queue
	Connection *amqp091.Connection
	Channel    *amqp091.Channel
	Close      func() error
}

type Options struct {
	Name       string
	Durable    bool
	AutoDelete bool
	Exclusive  bool
	NoWait     bool
	Args       amqp091.Table
}

type Option func(*Options)

func NewQueue(url string, opts ...Option) (*Queue, error) {
	conn, err := amqp091.Dial(url)
	if err != nil {
		return nil, err
	}

	ch, err := conn.Channel()
	if err != nil {
		return nil, err
	}

	cfg := Options{
		Name:       "default",
		Durable:    false,
		AutoDelete: false,
		Exclusive:  false,
		NoWait:     false,
		Args:       nil,
	}
	for _, opt := range opts {
		opt(&cfg)
	}

	q, err := ch.QueueDeclare(
		cfg.Name,
		cfg.Durable,
		cfg.AutoDelete,
		cfg.Exclusive,
		cfg.NoWait,
		cfg.Args,
	)
	if err != nil {
		return nil, err
	}

	return &Queue{
		Queue:      q,
		Connection: conn,
		Channel:    ch,
		Close: func() error {
			var errs error
			if err := ch.Close(); err != nil {
				errs = errors.Join(errs, fmt.Errorf("failed to close channel: %w", err))
			}
			if err := conn.Close(); err != nil {
				errs = errors.Join(errs, fmt.Errorf("failed to close connection: %w", err))
			}
			return errs
		},
	}, nil
}

func WithConfig(opts Options) Option {
	return func(o *Options) {
		if opts.Name == "" {
			o.Name = "default"
		} else {
			o.Name = opts.Name
		}
		o.Durable = opts.Durable
		o.AutoDelete = opts.AutoDelete
		o.Exclusive = opts.Exclusive
		o.NoWait = opts.NoWait
		o.Args = opts.Args
	}
}
