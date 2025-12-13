package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/Wladim1r/profile/connmanager"
	"github.com/Wladim1r/profile/internal/api/profile/service"
	"github.com/Wladim1r/profile/internal/models"
	"github.com/Wladim1r/profile/lib/errs"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

type handler struct {
	us service.UsersService
	cs service.CoinsService
	cm *connmanager.ConnectionManager
	cl *http.Client
}

func NewHandler(
	us service.UsersService,
	cs service.CoinsService,
	cm *connmanager.ConnectionManager,
) *handler {
	return &handler{us: us, cs: cs, cm: cm, cl: &http.Client{}}
}

func (h *handler) GetCoins(c *gin.Context) {
	userIDany, ok := c.Get("user_id")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "no cookie",
		})
		return
	}

	userID := userIDany.(float64)

	coins, err := h.cs.GetCoins(userID)
	if err != nil {
		switch {
		case errors.Is(err, errs.ErrRecordingWNF):
			c.JSON(http.StatusNotFound, gin.H{
				"error": "coins not found",
			})
		case errors.Is(err, errs.ErrDB):
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": err.Error(),
			})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "unknown error: " + err.Error(),
			})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"there are your coins": coins,
	})
}

func (h *handler) AddCoin(c *gin.Context) {
	var req models.CoinRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid body request",
		})
		return
	}

	userIDany, ok := c.Get("user_id")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "no cookie",
		})
		return
	}

	userID := userIDany.(float64)

	symbol := strings.ToLower(req.Symbol)

	if err := h.cs.AddCoin(userID, symbol, req.Quantity); err != nil {
		switch {
		case errors.Is(err, errs.ErrDB):
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": err.Error(),
			})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "unknown error: " + err.Error(),
			})
		}
		return
	}

	h.cm.FollowCoin(int(userID), symbol)

	urlQeury := fmt.Sprintf(
		"http://aggregator-service:8088/coin?symbol=%s&id=%d",
		url.QueryEscape(symbol),
		int(userID),
	)

	reqHTTP, err := http.NewRequestWithContext(c.Request.Context(), http.MethodGet, urlQeury, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	resp, err := h.cl.Do(reqHTTP)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to send http request: " + err.Error(),
		})
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to read body response: " + err.Error(),
		})
		return
	}

	var respBody map[string]interface{}
	json.Unmarshal(body, &respBody)

	c.JSON(http.StatusCreated, gin.H{
		"message":            "coin has successfully added",
		"aggregator message": respBody,
	})
}

func (h *handler) UpdateCoin(c *gin.Context) {
	var req models.CoinRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid body request",
		})
		return
	}

	userIDany, ok := c.Get("user_id")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "no cookie",
		})
		return
	}

	userID := userIDany.(float64)

	if err := h.cs.UpdateCoin(userID, req.Symbol, req.Quantity); err != nil {
		switch {
		case errors.Is(err, errs.ErrDB):
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": err.Error(),
			})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "unknown error: " + err.Error(),
			})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "choosed coin updated V",
	})
}

func (h *handler) DeleteCoin(c *gin.Context) {
	var req models.CoinRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid body request",
		})
		return
	}

	userIDany, ok := c.Get("user_id")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "no cookie",
		})
		return
	}

	userID := userIDany.(float64)

	symbol := strings.ToLower(req.Symbol)

	if err := h.cs.DeleteCoin(userID, symbol); err != nil {
		switch {
		case errors.Is(err, errs.ErrDB):
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": err.Error(),
			})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "unknown error: " + err.Error(),
			})
		}
		return
	}

	urlQeury := fmt.Sprintf(
		"http://aggregator-service:8088/coin?symbol=%s&id=%d",
		url.QueryEscape(symbol),
		int(userID),
	)

	reqHTTP, err := http.NewRequestWithContext(
		c.Request.Context(),
		http.MethodDelete,
		urlQeury,
		nil,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	resp, err := h.cl.Do(reqHTTP)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to send http request: " + err.Error(),
		})
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to read body response: " + err.Error(),
		})
		return
	}

	var respBody map[string]interface{}
	json.Unmarshal(body, &respBody)

	c.JSON(http.StatusOK, gin.H{
		"message": "choosed coin updated V",
	})
}

// ------------------------------------------------------------------

func (h *handler) DeleteUserProfile(c *gin.Context) {
	userIDany, ok := c.Get("user_id")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "no cookie",
		})
		return
	}

	userID := userIDany.(float64)

	if err := h.us.DeleteUserProfileByUserID(userID); err != nil {
		switch {
		case errors.Is(err, errs.ErrDB):
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": err.Error(),
			})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "unknown error: " + err.Error(),
			})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "user has deleted",
	})
}

func (h *handler) GetUserProfileWS(c *gin.Context) {
	userIDany, ok := c.Get("user_id")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "no cookie",
		})
		return
	}

	userID := userIDany.(float64)

	user, err := h.us.GetUserProfileByUserID(userID)
	if err != nil {
		switch {
		case errors.Is(err, errs.ErrRecordingWNF):
			c.JSON(http.StatusNotFound, gin.H{
				"error": err.Error(),
			})
		case errors.Is(err, errs.ErrDB):
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": err.Error(),
			})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "unknown error: " + err.Error(),
			})
		}
		return
	}

	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "could not upgrade http to websocket connection 😢",
		})
		return
	}

	// подписываем на все монеты пользователя
	for _, coin := range user.Coins {
		h.cm.FollowCoin(int(userID), strings.ToLower(coin.Symbol))
	}

	h.cm.Register(int(userID), user, conn)
}
