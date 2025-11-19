// Package midware
package midware

import (
	"database/sql"
	"net/http"

	repo "github.com/Wladim1r/auth/internal/api/repository"
	"github.com/Wladim1r/auth/internal/models"
	"github.com/Wladim1r/auth/periferia/reddis"
	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

func CheckCookie() gin.HandlerFunc {
	return func(c *gin.Context) {
		token, err := c.Cookie("token")
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"no cookie": err.Error(),
			})
			c.Abort()
			return
		}

		name, ok := reddis.TokensDB[token]
		if !ok {
			c.JSON(http.StatusConflict, gin.H{
				"error": "invalid cookie",
			})
			c.Abort()
			return
		}

		c.Set("token", token)
		c.Set("username", name)
		c.Next()
	}
}

func CheckUserExists(db repo.UsersDB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req models.Request
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": err.Error(),
			})
			c.Abort()
			return
		}

		password, err := db.SelectPwdByName(req.Name)
		if err != nil {
			if err == sql.ErrNoRows {
				c.JSON(http.StatusUnauthorized, gin.H{
					"message": "user does not registered ❌",
				})
				c.Abort()
				return
			}

			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "db error: " + err.Error(),
			})
			c.Abort()
			return
		}

		if err := bcrypt.CompareHashAndPassword([]byte(password), []byte(req.Password)); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"err": "passwords not equal 🚫🟰",
			})
			c.Abort()
		}

		c.Set("username", req.Name)
		c.Next()
	}
}
