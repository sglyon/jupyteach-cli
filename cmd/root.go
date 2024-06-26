/*
Copyright © 2024 Spencer Lyon spencerlyon2@gmail.com
*/
package cmd

import (
	"os"

	"github.com/charmbracelet/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cfgFile string
	path    string
	debug   bool
)

var logger = log.NewWithOptions(os.Stderr, log.Options{
	ReportTimestamp: true,
})

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "jupyteach",
	Short: "Command line interface to interact with the Jupyteach platform",
	Long:  `TODO: long description`,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		if debug {
			logger.SetLevel(log.DebugLevel)
			logger.SetReportCaller(true)
		}
	},
	// Uncomment the following line if your bare application
	// has an action associated with it:
	// Run: func(cmd *cobra.Command, args []string) { },
}

type VersionInfo struct {
	Version string
	Commit  string
	Date    string
	BuiltBy string
}

var versionInfo VersionInfo

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute(vi VersionInfo) {
	versionInfo = vi
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.jupyteach.yaml)")
	rootCmd.PersistentFlags().StringVar(&path, "path", ".", "Path of course contents")
	rootCmd.PersistentFlags().BoolVarP(&debug, "debug", "d", false, "enable debug logging")

}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := os.UserHomeDir()
		cobra.CheckErr(err)

		// Search config in home directory with name ".jupyteach" (without extension).
		viper.AddConfigPath(home)
		viper.SetConfigType("yaml")
		viper.SetConfigName(".jupyteach")
	}

	viper.SetDefault("BASE_URL", "https://app.jupyteach.com")

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err != nil {
		log.Error("Error reading config file:", viper.ConfigFileUsed())
	}

	viper.SetEnvPrefix("jupyteach")
	viper.AutomaticEnv() // read in environment variables that match
}
