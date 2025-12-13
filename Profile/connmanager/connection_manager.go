package connmanager

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/Wladim1r/profile/internal/models"
	"github.com/Wladim1r/profile/periferia/reddis"
	"github.com/gorilla/websocket"
	"github.com/shopspring/decimal"
)

type client struct {
	Conn     *websocket.Conn
	Profile  *models.User
	Prices   map[string]decimal.Decimal
	SendChan chan []byte
}

type ConnectionManager struct {
	clients    map[int]*client
	mu         sync.RWMutex
	httpClient *http.Client

	subscriber *reddis.Subscriber

	activeRedisSub map[string]struct{}
	userSubCoins   map[string]map[int]struct{}

	mainCtx context.Context
	mainWg  *sync.WaitGroup
}

func NewConnManager(
	ctx context.Context,
	wg *sync.WaitGroup,
	httpClient *http.Client,
) ConnectionManager {
	return ConnectionManager{
		clients:        make(map[int]*client),
		subscriber:     reddis.NewSubscriber(),
		activeRedisSub: make(map[string]struct{}),
		userSubCoins:   make(map[string]map[int]struct{}),
		mainCtx:        ctx,
		mainWg:         wg,
		httpClient:     httpClient,
	}
}

func (cm *ConnectionManager) Run() {
	defer func() {
		cm.mainWg.Done()
		cm.subscriber.Close()
	}()

	slog.Info("Starting ConnectionManager Redis dispatcher")

	for {
		select {
		case <-cm.mainCtx.Done():
			return
		case msg, ok := <-cm.subscriber.Messages:
			if !ok {
				slog.Warn("ConnectionManager Redis subscriber channel closed")
				return
			}
			cm.processRedisMessage(msg)
		}
	}
}

func (cm *ConnectionManager) FollowCoin(userID int, symbol string) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if _, ok := cm.userSubCoins[symbol]; !ok {
		cm.userSubCoins[symbol] = make(map[int]struct{})
	}
	cm.userSubCoins[symbol][userID] = struct{}{}
	slog.Info("CONN_MANAGER: User followed coin", "userID", userID, "symbol", symbol)

	if _, ok := cm.activeRedisSub[symbol]; !ok {
		if err := cm.subscriber.Subscribe(cm.mainCtx, symbol); err != nil {
			slog.Error(
				"CONN_MANAGER: Could not subscribe on coin stream",
				"coin",
				symbol,
				"error",
				err.Error(),
			)
			return
		}
		cm.activeRedisSub[symbol] = struct{}{}
		slog.Info("CONN_MANAGER: New active Redis subscription created", "symbol", symbol)
	}
}

func (cm *ConnectionManager) UnfollowAllCoins(userID int) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	for symbol, users := range cm.userSubCoins {
		delete(users, userID)

		// Если больше нет подписчиков на этот символ
		if len(users) == 0 {
			slog.Info("No more subscribers, unsubscribing from Redis", "symbol", symbol)
			delete(cm.userSubCoins, symbol)
			delete(cm.activeRedisSub, symbol)
			if err := cm.subscriber.Unsubscribe(cm.mainCtx, symbol); err != nil {
				slog.Error("Failed to unsubscribe from Redis", "symbol", symbol, "error", err)
			} else {
				slog.Info("Unsubscribed from Redis channel (no subscribers)", "symbol", symbol)
			}
		}
	}
}

func (cm *ConnectionManager) processRedisMessage(msg reddis.Message) {
	cm.mu.RLock()
	// Создаем копию ID подписчиков, чтобы избежать гонки состояний
	// при итерации и одновременной отписке пользователя.
	userIDs := make([]int, 0, len(cm.userSubCoins[msg.Channel]))
	for id := range cm.userSubCoins[msg.Channel] {
		userIDs = append(userIDs, id)
	}
	cm.mu.RUnlock()

	slog.Info("Processing Redis message",
		"channel", msg.Channel,
		"subscribers_count", len(userIDs))

	var secStat models.SecondStat
	if err := json.Unmarshal([]byte(msg.Payload), &secStat); err != nil {
		slog.Error("Failed to parse JSON into 'SecondStat'", "error", err)
		return
	}

	if len(userIDs) != 0 {
		for _, id := range userIDs {
			if err := cm.WriteToUser(id, secStat); err != nil {
				slog.Warn("Failed to write to user", "user_id", id, "error", err)
			}
		}
	}
}

func (c *client) writer(ctx context.Context, wg *sync.WaitGroup) {
	defer func() {
		wg.Done()
		slog.Info("Writer goroutine stopped", "user_id", c.Profile.ID)
	}()

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
			return

		case msg, ok := <-c.SendChan:
			if !ok {
				// Канал закрыт, отправляем close message
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.Conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				slog.Error("could not write message to websocket connection",
					"user_id", c.Profile.ID,
					"error", err.Error())
				return
			}

		case <-ticker.C:
			// Отправляем ping для keep-alive
			c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				slog.Error("could not write ping message",
					"user_id", c.Profile.ID,
					"error", err.Error())
				return
			}
		}
	}
}

