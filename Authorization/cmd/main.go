package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"sync"

	hand "github.com/Wladim1r/auth/internal/api/handlers"
	repo "github.com/Wladim1r/auth/internal/api/repository"
	serv "github.com/Wladim1r/auth/internal/api/service"
	"github.com/Wladim1r/auth/lib/getenv"
	"github.com/Wladim1r/auth/lib/midware"
	"github.com/Wladim1r/auth/periferia/db"
	"github.com/Wladim1r/auth/periferia/reddis"

	"github.com/gin-gonic/gin"
)

func main() {
	db := db.MustLoad()
	rdb := reddis.NewClient()

	ctx, cancel := context.WithCancel(context.Background())

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	wg := new(sync.WaitGroup)

	repo := repo.NewRepository(db)
	if err := repo.CreateTable(); err != nil {
		slog.Error("Could not create table", "error", err)
		os.Exit(1)
	}

	serv := serv.NewService(repo)
	hand := hand.NewHandler(ctx, serv, rdb)

	r := gin.Default()

	r.POST("/register", hand.Registration)
	r.POST("/login", midware.CheckUserExists(repo), hand.Login)

	logined := r.Group("/auth")
	logined.Use(midware.CheckCookie(ctx, rdb))
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
