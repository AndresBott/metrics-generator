package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/francescomari/httprun"
	"github.com/francescomari/metrics-generator/internal/api"
	"github.com/francescomari/metrics-generator/internal/limits"
	"github.com/francescomari/metrics-generator/internal/metrics"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"golang.org/x/sync/errgroup"
)

var requestDuration = promauto.NewHistogram(prometheus.HistogramOpts{
	Name: "metrics_generator_request_duration_seconds",
	Help: "Request duration in seconds",
})

var requestErrorsCount = promauto.NewCounter(prometheus.CounterOpts{
	Name: "metrics_generator_request_errors_count",
	Help: "Number of errors observed in requests",
})

func main() {
	if err := run(); err != nil {
		log.Fatalf("error: %v", err)
	}
}

func run() error {
	rand.Seed(time.Now().Unix())

	var g metricsGenerator

	flag.StringVar(&g.address, "addr", ":8080", "The address to listen to")
	flag.IntVar(&g.minDuration, "duration-min", 1, "Minimum request duration")
	flag.IntVar(&g.maxDuration, "duration-max", 10, "Maximum request duration")
	flag.IntVar(&g.reqHour, "requests-hour", 1000, "Metric generation rate")
	flag.Float64Var(&g.errorsPercentage, "errors-percentage", 10, "Which percentage of the requests will fail")
	flag.Parse()

	return g.run()
}

type metricsGenerator struct {
	address          string
	minDuration      int
	maxDuration      int
	reqHour          int
	errorsPercentage float64
}

func (g *metricsGenerator) run() error {
	config, err := g.buildLimitsConfig()
	if err != nil {
		return err
	}

	ctx, cancel := g.setupSignalHandler()
	defer cancel()

	if err := g.runServices(ctx, config); err != nil {
		return fmt.Errorf("run services: %v", err)
	}

	return nil
}

func (g *metricsGenerator) buildLimitsConfig() (*limits.Config, error) {
	var config limits.Config

	if err := config.SetDurationInterval(g.minDuration, g.maxDuration); err != nil {
		return nil, fmt.Errorf("set max duration: %v", err)
	}

	if err := config.SetErrorsPercentage(g.errorsPercentage); err != nil {
		return nil, fmt.Errorf("set errors percentage: %v", err)
	}

	if err := config.SetRequestsHour(g.reqHour); err != nil {
		return nil, fmt.Errorf("set request hour: %v", err)
	}

	return &config, nil
}

func (g *metricsGenerator) setupSignalHandler() (context.Context, context.CancelFunc) {
	return signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
}

func (g *metricsGenerator) runServices(ctx context.Context, config *limits.Config) error {
	group, ctx := errgroup.WithContext(ctx)

	group.Go(func() error {
		return g.runMetricsGenerator(ctx, config)
	})

	group.Go(func() error {
		return g.runAPIServer(ctx, config)
	})

	return group.Wait()
}

func (g *metricsGenerator) runMetricsGenerator(ctx context.Context, config *limits.Config) error {
	generator := metrics.Generator{
		Config:   config,
		Duration: requestDuration,
		Errors:   requestErrorsCount,
	}

	if err := g.handleMetricsGeneratorError(generator.Run(ctx)); err != nil {
		return fmt.Errorf("metrics generator: %v", err)
	}

	return nil
}

func (g *metricsGenerator) runAPIServer(ctx context.Context, config *limits.Config) error {
	handler := api.Handler{
		Config:  config,
		Metrics: promhttp.Handler(),
	}

	server := http.Server{
		Addr:    g.address,
		Handler: &handler,
	}

	runServer := httprun.Server{
		HTTPServer:      &server,
		ShutdownTimeout: time.Second,
	}

	if err := runServer.ListenAndServe(ctx); err != nil {
		return fmt.Errorf("API server: %v", err)
	}

	return nil
}

func (g *metricsGenerator) handleMetricsGeneratorError(err error) error {
	switch err {
	case context.Canceled:
		return nil
	default:
		return err
	}
}
