package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"home-gate/internal/monitor"
	"home-gate/internal/state"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// webCmd is the top-level command for web-related operations (monitoring, api, SPA, etc.)
var webCmd = &cobra.Command{
	Use:   "web",
	Short: "Run background monitoring or serve the web UI/API",
	Run:   runWeb,
}

func init() {
	rootCmd.AddCommand(webCmd)
	webCmd.Flags().String("username", "", "Fritzbox username")
	webCmd.Flags().String("password", "", "Fritzbox password")
	webCmd.Flags().String("mac", "", "MAC address to query usage for (optional)")
	webCmd.Flags().String("period", "day", "Period to query: hour or day")
	webCmd.Flags().Float64("activity-threshold", 0, "Minimum Byte/s to consider interval active")
	webCmd.Flags().String("policy", "", "Policy string for allowed minutes per day")
	webCmd.Flags().Bool("enforce", false, "Enforce policy by blocking devices that exceed limits")
	webCmd.Flags().Duration("interval", 5*time.Minute, "Interval between monitoring runs (default 5m)")

	_ = viper.BindPFlag("username", webCmd.Flags().Lookup("username"))
	_ = viper.BindPFlag("password", webCmd.Flags().Lookup("password"))
	_ = viper.BindPFlag("mac", webCmd.Flags().Lookup("mac"))
	_ = viper.BindPFlag("period", webCmd.Flags().Lookup("period"))
	_ = viper.BindPFlag("activity-threshold", webCmd.Flags().Lookup("activity-threshold"))
	_ = viper.BindPFlag("policy", webCmd.Flags().Lookup("policy"))
	_ = viper.BindPFlag("enforce", webCmd.Flags().Lookup("enforce"))
	_ = viper.BindPFlag("interval", webCmd.Flags().Lookup("interval"))

	_ = viper.BindEnv("username", "FRITZBOX_USERNAME")
	_ = viper.BindEnv("password", "FRITZBOX_PASSWORD")
}

func runWeb(cmd *cobra.Command, args []string) {
	_ = viper.BindPFlags(cmd.Flags()) // Re-bind to ensure flag values are correct
	interval := viper.GetDuration("interval")
	if interval <= 0 {
		fmt.Fprintln(os.Stderr, "Interval must be positive, got", interval)
		os.Exit(1)
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Start HTTP API server alongside monitor
	go func() {
		mux := http.NewServeMux()
		mux.HandleFunc("/status", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			status := state.Get()
			if err := json.NewEncoder(w).Encode(status); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
		})

		addr := ":8080"
		if port := viper.GetString("web-port"); port != "" {
			addr = ":" + port
		}
		server := &http.Server{Addr: addr, Handler: mux}
		go func() {
			<-ctx.Done()
			_ = server.Close()
		}()
		fmt.Printf("[web] API server listening at http://localhost%s/status\n", addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Fprintln(os.Stderr, "[web] HTTP server error:", err)
		}
	}()

	for {
		select {
		case <-ctx.Done():
			fmt.Fprintln(os.Stderr, "\nReceived interrupt, exiting background monitor")
			return
		default:
		}
		start := time.Now()
		fmt.Printf("[web] Starting monitoring at %s\n", start.Format(time.RFC3339))
		summary, err := monitor.Run(ctx, monitor.Options{
			Username:          viper.GetString("username"),
			Password:          viper.GetString("password"),
			Mac:               viper.GetString("mac"),
			Period:            viper.GetString("period"),
			ActivityThreshold: viper.GetFloat64("activity-threshold"),
			PolicyString:      viper.GetString("policy"),
			Enforce:           viper.GetBool("enforce"),
			Out:               io.Discard, // discard monitor logs when running as a daemon
		})
		state.Update(summary)
		if err != nil {
			fmt.Printf("[web] Finished run with errors, checked %d devices, fetched %d users, duration %s\n", summary.DevicesChecked, summary.UsersFetched, summary.Duration)
			for _, e := range summary.Errors {
				fmt.Printf("  error: %v\n", e)
			}
		} else {
			fmt.Printf("[web] Finished run: checked %d devices, fetched %d users, duration %s\n", summary.DevicesChecked, summary.UsersFetched, summary.Duration)
		}
		// Wait for next interval or exit
		select {
		case <-ctx.Done():
			return
		case <-time.After(interval):
		}
	}
}
