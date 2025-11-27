// Package handlers
package handlers

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/Wladim1r/auth/internal/api/service"
	"github.com/Wladim1r/auth/internal/models"
	"github.com/Wladim1r/auth/lib/errs"
	"github.com/Wladim1r/auth/lib/getenv"
	"github.com/Wladim1r/auth/lib/hashpwd"
	"github.com/gin-gonic/gin"
)

type handler struct {
	us service.UserService
	ts service.TokenService
}

func NewHandler(uServ service.UserService, tServ service.TokenService) *handler {
	return &handler{
		us: uServ,
		ts: tServ,
	}
}

func (h *handler) Registration(c *gin.Context) {
	var req models.UserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	err := h.us.CheckUserExistsByName(req.Name)

	if err != nil {
		switch {
		case errors.Is(err, errs.ErrRecordingWNF):
			hashPwd, err := hashpwd.HashPwd([]byte(req.Password))
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"Could not hash password": err.Error(),
				})
				return
			}

			userID, err := h.us.CreateUser(req.Name, hashPwd)
			if err != nil {
				switch {
				case errors.Is(err, errs.ErrRecordingWNC):
					c.JSON(http.StatusInternalServerError, gin.H{
						"Could not create user rawsAffected=0": err.Error(),
					})
					return

				default:
					c.JSON(http.StatusInternalServerError, gin.H{
						"Could not create user": err.Error(),
					})
					return
				}
			}

			userIDstr := strconv.Itoa(userID)
			refreshTTL := getenv.GetTime("REFRESH_TTL", 150*time.Second)

			c.SetCookie(
				"userID",
				userIDstr,
				int(refreshTTL),
				"/",
				getenv.GetString("COOKIE_DOMAIN", "localhost"),
				false,
				true,
			)

			c.JSON(http.StatusCreated, gin.H{
				"message": "user successful created 🎊🤩",
			})
			return

		default:
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "db error: " + err.Error(),
			})
			return
		}
	}

	c.JSON(http.StatusConflict, gin.H{
		"message": "user already exsited 💩",
	})
}

func (h *handler) Login(c *gin.Context) {
	userID, ok := getFromCtx[int](c, "user_id")
	if !ok {
		return
	}

	refreshTTL := getenv.GetTime("REFRESH_TTL", 150*time.Second)

	access, refresh, err := h.ts.SaveToken(userID, time.Now().Add(refreshTTL))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.SetCookie(
		"refreshToken",
		refresh,
		int(refreshTTL),
		"/",
		getenv.GetString("COOKIE_DOMAIN", "localhost"),
		false,
		true,
	)

	tStruct := struct {
		Access  string `json:"access"`
		Refresh string `json:"refresh"`
	}{
		Access:  access,
		Refresh: refresh,
	}

	c.JSON(http.StatusOK, gin.H{
		"msg":                  "Login success!🫦",
		"Here is your tokens🌐": tStruct,
	})
}

func (h *handler) Test(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"message": "molodec! 👍",
	})
}

func (h *handler) Logout(c *gin.Context) {
	token, ok := getFromCtx[string](c, "refresh_token")
	if !ok {
		return
	}

	if err := h.ts.DeleteToken(token); err != nil {
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
				"unknown error": err.Error(),
			})
		}
		return
	}

	cookieDomain := getenv.GetString("COOKIE_DOMAIN", "localhost")
	c.SetCookie("refreshToken", "", -1, "/", cookieDomain, false, true)
	c.SetCookie("userID", "", -1, "/", cookieDomain, false, true)

	c.JSON(http.StatusOK, gin.H{
		"msg": "you've successfully logouted",
	})

}

func (h *handler) Delacc(c *gin.Context) {
	userID, ok := getFromCtx[int](c, "user_id")
	if !ok {
		return
	}

	err := h.us.DeleteUser(userID)
	if err != nil {
		switch {
		case errors.Is(err, errs.ErrRecordingWND):
			c.JSON(http.StatusInternalServerError, gin.H{
				"Could not delete user rawsAffected=0": err.Error(),
			})
			return

		default:
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "❌🗑️ Could not delete user: " + err.Error(),
			})
			return
		}
	}

	cookieDomain := getenv.GetString("COOKIE_DOMAIN", "localhost")
	c.SetCookie("refreshToken", "", -1, "/", cookieDomain, false, true)
	c.SetCookie("userID", "", -1, "/", cookieDomain, false, true)

	c.JSON(http.StatusOK, gin.H{
		"message": "👍 user has successful deleted from DB",
	})
}

func (h *handler) Refresh(c *gin.Context) {
	token, ok := getFromCtx[string](c, "refresh_token")
	if !ok {
		return
	}
	userID, ok := getFromCtx[int](c, "user_id")
	if !ok {
		return
	}
	refreshTTL := getenv.GetTime("REFRESH_TTL", 150*time.Second)

	access, refresh, err := h.ts.RefreshAccessToken(token, userID, time.Now().Add(refreshTTL))
	if err != nil {
		switch {
		case errors.Is(err, errs.ErrTokenTTL):
			c.JSON(http.StatusForbidden, gin.H{
				"error": err.Error(),
			})
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
				"unknown error": err.Error(),
			})
		}
		return
	}

	c.SetCookie(
		"refreshToken",
		refresh,
		int(refreshTTL),
		"/",
		getenv.GetString("COOKIE_DOMAIN", "localhost"),
		false,
		true,
	)

	c.JSON(http.StatusOK, gin.H{
		"msg":         "tokens have refreshed",
		"new access":  access,
		"new refresh": refresh,
	})
}
