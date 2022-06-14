package cmd

import (
	"fmt"
	"os"

	"github.com/danesparza/daydash-service/internal/logger"
	homedir "github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

var (
	cfgFile    string
	rootlogger *zap.Logger
	zlog       *zap.SugaredLogger
)

type fwdToZapWriter struct {
	logger *zap.SugaredLogger
}

func (fw *fwdToZapWriter) Write(p []byte) (n int, err error) {
	fw.logger.Errorw(string(p))
	return len(p), nil
}

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "daydash-service",
	Short: "A REST data service for daydash",
	Long:  `daydash-service is a RESTful data service for daydash`,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/daydash-service.yaml)")

	rootlogger, _ = logger.NewProd()
	defer rootlogger.Sync() // flushes buffer, if any
	zlog = rootlogger.Sugar()
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	// Find home directory.
	home, err := homedir.Dir()
	if err != nil {
		zlog.Errorw(
			"Couldn't find home directory",
			"error", err,
		)
	}

	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		viper.AddConfigPath(home)              // adding home directory as first search path
		viper.AddConfigPath(".")               // also look in the working directory
		viper.SetConfigName("daydash-service") // name the config file (without extension)
	}

	viper.AutomaticEnv() // read in environment variables that match

	//	Set our defaults
	viper.SetDefault("server.port", "80")
	viper.SetDefault("server.httponly", false)
	viper.SetDefault("server.allowed-origins", "*")
	viper.SetDefault("log.level", "info")

	// If a config file is found, read it in
	viper.ReadInConfig()

	//	Set the log level based on configuration:
	//	Need to move this to the package level
	// loglevel := viper.GetString("log.level")

}
