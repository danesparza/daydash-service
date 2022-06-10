package cmd

import (
	"context"
	"crypto/tls"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/danesparza/daydash-service/api"
	_ "github.com/danesparza/daydash-service/docs" // swagger docs location
	"github.com/danesparza/daydash-service/internal/telemetry"
	"github.com/gorilla/mux"
	"github.com/newrelic/go-agent/v3/integrations/nrgorilla"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	httpSwagger "github.com/swaggo/http-swagger"
	"golang.org/x/crypto/acme/autocert"
)

// serveCmd represents the serve command
var serveCmd = &cobra.Command{
	Use:   "start",
	Short: "Start the server",
	Long:  `The serve command starts hosting the service`,
	Run:   start,
}

func start(cmd *cobra.Command, args []string) {

	//	If we have a config file, report it:
	if viper.ConfigFileUsed() != "" {
		zlog.Debugw(
			"Using config file",
			"config", viper.ConfigFileUsed(),
		)
	} else {
		zlog.Debugf("No config file found")
	}

	loglevel := viper.GetString("log.level")

	//	Emit what we know:
	zlog.Infow(
		"Starting up",
		"loglevel", loglevel,
		"nrAppName", telemetry.NRAppName,
	)

	//	Create an api service object
	apiService := api.Service{
		StartTime: time.Now(),
	}

	//	Trap program exit appropriately
	ctx, cancel := context.WithCancel(context.Background())
	sigs := make(chan os.Signal, 2)
	signal.Notify(sigs, os.Interrupt, syscall.SIGTERM)
	go handleSignals(ctx, sigs, cancel)

	//	Create a router and setup our REST and UI endpoints...
	restRouter := mux.NewRouter()
	restRouter.Use(nrgorilla.Middleware(telemetry.NRApp))
	restRouter.Use(api.ApiVersionMiddleware)

	//	DATA ROUTES
	restRouter.HandleFunc("/v2/alerts", apiService.GetWeatherAlerts).Methods("POST")            // Get weather alerts data
	restRouter.HandleFunc("/v2/calendar", apiService.GetCalendar).Methods("POST")               // Get calendar data
	restRouter.HandleFunc("/v2/mapimage", apiService.GetMapImageForCoordinates).Methods("POST") // Get map data
	restRouter.HandleFunc("/v2/news", apiService.GetNewsReport).Methods("GET")                  // Get news data
	restRouter.HandleFunc("/v2/pollen", apiService.GetPollenReport).Methods("POST")             // Get pollen data
	restRouter.HandleFunc("/v2/weather", apiService.GetWeatherReport).Methods("POST")           // Get weather data
	// restRouter.HandleFunc("/v2/zipgeo", apiService.GetCalendar).Methods("POST")                 // Get zipgeo data

	//	SWAGGER ROUTES
	restRouter.PathPrefix("/v2/swagger").Handler(httpSwagger.WrapHandler)

	//	Letsencrypt configuration
	certManager := autocert.Manager{
		Prompt:     autocert.AcceptTOS,
		HostPolicy: autocert.HostWhitelist("api.daydash.net", "api-v2.daydash.net"),
		Cache:      autocert.DirCache("/etc/daydash-service/certs"),
	}

	server := &http.Server{
		Addr:    ":https",
		Handler: restRouter,
		TLSConfig: &tls.Config{
			GetCertificate: certManager.GetCertificate,
		},
	}

	if viper.GetBool("server.httponly") {
		//	HTTP server
		go func() {
			zlog.Infow("Started REST service (http only)",
				"server", ":http",
			)
			zlog.Errorw("HTTP API service error",
				"error", http.ListenAndServe(":http", restRouter),
			)
		}()
	} else {
		//	HTTP server
		go func() {
			zlog.Infow("Started REST service (redirects to TLS)",
				"server", ":http",
			)
			zlog.Errorw("HTTP API service error",
				"error", http.ListenAndServe(":http", certManager.HTTPHandler(nil)),
			)
		}()

		//	TLS server
		go func() {
			zlog.Infow("Started TLS REST service",
				"server", ":https",
			)
			zlog.Errorw("TLS API service error",
				"error", server.ListenAndServeTLS("", ""), // Cert / key provided by letsencrypt
			)
		}()
	}

	//	Wait for our signal and shutdown gracefully
	<-ctx.Done()
}

func handleSignals(ctx context.Context, sigs <-chan os.Signal, cancel context.CancelFunc) {
	select {
	case <-ctx.Done():
	case sig := <-sigs:
		switch sig {
		case os.Interrupt:
			zlog.Infow("Shutting down",
				"signal", "SIGINT",
			)
		case syscall.SIGTERM:
			zlog.Infow("Shutting down",
				"signal", "SIGTERM",
			)
		}

		cancel()
		os.Exit(0)
	}
}

func init() {
	rootCmd.AddCommand(serveCmd)
}
