package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"time"

	"github.com/Wladim1r/profile/connmanager"
	hand "github.com/Wladim1r/profile/internal/api/auth/handlers"
	handler "github.com/Wladim1r/profile/internal/api/profile/handlers"
	"github.com/Wladim1r/profile/internal/api/profile/repository"
	"github.com/Wladim1r/profile/internal/api/profile/service"
	"github.com/Wladim1r/profile/lib/getenv"
	"github.com/Wladim1r/profile/lib/midware"
	"github.com/Wladim1r/profile/periferia/db"
	"github.com/Wladim1r/proto-crypto/gen/protos/auth-portfile"
	"github.com/gin-gonic/gin"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	db := db.MustLoad()
	uRepo, cRepo := repository.NewProfileRepository(db)
	uRepo.CreateTables()

	wg := new(sync.WaitGroup)
	ctx, cancel := context.WithCancel(context.Background())

	uServ, cServ := service.NewProfileService(uRepo, cRepo)
	connManager := connmanager.NewConnManager(ctx, wg, &http.Client{})

	handServ := handler.NewHandler(uServ, cServ, &connManager)

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	conn, err := grpc.NewClient(
		getenv.GetString("GRPC_ADDR", "localhost:50051"),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		panic(err)
	}

	authConn := auth.NewAuthClient(conn)

	handAuth := hand.NewClient(authConn, uServ)

	r := gin.Default()

	v1 := r.Group("/v1")
	{
		v1.POST("/register", handAuth.Registration)
		v1.POST("/login", handAuth.Login)

		v1.POST("/refresh", midware.CheckAuth(true), handAuth.Refresh)

		auth := v1.Group("/auth")
		auth.Use(midware.CheckAuth(false))
		{
			auth.POST("/test", handAuth.Test)
			auth.POST("/logout", handAuth.Logout)
		}
	}

	v2 := r.Group("/v2")
	v2.Use(midware.CheckAuth(false))
	{
		coins := v2.Group("/coin")
		{
			coins.GET("/symbols", handServ.GetCoins)
			coins.POST("/symbol", handServ.AddCoin)
			coins.PATCH("/symbol", handServ.UpdateCoin)
			coins.DELETE("/symbol", handServ.DeleteCoin)
		}

		user := v2.Group("/user")
		{
			// user.GET("/profile", handServ.GetUserProfile)
			user.DELETE("/profile", handServ.DeleteUserProfile)
			user.GET("/profile/ws", handServ.GetUserProfileWS)
		}
	}

	server := http.Server{
		Addr:    getenv.GetString("SERVER_ADDR", ":8080"),
		Handler: r,
	}
	go func() {
		if err := server.ListenAndServe(); err != nil {
			slog.Error("Failed to run server", "error", err)
			c <- os.Interrupt
		}
	}()

	wg.Add(1)
	go connManager.Run()

	<-c

	cancel()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		slog.Error("Failed to gracefully shutdown HTTP server", "error", err)
	} else {
		slog.Info("✅ HTTP server stopped gracefully")
	}

	wg.Wait()
}
