package main

import (
	"context"
	"net/http"
	"sync"

	"github.com/gorilla/mux"

	"github.com/amazingchow/tiny-but-powerful-websocket-gateway/internal/common/config"
	"github.com/amazingchow/tiny-but-powerful-websocket-gateway/internal/common/logger"
	"github.com/amazingchow/tiny-but-powerful-websocket-gateway/internal/service"
)

func SetupWebsocketGatewayService(ctx context.Context, wg *sync.WaitGroup, stop chan struct{}) {
	defer wg.Done()

	router := mux.NewRouter()
	router.HandleFunc("/", service.ServeWebsocketConnection)
	server := &http.Server{
		Addr:    config.GetConfig().ServiceWsEndpoint,
		Handler: router,
	}
	go func() {
		logger.GetGlobalLogger().Infof("WebsocketGatewayService is running @\x1b[1;31mws://%s/go\x1b[0m.",
			config.GetConfig().ServiceWsEndpoint)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.GetGlobalLogger().WithError(err).Error("Failed to serve WebsocketGatewayService.")
		}
	}()

	<-stop
	server.Shutdown(ctx)
	logger.GetGlobalLogger().Warning("Stopped WebsocketGatewayService.")
}
