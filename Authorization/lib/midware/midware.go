// Package midware
package midware

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	repo "github.com/Wladim1r/auth/internal/api/repository"
	"github.com/Wladim1r/auth/internal/models"
	"github.com/Wladim1r/auth/lib/errs"
	"github.com/Wladim1r/auth/lib/getenv"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

func verifyToken(c *gin.Context, token string, r repo.UserRepository) error {
	jwtToken, err := jwt.Parse(token, func(t *jwt.Token) (any, error) {
		return []byte(getenv.GetString("SECRET_KEY", "default_secret_key")), nil
	})
	if err != nil {
		return fmt.Errorf("failed to parse token: %w", err)
	}
	if !jwtToken.Valid {
		return fmt.Errorf("token is not valid")
	}

	claims, ok := jwtToken.Claims.(jwt.MapClaims)
	if !ok {
		return fmt.Errorf("could not parse jwtToken into MapClaims struct")
	}

	expRaw, ok := claims["exp"]
	if !ok {
		return fmt.Errorf("field 'expiration' did not found")
	}
	exp64, ok := expRaw.(float64)
	if !ok {
		return fmt.Errorf("failed to parse expRaw into int64")
	}
	exp := int64(exp64)

	if time.Now().Unix() > exp {
		return fmt.Errorf("token life time has expired")
	}

	userIDRaw, ok := claims["sub"]
	if !ok {
		return fmt.Errorf("field 'userID' did not found")
	}
	userID, ok := userIDRaw.(float64)
	if !ok {
		return fmt.Errorf("failed to parse 'userIDRaw' into float64")
	}

	if err := r.CheckUserExistsByID(uint(userID)); err != nil {
		return fmt.Errorf("user does not exist: %w", err)
	}

	c.Set("user_id", userID)
	return nil
}

func CheckAuth(r repo.UserRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeadVal := c.GetHeader("Authorization")
		if authHeadVal == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header is required"})
			c.Abort()
			return
		}

		bearerToken := strings.Split(authHeadVal, " ")
		if len(bearerToken) != 2 {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token format"})
			c.Abort()
			return
		}

		if err := verifyToken(c, bearerToken[1], r); err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": err.Error(),
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

func CheckCookieRefToken() gin.HandlerFunc {
	return func(c *gin.Context) {
		token, err := c.Cookie("refreshToken")
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"no cookie 🔒": err.Error(),
			})
			c.Abort()
			return
		}

		c.Set("refresh_token", token)
		c.Next()
	}
}

func CheckCookieUserID() gin.HandlerFunc {
	return func(c *gin.Context) {
		userIDstr, err := c.Cookie("userID")
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"no cookie 🆔": err.Error(),
			})
			c.Abort()
			return
		}

		userID, err := strconv.Atoi(userIDstr)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "could not convert str -> int",
			})
			c.Abort()
			return
		}

		c.Set("user_id", userID)
		c.Next()
	}
}

func CheckUserExists(db repo.UserRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req models.UserRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": err.Error(),
			})
			c.Abort()
			return
		}

		password, err := db.SelectPwdByName(req.Name)
		if err != nil {
			switch {
			case errors.Is(err, errs.ErrRecordingWNF):
				c.JSON(http.StatusUnauthorized, gin.H{
					"message": "user does not registered ❌",
				})
				c.Abort()
				return
			default:
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "db error: " + err.Error(),
				})
				c.Abort()
				return
			}
		}

		if err := bcrypt.CompareHashAndPassword([]byte(password), []byte(req.Password)); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"err": "passwords not equal 🚫🟰",
			})
			c.Abort()
		}

		c.Next()
	}
}
