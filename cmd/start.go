package cmd

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	// _ "github.com/danesparza/daydash-service/docs" // swagger docs location
	"github.com/danesparza/daydash-service/telemetry"
	"github.com/gorilla/mux"
	"github.com/newrelic/go-agent/v3/integrations/nrgorilla"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	httpSwagger "github.com/swaggo/http-swagger"
)

var NRLicense = "Unknown"
var NRAppName = "Dashboard"

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
		"nrAppName", NRAppName,
	)

	//	Create an api service object
	/*
		apiService := api.Service{
			StartTime: time.Now(),
		}
	*/

	//	Trap program exit appropriately
	ctx, cancel := context.WithCancel(context.Background())
	sigs := make(chan os.Signal, 2)
	signal.Notify(sigs, os.Interrupt, syscall.SIGTERM)
	go handleSignals(ctx, sigs, cancel)

	//	Create a router and setup our REST and UI endpoints...
	restRouter := mux.NewRouter()
	restRouter.Use(nrgorilla.Middleware(telemetry.NRApp))

	//	MAP ROUTES
	/*
		restRouter.HandleFunc("/v1/image/map/{lat},{long}", apiService.GetMapImageForCoordinates).Methods("GET")        // Get the map at the given coordinates and default zoom
		restRouter.HandleFunc("/v1/image/map/{lat},{long}/{zoom}", apiService.GetMapImageForCoordinates).Methods("GET") // Get the map at the given coordinates

		//	CONFIG ROUTES
		restRouter.HandleFunc("/v1/config", apiService.GetConfig).Methods("GET")  // Get config
		restRouter.HandleFunc("/v1/config", apiService.SetConfig).Methods("POST") // Update config

		//	DASHBOARD DATA ROUTES
		restRouter.HandleFunc("/v1/dashboard/pollen", apiService.GetPollen).Methods("GET")           // Get pollen data
		restRouter.HandleFunc("/v1/dashboard/weather", apiService.GetWeather).Methods("GET")         // Get weather data
		restRouter.HandleFunc("/v1/dashboard/nwsalerts", apiService.GetWeatherAlerts).Methods("GET") // Get weather alerts data
		restRouter.HandleFunc("/v1/dashboard/news", apiService.GetNews).Methods("GET")               // Get breaking news
		restRouter.HandleFunc("/v1/dashboard/calendar", apiService.GetCalendar).Methods("GET")       // Get calendar data
		restRouter.HandleFunc("/v1/dashboard/earthquakes", apiService.GetEarthquakes).Methods("GET") // Get earthquake data

		restRouter.HandleFunc("/v1/dashboard/zipgeo/{zipcode}", apiService.GetGeoForZipcode).Methods("GET") // Get lat/long for a zipcode
	*/

	//	SWAGGER ROUTES
	restRouter.PathPrefix("/v1/swagger").Handler(httpSwagger.WrapHandler)

	//	Start the service and display how to access it
	go func() {
		zlog.Infow("Started REST service")
		zlog.Errorw("API service error",
			"error", http.ListenAndServe(viper.GetString("server.bind")+":"+viper.GetString("server.port"), restRouter),
		)
	}()

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
