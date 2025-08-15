package cmd

import (
	"fmt"
	"os"

	"llmbench/internal/config"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cfgFile   string
	configMgr *config.Manager
	rootCmd   = &cobra.Command{
		Use:   "llmbench",
		Short: "A benchmark tool for LLM service providers",
		Long: `LLMBench is a CLI tool for benchmarking and comparing LLM service providers
that use OpenAI-compliant APIs. It supports multiple providers, concurrent requests,
and provides detailed performance metrics.`,
	}
)

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	// Global flags
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.config/llmbench/llmbench.yaml)")
	rootCmd.PersistentFlags().Bool("verbose", false, "verbose output")

	// Bind flags to viper
	viper.BindPFlag("verbose", rootCmd.PersistentFlags().Lookup("verbose"))
}

// initConfig reads in config file and ENV variables.
func initConfig() {
	configMgr = config.NewManager()
	
	// Skip config loading for config init command to avoid chicken-and-egg problem
	if len(os.Args) >= 3 && os.Args[1] == "config" && os.Args[2] == "init" {
		return
	}
	
	if err := configMgr.Load(cfgFile); err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}
}
