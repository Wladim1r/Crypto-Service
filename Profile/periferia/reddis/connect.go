package reddis

import (
	"context"
	"log/slog"
	"sync"

	"github.com/redis/go-redis/v9"
)

type Message struct {
	Channel string
	Payload string
}

type Subscriber struct {
	Client   *redis.Client
	Messages chan Message

	subscriptions map[string]*redis.PubSub
	mu            sync.RWMutex
}

func NewSubscriber() *Subscriber {
	return &Subscriber{
		Client: redis.NewClient(&redis.Options{
			Addr:     "redis:6379",
			Password: "",
			DB:       0,
		}),
		Messages:      make(chan Message, 200),
		subscriptions: make(map[string]*redis.PubSub),
	}
}

func (s *Subscriber) Subscribe(ctx context.Context, symbol string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.subscriptions[symbol]; exists {
		slog.Debug("Already subscribed to channel", "channel", symbol)
		return nil
	}

	pubsub := s.Client.Subscribe(ctx, symbol)

	_, err := pubsub.Receive(ctx)
	if err != nil {
		slog.Error("Failed to subscribe to redis channel", "channel", symbol, "error", err)
		return err
	}

	s.subscriptions[symbol] = pubsub
	slog.Info("Subscribed to new redis channel", "channel", symbol)

	go s.listener(ctx, pubsub, symbol)

	return nil
}

func (s *Subscriber) Unsubscribe(ctx context.Context, symbol string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	pubsub, exists := s.subscriptions[symbol]
	if !exists {
		slog.Debug("Not subscribed to channel", "channel", symbol)
		return nil
	}

	// Отписываемся от канала
	if err := pubsub.Unsubscribe(ctx, symbol); err != nil {
		slog.Error("Failed to unsubscribe from channel", "channel", symbol, "error", err)
		return err
	}

	// Закрываем PubSub (это остановит listener горутину)
	if err := pubsub.Close(); err != nil {
		slog.Warn("Error closing pubsub", "channel", symbol, "error", err)
	}

	delete(s.subscriptions, symbol)
	slog.Info("Unsubscribed from redis channel", "channel", symbol)

	return nil
}

func (s *Subscriber) listener(ctx context.Context, pubsub *redis.PubSub, symbol string) {
	defer func() {
		slog.Debug("Listener stopped", "channel", symbol)
	}()

	ch := pubsub.Channel()

	for {
		select {
		case <-ctx.Done():
			return
		case msg, ok := <-ch:
			if !ok {
				slog.Warn("Redis pubsub channel closed", "channel", symbol)
				return
			}

			select {
			case s.Messages <- Message{Channel: msg.Channel, Payload: msg.Payload}:
			case <-ctx.Done():
				return
			default:
				slog.Warn("Messages channel full, dropping message", "channel", symbol)
			}
		}
	}
}

func (s *Subscriber) Close() {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Закрываем все активные подписки
	for symbol, pubsub := range s.subscriptions {
		if err := pubsub.Close(); err != nil {
			slog.Warn("Error closing pubsub during shutdown", "channel", symbol, "error", err)
		}
	}

	if s.Client != nil {
		s.Client.Close()
	}

	close(s.Messages)
	slog.Info("Subscriber closed, all subscriptions cleaned up")
}
