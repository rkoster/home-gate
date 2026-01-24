package cmd

import (
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"time"

	fritzbox "home-gate/internal/fritzbox"
	"home-gate/internal/policy"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// monitorCmd represents the monitor command
var monitorCmd = &cobra.Command{
	Use:   "monitor",
	Short: "Monitor device usage",
	Long:  `Monitor Fritz!Box device usage and check against policies.`,
	Run: func(cmd *cobra.Command, args []string) {
		runMonitor()
	},
}

func init() {
	rootCmd.AddCommand(monitorCmd)

	// Define flags
	monitorCmd.Flags().String("username", "", "Fritzbox username")
	monitorCmd.Flags().String("password", "", "Fritzbox password")
	monitorCmd.Flags().String("mac", "", "MAC address to query usage for (optional)")
	monitorCmd.Flags().String("period", "day", "Period to query: hour or day")
	monitorCmd.Flags().Float64("activity-threshold", 0, "Minimum Byte/s to consider interval active")
	monitorCmd.Flags().String("policy", "", "Policy string for allowed minutes per day")
	monitorCmd.Flags().Bool("enforce", false, "Enforce policy by blocking devices that exceed limits")

	// Bind flags to viper
	_ = viper.BindPFlag("username", monitorCmd.Flags().Lookup("username"))
	_ = viper.BindPFlag("password", monitorCmd.Flags().Lookup("password"))
	_ = viper.BindPFlag("mac", monitorCmd.Flags().Lookup("mac"))
	_ = viper.BindPFlag("period", monitorCmd.Flags().Lookup("period"))
	_ = viper.BindPFlag("activity-threshold", monitorCmd.Flags().Lookup("activity-threshold"))
	_ = viper.BindPFlag("policy", monitorCmd.Flags().Lookup("policy"))
	_ = viper.BindPFlag("enforce", monitorCmd.Flags().Lookup("enforce"))

	// Bind environment variables
	_ = viper.BindEnv("username", "FRITZBOX_USERNAME")
	_ = viper.BindEnv("password", "FRITZBOX_PASSWORD")

	// Flags are required via validation in runMonitor
}

func runMonitor() {
	username := viper.GetString("username")
	password := viper.GetString("password")
	mac := viper.GetString("mac")
	period := viper.GetString("period")
	activityThreshold := viper.GetFloat64("activity-threshold")
	policyStr := viper.GetString("policy")
	enforce := viper.GetBool("enforce")

	if username == "" || password == "" {
		log.Fatal("username and password are required (set via --username/--password flags or FRITZBOX_USERNAME/FRITZBOX_PASSWORD env vars)")
	}

	client := fritzbox.New(username, password)

	fmt.Println("Connecting to Fritz!Box")
	if err := client.Connect(); err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	fmt.Println("Connected")

	var pm *policy.PolicyManager
	if policyStr != "" {
		var err error
		pm, err = policy.NewPolicyManager(policyStr)
		if err != nil {
			log.Fatalf("Failed to parse policy: %v", err)
		}
	}

	fmt.Println("Fetching landevices")
	landevices, err := client.GetLandevices()
	if err != nil {
		log.Fatalf("Failed to fetch landevices: %v", err)
	}
	fmt.Printf("Fetched %d devices\n", len(landevices))

	// Map MAC to userUID
	macToUserUID := make(map[string]string)
	for _, dev := range landevices {
		if dev.UserUIDs != "" {
			normalizedMac := strings.ToLower(strings.ReplaceAll(dev.MAC, ":", ""))
			macToUserUID[normalizedMac] = dev.UserUIDs
		}
	}

	var config fritzbox.MonitorConfig
	if mac == "" {
		config, err = client.GetMonitorConfig()
		if err != nil {
			log.Fatalf("Failed to fetch monitor config: %v", err)
		}
	}

	RunMonitor(client, pm, mac, period, activityThreshold, landevices, config, macToUserUID, enforce, os.Stdout)
}

func RunMonitor(client fritzbox.Client, pm *policy.PolicyManager, mac string, period string, activityThreshold float64, landevices []fritzbox.Landevice, config fritzbox.MonitorConfig, macToUserUID map[string]string, enforce bool, writer io.Writer) {

	var targetMACs []string
	var targetNames []string

	if mac != "" {
		normalizedMac := strings.ToLower(strings.ReplaceAll(mac, ":", ""))
		targetMACs = []string{normalizedMac}
		targetNames = []string{mac}
	} else {
		fmt.Fprintln(writer, "No MAC specified, fetching configured devices")

		uids := strings.Split(config.DisplayHomenetDevices, ",")
		fmt.Fprintf(writer, "Configured UIDs: %v\n", uids)

		for _, uid := range uids {
			for _, dev := range landevices {
				if dev.UID == uid {
					normalizedMac := strings.ToLower(strings.ReplaceAll(dev.MAC, ":", ""))
					targetMACs = append(targetMACs, normalizedMac)
					targetNames = append(targetNames, dev.FriendlyName)
					fmt.Fprintf(writer, "Added device: %s (%s)\n", dev.FriendlyName, normalizedMac)
					break
				}
			}
		}
		fmt.Fprintf(writer, "Total target devices: %d\n", len(targetMACs))
	}

	// Fetch data
	var subset string
	var intervalSeconds float64
	switch period {
	case "hour":
		subset = "subset0001"
		intervalSeconds = 60
	case "day":
		subset = "subset0002"
		intervalSeconds = 900
	default:
		log.Fatalf("Invalid period: %s. Use 'hour' or 'day'", period)
	}

	response, err := client.GetMonitorData("macaddrs", subset)
	if err != nil {
		log.Fatalf("Failed to fetch monitor data: %v", err)
	}

	// Process each target
	for idx, normalizedMac := range targetMACs {
		name := targetNames[idx]

		var rcvMeasurements, sndMeasurements []float64
		for _, sd := range response {
			if strings.HasSuffix(sd.DataSourceName, normalizedMac) {
				if strings.HasPrefix(sd.DataSourceName, "rcv_") {
					rcvMeasurements = sd.Measurements
				} else if strings.HasPrefix(sd.DataSourceName, "snd_") {
					sndMeasurements = sd.Measurements
				}
			}
		}
		if rcvMeasurements == nil || sndMeasurements == nil {
			fmt.Fprintf(writer, "MAC %s not found in data\n", name)
			continue
		}

		// Find the device to check blocked status
		var device fritzbox.Landevice
		for _, dev := range landevices {
			if strings.ToLower(strings.ReplaceAll(dev.MAC, ":", "")) == normalizedMac {
				device = dev
				break
			}
		}

		if period == "hour" {
			var totalRcv, totalSnd int64
			for _, val := range rcvMeasurements {
				totalRcv += int64(val * intervalSeconds)
			}
			for _, val := range sndMeasurements {
				totalSnd += int64(val * intervalSeconds)
			}

			fmt.Fprintf(writer, "%s usage in last hour:\n", name)
			fmt.Fprintf(writer, "Downstream: %d bytes\n", totalRcv)
			fmt.Fprintf(writer, "Upstream: %d bytes\n", totalSnd)
		} else {
			// Compute today's active intervals only (midnight -> now)
			now := time.Now()
			minutesPastMidnight := now.Hour()*60 + now.Minute()
			intervalsSinceMidnight := minutesPastMidnight / 15

			// dailyStart is the index in the measurements that corresponds to today's midnight
			dailyStart := len(rcvMeasurements) - intervalsSinceMidnight
			if dailyStart < 0 {
				dailyStart = 0
			}

			dailyActiveCount := 0
			for i := dailyStart; i < len(rcvMeasurements); i++ {
				rcv := rcvMeasurements[i]
				snd := 0.0
				if i < len(sndMeasurements) {
					snd = sndMeasurements[i]
				}
				if rcv > activityThreshold || snd > activityThreshold {
					dailyActiveCount++
				}
			}
			dailyActiveMinutes := dailyActiveCount * 15

			// Last 12 hours timeline
			numIntervals := 48
			start := len(rcvMeasurements) - numIntervals
			if start < 0 {
				start = 0
				numIntervals = len(rcvMeasurements)
			}

			activeCount := 0
			var activity []bool
			for i := start; i < len(rcvMeasurements); i++ {
				rcv := rcvMeasurements[i]
				snd := 0.0
				if i < len(sndMeasurements) {
					snd = sndMeasurements[i]
				}
				isActive := rcv > activityThreshold || snd > activityThreshold
				activity = append(activity, isActive)
				if isActive {
					activeCount++
				}
			}
			activeMinutes := activeCount * 15

			// Day start marker
			intervalsPastMidnight := intervalsSinceMidnight
			dayStartPos := numIntervals - intervalsPastMidnight

			var viz strings.Builder
			for i, act := range activity {
				if dayStartPos >= 0 && dayStartPos < numIntervals && i == int(dayStartPos) {
					viz.WriteString("|")
				} else if act {
					viz.WriteString("*")
				} else {
					viz.WriteString(".")
				}
			}

			fmt.Fprintf(writer, "%s activity in last 12 hours:\n", name)
			fmt.Fprintf(writer, "Active: %d minutes (%d/%d intervals)\n", activeMinutes, activeCount, numIntervals)
			fmt.Fprintf(writer, "Daily total: %d minutes (%d/96 intervals)\n", dailyActiveMinutes, dailyActiveCount)
			if pm != nil {
				allowed := pm.AllowedToday()
				if dailyActiveMinutes < allowed {
					fmt.Fprintf(writer, "Within policy\n")
					if enforce && device.Blocked == "1" {
						// prefer userUID from the landevice entry, fallback to macToUserUID map
						userUID := device.UserUIDs
						if userUID == "" {
							if u, ok := macToUserUID[normalizedMac]; ok {
								userUID = u
							}
						}
						if userUID != "" {
							err := client.BlockDevice(userUID, false)
							if err != nil {
								fmt.Fprintf(writer, "Failed to unblock device: %v\n", err)
							} else {
								fmt.Fprintf(writer, "Device unblocked\n")
							}
						} else {
							fmt.Fprintf(writer, "No user UID found for device, cannot unblock\n")
						}
					}
				} else {
					fmt.Fprintf(writer, "Exceeded policy\n")
					if enforce {
						// prefer userUID from the landevice entry, fallback to macToUserUID map
						userUID := device.UserUIDs
						if userUID == "" {
							if u, ok := macToUserUID[normalizedMac]; ok {
								userUID = u
							}
						}
						if userUID == "" {
							// try landevice UID as a last resort
							userUID = device.UID
						}
						if userUID != "" {
							fmt.Fprintf(writer, "Blocking using UID: %s\n", userUID)
							err := client.BlockDevice(userUID, true)
							if err != nil {
								fmt.Fprintf(writer, "Failed to block device: %v\n", err)
							} else {
								fmt.Fprintf(writer, "Device blocked\n")
							}
						} else {
							fmt.Fprintf(writer, "No user UID found for device, cannot block\n")
						}
					}
				}
			}
			fmt.Fprintf(writer, "Timeline: %s\n", viz.String())
		}
		fmt.Fprintln(writer)
	}
}