func (c *client) reader(cm *ConnectionManager) {
	defer func() {
		cm.unregister(int(c.Profile.ID))
	}()

	c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.Conn.SetPongHandler(
		func(string) error { c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second)); return nil },
	)

	for {
		if _, _, err := c.Conn.ReadMessage(); err != nil {
			if websocket.IsUnexpectedCloseError(
				err,
				websocket.CloseGoingAway,
				websocket.CloseAbnormalClosure,
				websocket.CloseNormalClosure,
			) {
				slog.Warn(
					"Websocket connection closed unexpectedly",
					"user_id",
					c.Profile.ID,
					"error",
					err,
				)
			} else {
				slog.Info("Websocket connection closed", "user_id", c.Profile.ID)
			}
			break
		}
	}
}

func (cm *ConnectionManager) Register(userID int, profile *models.User, conn *websocket.Conn) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	// Если пользователь уже подключен, закрываем старое соединение
	if oldClient, exists := cm.clients[userID]; exists {
		slog.Info("CONN_MANAGER: User reconnecting, closing old connection", "user_id", userID)
		// Закрываем старое соединение
		oldClient.Conn.Close()
		close(oldClient.SendChan)
	}

	sendChan := make(chan []byte, 100)
	client := &client{
		Conn:     conn,
		Profile:  profile,
		Prices:   make(map[string]decimal.Decimal),
		SendChan: sendChan,
	}

	cm.clients[userID] = client

	keys := make([]int, 0, len(cm.clients))
	for k := range cm.clients {
		keys = append(keys, k)
	}
	slog.Info("CONN_MANAGER: User registered", "userID", userID, "active_clients", keys)

	cm.mainWg.Add(1)
	go client.writer(cm.mainCtx, cm.mainWg)
	go client.reader(cm)
}
func (cm *ConnectionManager) WriteToUser(userID int, msg models.SecondStat) error {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	client, ok := cm.clients[userID]
	if !ok {
		return fmt.Errorf("Connection for user %d does not exist", userID)
	}

	client.Prices[msg.Symbol] = decimal.NewFromFloat(msg.Price)

	profile := models.Profile{
		ID:   uint(userID),
		Name: client.Profile.Name,
		Coins: models.CoinsProfile{
			Quantities: make(map[string]decimal.Decimal),
			Prices:     make(map[string]decimal.Decimal),
			Totals:     make(map[string]decimal.Decimal),
		},
	}

	for _, coin := range client.Profile.Coins {
		if price, ok := client.Prices[coin.Symbol]; ok {
			profile.Coins.Quantities[coin.Symbol] = coin.Quantity
			profile.Coins.Prices[coin.Symbol] = price
			profile.Coins.Totals[coin.Symbol] = price.Mul(coin.Quantity)
		}
	}

	profileJSON, err := json.Marshal(profile)
	if err != nil {
		return fmt.Errorf("could not parse 'profile' into JSON: %w", err)
	}

	select {
	case client.SendChan <- profileJSON:
	default:
		slog.Warn("Send channel is full, message dropped", "user_id", userID)
	}

	return nil
}

func (cm *ConnectionManager) unregister(userID int) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	client, ok := cm.clients[userID]
	if !ok {
		return // already unregistered
	}

	slog.Info("CONN_MANAGER: Unregistering user", "userID", userID)

	// Отписываемся от всех монет пользователя
	for _, coin := range client.Profile.Coins {
		symbol := coin.Symbol
		if users, ok := cm.userSubCoins[symbol]; ok {
			delete(users, userID)

			// Notify aggregator that a user has unfollowed
			urlQuery := fmt.Sprintf(
				"http://aggregator-service:8088/coin?symbol=%s&id=%d",
				url.QueryEscape(symbol),
				userID,
			)
			req, err := http.NewRequestWithContext(cm.mainCtx, http.MethodDelete, urlQuery, nil)
			if err == nil {
				go cm.httpClient.Do(req)
			}

			remainingUsers := make([]int, 0, len(users))
			for k := range users {
				remainingUsers = append(remainingUsers, k)
			}
			slog.Info(
				"CONN_MANAGER: User removed from coin subscription",
				"userID",
				userID,
				"symbol",
				symbol,
				"remaining_users",
				remainingUsers,
			)

			// Если больше нет подписчиков
			if len(users) == 0 {
				delete(cm.userSubCoins, symbol)
				delete(cm.activeRedisSub, symbol)

				if err := cm.subscriber.Unsubscribe(cm.mainCtx, symbol); err != nil {
					slog.Error(
						"CONN_MANAGER: Failed to unsubscribe from Redis",
						"symbol",
						symbol,
						"error",
						err,
					)
				} else {
					slog.Info("CONN_MANAGER: Unsubscribed from Redis (user disconnected)", "symbol", symbol)
				}
			}
		}
	}

	delete(cm.clients, userID)

	if client.Conn != nil {
		client.Conn.Close()
	}

	// Safely close the channel
	close(client.SendChan)

	keys := make([]int, 0, len(cm.clients))
	for k := range cm.clients {
		keys = append(keys, k)
	}
	slog.Info("CONN_MANAGER: User unregistered", "userID", userID, "active_clients", keys)
}
