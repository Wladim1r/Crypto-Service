package main

import (
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

	"github.com/gin-gonic/gin"
)

func main() {
	db := db.MustLoad()

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	wg := new(sync.WaitGroup)

	uRepo, tRepo := repo.NewRepositories(db)
	uRepo.CreateTable()

	uServ, tServ := serv.NewServices(uRepo, tRepo)
	hand := hand.NewHandler(uServ, tServ)

	r := gin.Default()

	r.POST("/register", hand.Registration)
	r.POST("/login", midware.CheckCookieUserID(), midware.CheckUserExists(uRepo), hand.Login)

	test := r.Group("/auth")
	test.Use(midware.CheckAuth(uRepo))
	{
		test.POST("/test", hand.Test)
	}

	refresh := r.Group("/auth")
	refresh.Use(midware.CheckCookieRefToken(), midware.CheckCookieUserID())
	{
		refresh.POST("/refresh", hand.Refresh)
	}

	logined := r.Group("/auth")
	logined.Use(
		midware.CheckAuth(uRepo),
		midware.CheckCookieRefToken(),
		midware.CheckCookieUserID(),
	)
	{
		logined.POST("/logout", hand.Logout)
		logined.POST("/delacc", hand.Delacc)
	}

	server := http.Server{
		Addr:    getenv.GetString("SERVER_ADDR", ":8080"),
		Handler: r,
	}

	wg.Add(1)
	go server.ListenAndServe()

	<-c
	wg.Wait()
}
