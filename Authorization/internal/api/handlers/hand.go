// Package handlers
package handlers

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"net/http"

	repo "github.com/Wladim1r/auth/internal/api/repository"
	"github.com/Wladim1r/auth/internal/models"
	"github.com/Wladim1r/auth/lib/hashpwd"
	"github.com/Wladim1r/auth/periferia/reddis"
	"github.com/gin-gonic/gin"
)

type handler struct {
	repo repo.UsersDB
}

func NewHandler(repo repo.UsersDB) *handler {
	return &handler{
		repo: repo,
	}
}

func (h *handler) Registration(c *gin.Context) {
	var req models.Request
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	row := h.repo.SelectIDByName(req.Name)

	var id int
	err := row.Scan(&id)

	if err != nil {
		if err == sql.ErrNoRows {
			hashPwd, err := hashpwd.HashPwd(h.repo, []byte(req.Password), req.Name)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"Could not hash password": err.Error(),
				})
				return
			}

			err = h.repo.CreateUser(req.Name, hashPwd)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"Could not create user": err.Error(),
				})
				return
			}

			c.JSON(http.StatusCreated, gin.H{
				"message": "user successful created 🎊🤩",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "db error: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusConflict, gin.H{
		"message": "user already exsited 💩",
	})
}

func (h *handler) Login(c *gin.Context) {
	name, ok := getFromCtx(c, "username")
	if !ok {
		return
	}

	key := make([]byte, 32)
	rand.Read(key)
	token := hex.EncodeToString(key)

	reddis.TokensDB[token] = name

	c.SetCookie("token", token, 80, "/", "localhost", false, true)
	c.JSON(http.StatusOK, "Login success!🫦")
}

func (h *handler) Test(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"message": "molodec! 👍",
	})
}

func (h *handler) Logout(c *gin.Context) {
	token, ok := getFromCtx(c, "token")
	if !ok {
		return
	}

	delete(reddis.TokensDB, token)

	c.SetCookie("token", "", -1, "/", "localhost", false, true)
	c.JSON(http.StatusOK, gin.H{
		"message": "you've got rid of 🍪🗑️",
	})
}

func (h *handler) Delacc(c *gin.Context) {
	name, ok := getFromCtx(c, "username")
	if !ok {
		return
	}

	token, ok := getFromCtx(c, "token")
	if !ok {
		return
	}

	err := h.repo.DeleteUser(name)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "❌🗑️ Could not delete user: " + err.Error(),
		})
		return
	}

	delete(reddis.TokensDB, token)
	c.SetCookie("token", "", -1, "/", "localhost", false, true)

	c.JSON(http.StatusOK, gin.H{
		"message": "👍 user has successful deleted from DB",
	})
}

func getFromCtx(c *gin.Context, key string) (string, bool) {
	username, exists := c.Get(key)
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{
			"err": fmt.Sprintf("context var %s does not exist", key),
		})
		return "", false
	}
	name := username.(string)

	return name, true
}
