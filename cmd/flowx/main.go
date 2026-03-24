package main

import (
	// Go Internal Packages
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	// Local Packages
	config "flowx/config"
	"flowx/flow"
	http "flowx/http"
	handlers "flowx/http/handlers"
	mongodb "flowx/repositories/mongodb"
	health "flowx/services/health"
	"flowx/services/executor"
	runsvc "flowx/services/run"
	"flowx/utils/slack"
	helpers "flowx/utils/helpers"

	// External Packages
	"github.com/alecthomas/kingpin/v2"
	_ "github.com/jsternberg/zap-logfmt"
	"github.com/knadh/koanf"
	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/providers/rawbytes"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// InitializeServer sets up the HTTP server with all dependencies wired together:
// MongoDB → Repositories → Services → Handlers → Server
func InitializeServer(ctx context.Context, k config.Config, logger *zap.Logger) (*http.Server, error) {
	mongoClient, err := mongodb.Connect(ctx, k.Mongo.URI)
	if err != nil {
		return nil, err
	}

	// Initialize Slack Alerter
	slack := slack.NewSender(k.Slack, k.IsProdMode)

	// Repositories
	runRepo := mongodb.NewRunRepository(mongoClient)
	stepRunRepo := mongodb.NewStepRunRepository(mongoClient)

	// Services
	healthSvc := health.NewService(logger, mongoClient)
	exec := executor.NewExecutor(logger, stepRunRepo, flow.Dummy)
	runSvc := runsvc.NewRunService(logger, k.Queue, runRepo, exec, slack)

	// Start the run service (spawns workers and re-enqueues incomplete runs)
	if err := runSvc.Start(ctx); err != nil {
		logger.Error("Failed To Start Run Service", zap.Error(err))
		return nil, err
	}

	// Handlers
	healthHandler := handlers.NewHealthCheckHandler(healthSvc)
	runHandler := handlers.NewRunHandler(runSvc)

	closeCallback := func() {
		_ = mongoClient.Disconnect(ctx)
		logger.Info("Server Stopped Successfully")
	}

	server := http.NewServer(logger, k.Prefix, healthHandler, runHandler, closeCallback)
	return server, nil
}

// LoadConfig loads the default configuration and overrides it with the config file
// specified by the --config flag.
func LoadConfig() *koanf.Koanf {
	configPath := kingpin.Flag("config", "Path To The Application Config File").
		Short('c').Default("config.yml").String()

	kingpin.Parse()

	k := koanf.New(".")
	_ = k.Load(rawbytes.Provider(config.DefaultConfig), yaml.Parser())
	if *configPath != "" {
		_ = k.Load(file.Provider(*configPath), yaml.Parser())
	}
	return k
}

// NewLogger builds a production zap logger configured with logfmt encoding
// and the application's hostname and service name as initial fields.
func NewLogger(k config.Config) *zap.Logger {
	zapCfg := zap.NewProductionConfig()
	zapCfg.Encoding = k.Logger.Encoding
	_ = zapCfg.Level.UnmarshalText([]byte(k.Logger.Level))
	zapCfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	zapCfg.OutputPaths = []string{"stdout"}

	hostname, _ := os.Hostname()
	zapCfg.InitialFields = map[string]any{
		"host":    hostname,
		"service": k.Application,
	}

	logger, _ := zapCfg.Build()
	return logger
}

// main is the entrypoint that loads config, sets up logging,
// and starts the HTTP server with graceful shutdown.
func main() {
	k := LoadConfig()

	// Unmarshal Config
	appKonf := config.Config{}
	if err := k.Unmarshal("", &appKonf); err != nil {
		log.Fatalf("Error Loading Config: %v", err)
	}

	// Validate Config
	if err := appKonf.Validate(); err != nil {
		helpers.LogValidationErrors(err)
		log.Fatalf("Invalid Configuration")
	}

	// Print Config in Dev Mode
	if !appKonf.IsProdMode {
		k.Print()
	}

	// Initialize Logger
	logger := NewLogger(appKonf)
	defer func() {
		_ = logger.Sync()
	}()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	srv, err := InitializeServer(ctx, appKonf, logger)
	if err != nil {
		logger.Fatal("Cannot Initialize Server", zap.Error(err))
	}

	if err = srv.Listen(ctx, appKonf.Listen); err != nil {
		logger.Fatal("Cannot Listen On Port", zap.Error(err))
	}
}
