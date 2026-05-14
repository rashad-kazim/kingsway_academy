package queue

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"go.uber.org/zap"
)

func DialRabbitMQ(url string) (*amqp.Connection, error) {
	return amqp.Dial(url)
}

type TopicPublisher struct {
	conn     *amqp.Connection
	exchange string
}

func NewTopicPublisher(conn *amqp.Connection, exchange string) *TopicPublisher {
	return &TopicPublisher{conn: conn, exchange: exchange}
}

func (p *TopicPublisher) EnsureExchange(ctx context.Context) error {
	ch, err := p.conn.Channel()
	if err != nil {
		return err
	}
	defer func() { _ = ch.Close() }()

	done := make(chan error, 1)
	go func() {
		done <- ch.ExchangeDeclare(
			p.exchange,
			"topic",
			true,
			false,
			false,
			false,
			nil,
		)
	}()

	select {
	case err := <-done:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (p *TopicPublisher) Publish(ctx context.Context, topic string, payload any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	return p.PublishJSON(ctx, topic, body)
}

func (p *TopicPublisher) PublishJSON(ctx context.Context, topic string, body []byte) error {
	ch, err := p.conn.Channel()
	if err != nil {
		return err
	}
	defer func() { _ = ch.Close() }()

	return ch.PublishWithContext(ctx, p.exchange, topic, false, false, amqp.Publishing{
		ContentType:  "application/json",
		DeliveryMode: amqp.Persistent,
		Timestamp:    time.Now().UTC(),
		Body:         body,
	})
}

type EventHandler func(ctx context.Context, topic string, body []byte) error

type TopicConsumer struct {
	conn     *amqp.Connection
	exchange string
	queue    string
	bindings []string
}

func NewTopicConsumer(conn *amqp.Connection, exchange string, queue string, bindings []string) *TopicConsumer {
	return &TopicConsumer{conn: conn, exchange: exchange, queue: queue, bindings: bindings}
}

func (c *TopicConsumer) Start(ctx context.Context, handler EventHandler, log *zap.Logger) (<-chan struct{}, error) {
	if log == nil {
		log = zap.NewNop()
	}
	ch, err := c.conn.Channel()
	if err != nil {
		return nil, err
	}

	if err := ch.ExchangeDeclare(c.exchange, "topic", true, false, false, false, nil); err != nil {
		_ = ch.Close()
		return nil, err
	}
	queue, err := ch.QueueDeclare(c.queue, true, false, false, false, nil)
	if err != nil {
		_ = ch.Close()
		return nil, err
	}
	for _, binding := range c.bindings {
		if err := ch.QueueBind(queue.Name, binding, c.exchange, false, nil); err != nil {
			_ = ch.Close()
			return nil, err
		}
	}
	if err := ch.Qos(10, 0, false); err != nil {
		_ = ch.Close()
		return nil, err
	}

	deliveries, err := ch.Consume(queue.Name, "", false, false, false, false, nil)
	if err != nil {
		_ = ch.Close()
		return nil, err
	}

	done := make(chan struct{})
	go func() {
		defer close(done)
		defer func() { _ = ch.Close() }()
		for {
			select {
			case <-ctx.Done():
				log.Info("rabbitmq consumer stopped", zap.String("queue", c.queue))
				return
			case delivery, ok := <-deliveries:
				if !ok {
					log.Warn("rabbitmq consumer delivery channel closed", zap.String("queue", c.queue))
					return
				}
				if err := handler(ctx, delivery.RoutingKey, delivery.Body); err != nil {
					requeue := !errors.Is(ctx.Err(), context.Canceled)
					_ = delivery.Nack(false, requeue)
					log.Error("rabbitmq event handling failed", zap.String("topic", delivery.RoutingKey), zap.Error(err))
					continue
				}
				_ = delivery.Ack(false)
			}
		}
	}()

	return done, nil
}
