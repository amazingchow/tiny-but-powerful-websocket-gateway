package main

import (
	"context"
	"flag"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"

	"github.com/amazingchow/tiny-but-powerful-websocket-gateway/internal/common/config"
	"github.com/amazingchow/tiny-but-powerful-websocket-gateway/internal/common/logger"
	"github.com/amazingchow/tiny-but-powerful-websocket-gateway/internal/metrics"
)

var (
	_ConfigFile = flag.String("conf", "./etc/infra-websocket-gateway-service-dev.json", "config file path")
)

func main() {
	// Seed random number generator, is deprecated since Go 1.20.
	// rand.Seed(time.Now().UnixNano())
	logrus.SetLevel(logrus.DebugLevel)

	// Parse command-line flags.
	flag.Parse()
	// Load configuration file.
	config.LoadConfigFileOrPanic(*_ConfigFile)
	// Run service-initialization and service-cleanup.
	defer SetupTeardown()()

	// Run service-main.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	stopCh := make(chan struct{})
	wg := &sync.WaitGroup{}
	defer wg.Wait()
	wg.Add(1)
	go SetupWebsocketGatewayService(ctx, wg, stopCh)

	// Wait for signal to stop the service.
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)
	<-sigCh

	// Stop the service.
	close(stopCh)
}

func SetupTeardown() func() {
	logrus.Debug("Run service-initialization.")
	SetupRuntimeEnvironment(config.GetConfig())
	return func() {
		logrus.Debug("Run service-cleanup.")
		ClearRuntimeEnvironment(config.GetConfig())
	}
}

func SetupRuntimeEnvironment(conf *config.Config) {
	logger.SetGlobalLogger(conf)
	if len(conf.ServiceMetricsEndpoint) > 0 {
		go func() {
			metrics.Register()
			http.Handle("/metrics", promhttp.Handler())
			logger.GetGlobalLogger().Error(http.ListenAndServe(conf.ServiceMetricsEndpoint, nil))
		}()
	}
	// Add more service initialization here.
}

func ClearRuntimeEnvironment(_ *config.Config) {
}
