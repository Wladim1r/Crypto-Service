package main

import (
	"log/slog"
	"net/http"
	"os"

	hand "github.com/Wladim1r/auth/internal/api/handlers"
	repo "github.com/Wladim1r/auth/internal/api/repository"
	"github.com/Wladim1r/auth/internal/db"
	"github.com/Wladim1r/auth/lib/midware"

	"github.com/gin-gonic/gin"
)

func main() {
	db := db.MustLoad()

	repo := repo.NewRepository(db)
	if err := repo.CreateTable(); err != nil {
		slog.Error("Could not create table", "error", err)
		os.Exit(1)
	}
	defer repo.Close()

	hand := hand.NewHandler(repo)

	r := gin.Default()

	r.POST("/register", hand.Registration)
	r.POST("/login", midware.CheckUserExists(repo), hand.Login)

	logined := r.Group("/auth")
	logined.Use(midware.CheckCookie())
	{
		logined.POST("/test", hand.Test)
		logined.POST("/logout", hand.Logout)
		logined.POST("/delacc", hand.Delacc)
	}

	server := http.Server{
		Addr:    ":8080",
		Handler: r,
	}

	server.ListenAndServe()
}
