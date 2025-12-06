package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"sync"

	hand "github.com/Wladim1r/auth/internal/api/handlers"
	"github.com/Wladim1r/auth/internal/api/repository"
	serv "github.com/Wladim1r/auth/internal/api/service"
	"github.com/Wladim1r/auth/internal/models"
	"github.com/Wladim1r/auth/lib/getenv"
	"github.com/Wladim1r/auth/lib/midware"
	"github.com/Wladim1r/auth/periferia/db"

	"github.com/gin-gonic/gin"
)

func main() {
	db := db.MustLoad()

	ctx, cancel := context.WithCancel(context.Background())

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	wg := new(sync.WaitGroup)

	if err := db.AutoMigrate(&models.User{}, &models.Session{}); err != nil {
		slog.Error("Could not migrate database tables", "errors", err)
		os.Exit(1)
	}

	userRepo := repository.NewUserRepository(db)
	tokenRepo := repository.NewTokenRepository(db)

	userService := serv.NewUserService(userRepo)
	tokenService := serv.NewTokenService(tokenRepo)
	hand := hand.NewHandler(ctx, userService, tokenService)

	r := gin.Default()

	r.POST("/register", hand.Registration)
	r.POST("/login", hand.Login)
	r.POST("/refresh", hand.RefreshToken)

	logined := r.Group("/auth")
	logined.Use(midware.CheckAuth())
	{
		logined.POST("/test", hand.Test)
		logined.POST("/logout", hand.Logout)
		logined.POST("/delacc", hand.Delacc)
	}

	server := http.Server{
		Addr:    getenv.GetString("SERVER_ADDR", ":8080"),
		Handler: r,
	}

	// wg.Add(1)
	go server.ListenAndServe()

	<-c
	cancel()
	wg.Wait()
}
