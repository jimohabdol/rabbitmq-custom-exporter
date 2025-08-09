package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"rabbitmq-exporter/metrics"
	"rabbitmq-exporter/rabbitmq"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type Config struct {
	RabbitMQURL      string        `mapstructure:"rabbitmq_url"`
	RabbitMQUsername string        `mapstructure:"rabbitmq_username"`
	RabbitMQPassword string        `mapstructure:"rabbitmq_password"`
	ScrapeInterval   time.Duration `mapstructure:"scrape_interval"`
	ListenPort       int           `mapstructure:"listen_port"`
	Timeout          time.Duration `mapstructure:"timeout"`
}

const (
	DefaultRabbitMQURL      = "http://localhost:15672"
	DefaultRabbitMQUsername = "guest"
	DefaultRabbitMQPassword = "guest"
	DefaultScrapeInterval   = 15 * time.Second
	DefaultListenPort       = 9419
	DefaultTimeout          = 10 * time.Second
)

var (
	config  Config
	rootCmd = &cobra.Command{
		Use:   "rabbitmq-exporter",
		Short: "A custom RabbitMQ Prometheus exporter",
		Long: `A lightweight, efficient RabbitMQ Prometheus exporter that provides 
granular queue-level visibility beyond standard metrics.`,
		RunE: run,
	}
)

func init() {
	rootCmd.Flags().String("config", "", "Path to config file (default: config.yaml)")
	rootCmd.Flags().String("rabbitmq-url", DefaultRabbitMQURL, "RabbitMQ Management API URL")
	rootCmd.Flags().String("username", DefaultRabbitMQUsername, "RabbitMQ username")
	rootCmd.Flags().String("password", DefaultRabbitMQPassword, "RabbitMQ password")
	rootCmd.Flags().Duration("scrape-interval", DefaultScrapeInterval, "Scrape interval")
	rootCmd.Flags().Int("port", DefaultListenPort, "Listen port")
	rootCmd.Flags().Duration("timeout", DefaultTimeout, "Request timeout")

	viper.BindPFlag("rabbitmq_url", rootCmd.Flags().Lookup("rabbitmq-url"))
	viper.BindPFlag("rabbitmq_username", rootCmd.Flags().Lookup("username"))
	viper.BindPFlag("rabbitmq_password", rootCmd.Flags().Lookup("password"))
	viper.BindPFlag("scrape_interval", rootCmd.Flags().Lookup("scrape-interval"))
	viper.BindPFlag("listen_port", rootCmd.Flags().Lookup("port"))
	viper.BindPFlag("timeout", rootCmd.Flags().Lookup("timeout"))

	viper.SetEnvPrefix("RABBITMQ_EXPORTER")
	viper.AutomaticEnv()

	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("/etc/rabbitmq-exporter/")
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run(cmd *cobra.Command, args []string) error {
	if configFile, _ := cmd.Flags().GetString("config"); configFile != "" {
		viper.SetConfigFile(configFile)
		if err := viper.ReadInConfig(); err != nil {
			return fmt.Errorf("failed to read config file %s: %w", configFile, err)
		}
		log.Printf("Using config file: %s", configFile)
	} else {
		if err := viper.ReadInConfig(); err != nil {
			if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
				return fmt.Errorf("failed to read config file: %w", err)
			}
			log.Printf("No config file found, using defaults and command line flags")
		} else {
			log.Printf("Using config file: %s", viper.ConfigFileUsed())
		}
	}

	if err := viper.Unmarshal(&config); err != nil {
		return fmt.Errorf("failed to unmarshal config: %w", err)
	}

	if config.RabbitMQURL == "" {
		config.RabbitMQURL = DefaultRabbitMQURL
	}
	if config.RabbitMQUsername == "" {
		config.RabbitMQUsername = DefaultRabbitMQUsername
	}
	if config.RabbitMQPassword == "" {
		config.RabbitMQPassword = DefaultRabbitMQPassword
	}
	if config.ScrapeInterval == 0 {
		config.ScrapeInterval = DefaultScrapeInterval
	}
	if config.ListenPort == 0 {
		config.ListenPort = DefaultListenPort
	}
	if config.Timeout == 0 {
		config.Timeout = DefaultTimeout
	}

	log.Printf("Starting RabbitMQ Exporter")
	log.Printf("Configuration:")
	log.Printf("  RabbitMQ URL: %s", config.RabbitMQURL)
	log.Printf("  Username: %s", config.RabbitMQUsername)
	log.Printf("  Scrape Interval: %v", config.ScrapeInterval)
	log.Printf("  Listen Port: %d", config.ListenPort)
	log.Printf("  Timeout: %v", config.Timeout)

	client := rabbitmq.NewClient(config.RabbitMQURL, config.RabbitMQUsername, config.RabbitMQPassword, config.Timeout)
	defer client.Close()

	if err := client.HealthCheck(context.Background()); err != nil {
		return fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}
	log.Printf("Successfully connected to RabbitMQ")

	metrics := metrics.NewMetrics()

	collector := NewCollector(client, metrics, config.ScrapeInterval)
	defer collector.Stop()

	prometheus.MustRegister(collector)

	mux := http.NewServeMux()

	mux.Handle("/metrics", promhttp.Handler())

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		if err := client.HealthCheck(r.Context()); err != nil {
			http.Error(w, "Health check failed", http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`
<!DOCTYPE html>
<html>
<head>
    <title>RabbitMQ Custom Exporter</title>
</head>
<body>
    <h1>RabbitMQ Custom Prometheus Exporter</h1>
    <p>This exporter provides detailed queue-level metrics for RabbitMQ.</p>
    <ul>
        <li><a href="/metrics">Metrics</a> - Prometheus metrics endpoint</li>
        <li><a href="/health">Health</a> - Health check endpoint</li>
    </ul>
</body>
</html>
`))
	})

	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", config.ListenPort),
		Handler: mux,
	}

	go func() {
		log.Printf("Starting HTTP server on port %d", config.ListenPort)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTP server error: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Printf("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Printf("Server shutdown error: %v", err)
	}

	log.Printf("Server stopped")
	return nil
}
