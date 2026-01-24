package cmd

import (
	"context"
	"fmt"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"home-gate/internal/monitor"
	"os"
)

// monitorCmd represents the monitor command
var monitorCmd = &cobra.Command{
	Use:   "monitor",
	Short: "Monitor device usage",
	Long:  `Monitor Fritz!Box device usage and check against policies.`,
	Run: func(cmd *cobra.Command, args []string) {
		_ = viper.BindPFlags(cmd.Flags())
		runMonitor()
	},
}

func init() {
	rootCmd.AddCommand(monitorCmd)

	monitorCmd.Flags().String("username", "", "Fritzbox username")
	monitorCmd.Flags().String("password", "", "Fritzbox password")
	monitorCmd.Flags().String("mac", "", "MAC address to query usage for (optional)")
	monitorCmd.Flags().String("period", "day", "Period to query: hour or day")
	monitorCmd.Flags().Float64("activity-threshold", 0, "Minimum Byte/s to consider interval active")
	monitorCmd.Flags().String("policy", "", "Policy string for allowed minutes per day")
	monitorCmd.Flags().Bool("enforce", false, "Enforce policy by blocking devices that exceed limits")

	_ = viper.BindPFlag("username", monitorCmd.Flags().Lookup("username"))
	_ = viper.BindPFlag("password", monitorCmd.Flags().Lookup("password"))
	_ = viper.BindPFlag("mac", monitorCmd.Flags().Lookup("mac"))
	_ = viper.BindPFlag("period", monitorCmd.Flags().Lookup("period"))
	_ = viper.BindPFlag("activity-threshold", monitorCmd.Flags().Lookup("activity-threshold"))
	_ = viper.BindPFlag("policy", monitorCmd.Flags().Lookup("policy"))
	_ = viper.BindPFlag("enforce", monitorCmd.Flags().Lookup("enforce"))

	_ = viper.BindEnv("username", "FRITZBOX_USERNAME")
	_ = viper.BindEnv("password", "FRITZBOX_PASSWORD")
}

func runMonitor() {

	summary, err := monitor.Run(
		context.Background(),
		monitor.Options{
			Username:          viper.GetString("username"),
			Password:          viper.GetString("password"),
			Mac:               viper.GetString("mac"),
			Period:            viper.GetString("period"),
			ActivityThreshold: viper.GetFloat64("activity-threshold"),
			PolicyString:      viper.GetString("policy"),
			Enforce:           viper.GetBool("enforce"),
			Out:               os.Stdout,
		},
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Monitoring error: %v\n", err)
		os.Exit(1)
	}
	_, _ = fmt.Fprintf(os.Stdout, "Monitoring done: checked %d devices, fetched %d users, duration %s\n",
		summary.DevicesChecked, summary.UsersFetched, summary.Duration,
	)
}
