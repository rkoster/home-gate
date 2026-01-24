// Package monitor provides the core monitoring logic that was previously managed via the CLI command.
// This package exposes a Run function that handles monitoring based on options suitable for CLI or background use.
package monitor

import (
	"context"
	"errors"
	"fmt"
	"home-gate/internal/fritzbox"
	"home-gate/internal/policy"
	"io"
	"strings"
	"time"
)

// Options holds the arguments for a monitor run.
type Options struct {
	Username          string
	Password          string
	Mac               string
	Period            string
	ActivityThreshold float64
	PolicyString      string
	Enforce           bool
	Out               io.Writer
	// TestClient is used only for dependency injection in testing. Leave nil in production.
	TestClient fritzbox.Client
}

// Summary holds high-level details about a monitoring run.
type Summary struct {
	DevicesChecked int
	UsersFetched   int
	Errors         []error
	StartTime      time.Time
	Duration       time.Duration
}

// Run executes a monitoring run with the given options, returning a summary.
func Run(ctx context.Context, opts Options) (Summary, error) {
	start := time.Now()
	w := opts.Out
	var summary Summary
	if w == nil {
		w = io.Discard
	}

	if opts.Username == "" || opts.Password == "" {
		err := errors.New("username and password are required")
		summary.Errors = append(summary.Errors, err)
		return summary, err
	}

	var client fritzbox.Client
	if opts.TestClient != nil {
		client = opts.TestClient
	} else {
		client = fritzbox.New(opts.Username, opts.Password)
	}
	_, _ = fmt.Fprintln(w, "Connecting to Fritz!Box")
	if err := client.Connect(); err != nil {
		err = fmt.Errorf("failed to connect: %w", err)
		summary.Errors = append(summary.Errors, err)
		return summary, err
	}
	_, _ = fmt.Fprintln(w, "Connected")

	var pm *policy.PolicyManager
	if opts.PolicyString != "" {
		var err error
		pm, err = policy.NewPolicyManager(opts.PolicyString)
		if err != nil {
			err = fmt.Errorf("failed to parse policy: %w", err)
			summary.Errors = append(summary.Errors, err)
			return summary, err
		}
	}

	_, _ = fmt.Fprintln(w, "Fetching landevices")
	landevices, err := client.GetLandevices()
	if err != nil {
		err = fmt.Errorf("failed to fetch landevices: %w", err)
		summary.Errors = append(summary.Errors, err)
		return summary, err
	}
	_, _ = fmt.Fprintf(w, "Fetched %d devices\n", len(landevices))
	summary.DevicesChecked = len(landevices)

	macToUserUID := make(map[string]string)
	for _, dev := range landevices {
		if dev.UserUIDs != "" {
			normalizedMac := strings.ToLower(strings.ReplaceAll(dev.MAC, ":", ""))
			macToUserUID[normalizedMac] = dev.UserUIDs
		}
	}
	summary.UsersFetched = len(macToUserUID)

	var config fritzbox.MonitorConfig
	if opts.Mac == "" {
		config, err = client.GetMonitorConfig()
		if err != nil {
			err = fmt.Errorf("failed to fetch monitor config: %w", err)
			summary.Errors = append(summary.Errors, err)
			return summary, err
		}
	}

	var targetMACs []string
	var targetNames []string
	if opts.Mac != "" {
		normalizedMac := strings.ToLower(strings.ReplaceAll(opts.Mac, ":", ""))
		targetMACs = []string{normalizedMac}
		targetNames = []string{opts.Mac}
	} else {
		_, _ = fmt.Fprintln(w, "No MAC specified, fetching configured devices")
		uids := strings.Split(config.DisplayHomenetDevices, ",")
		_, _ = fmt.Fprintf(w, "Configured UIDs: %v\n", uids)
		for _, uid := range uids {
			for _, dev := range landevices {
				if dev.UID == uid {
					normalizedMac := strings.ToLower(strings.ReplaceAll(dev.MAC, ":", ""))
					targetMACs = append(targetMACs, normalizedMac)
					targetNames = append(targetNames, dev.FriendlyName)
					_, _ = fmt.Fprintf(w, "Added device: %s (%s)\n", dev.FriendlyName, normalizedMac)
					break
				}
			}
		}
		_, _ = fmt.Fprintf(w, "Total target devices: %d\n", len(targetMACs))
	}

	var subset string
	var intervalSeconds float64
	switch opts.Period {
	case "hour":
		subset = "subset0001"
		intervalSeconds = 60
	case "day":
		subset = "subset0002"
		intervalSeconds = 900
	default:
		err = fmt.Errorf("invalid period: %s. Use 'hour' or 'day'", opts.Period)
		summary.Errors = append(summary.Errors, err)
		return summary, err
	}

	response, err := client.GetMonitorData("macaddrs", subset)
	if err != nil {
		err = fmt.Errorf("failed to fetch monitor data: %w", err)
		summary.Errors = append(summary.Errors, err)
		return summary, err
	}

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
			_, _ = fmt.Fprintf(w, "MAC %s not found in data\n", name)
			summary.Errors = append(summary.Errors, fmt.Errorf("MAC %s not found in data", name))
			continue
		}

		var device fritzbox.Landevice
		for _, dev := range landevices {
			if strings.ToLower(strings.ReplaceAll(dev.MAC, ":", "")) == normalizedMac {
				device = dev
				break
			}
		}

		if opts.Period == "hour" {
			var totalRcv, totalSnd int64
			for _, val := range rcvMeasurements {
				totalRcv += int64(val * intervalSeconds)
			}
			for _, val := range sndMeasurements {
				totalSnd += int64(val * intervalSeconds)
			}
			_, _ = fmt.Fprintf(w, "%s usage in last hour:\n", name)
			_, _ = fmt.Fprintf(w, "Downstream: %d bytes\n", totalRcv)
			_, _ = fmt.Fprintf(w, "Upstream: %d bytes\n", totalSnd)
		} else {
			now := time.Now()
			minutesPastMidnight := now.Hour()*60 + now.Minute()
			intervalsSinceMidnight := minutesPastMidnight / 15
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
				if rcv > opts.ActivityThreshold || snd > opts.ActivityThreshold {
					dailyActiveCount++
				}
			}
			dailyActiveMinutes := dailyActiveCount * 15

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
				isActive := rcv > opts.ActivityThreshold || snd > opts.ActivityThreshold
				activity = append(activity, isActive)
				if isActive {
					activeCount++
				}
			}
			activeMinutes := activeCount * 15

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
			_, _ = fmt.Fprintf(w, "%s activity in last 12 hours:\n", name)
			_, _ = fmt.Fprintf(w, "Active: %d minutes (%d/%d intervals)\n", activeMinutes, activeCount, numIntervals)
			_, _ = fmt.Fprintf(w, "Daily total: %d minutes (%d/96 intervals)\n", dailyActiveMinutes, dailyActiveCount)
			if pm != nil {
				allowed := pm.AllowedToday()
				if dailyActiveMinutes < allowed {
					_, _ = fmt.Fprintf(w, "Within policy\n")
					if opts.Enforce && device.Blocked == "1" {
						userUID := device.UserUIDs
						if userUID == "" {
							if u, ok := macToUserUID[normalizedMac]; ok {
								userUID = u
							}
						}
						if userUID != "" {
							err := client.BlockDevice(userUID, false)
							if err != nil {
								_, _ = fmt.Fprintf(w, "Failed to unblock device: %v\n", err)
								summary.Errors = append(summary.Errors, fmt.Errorf("failed to unblock device: %w", err))
							} else {
								_, _ = fmt.Fprintf(w, "Device unblocked\n")
							}
						} else {
							_, _ = fmt.Fprintf(w, "No user UID found for device, cannot unblock\n")
							summary.Errors = append(summary.Errors, fmt.Errorf("cannot unblock, no user UID for device"))
						}
					}
				} else {
					_, _ = fmt.Fprintf(w, "Exceeded policy\n")
					if opts.Enforce {
						userUID := device.UserUIDs
						if userUID == "" {
							if u, ok := macToUserUID[normalizedMac]; ok {
								userUID = u
							}
						}
						if userUID == "" {
							userUID = device.UID
						}
						if userUID != "" {
							_, _ = fmt.Fprintf(w, "Blocking using UID: %s\n", userUID)
							err := client.BlockDevice(userUID, true)
							if err != nil {
								_, _ = fmt.Fprintf(w, "Failed to block device: %v\n", err)
								summary.Errors = append(summary.Errors, fmt.Errorf("failed to block device: %w", err))
							} else {
								_, _ = fmt.Fprintf(w, "Device blocked\n")
							}
						} else {
							_, _ = fmt.Fprintf(w, "No user UID found for device, cannot block\n")
							summary.Errors = append(summary.Errors, fmt.Errorf("cannot block, no user UID for device"))
						}
					}
				}
			}
			_, _ = fmt.Fprintf(w, "Timeline: %s\n", viz.String())
		}
		_, _ = fmt.Fprintln(w)
	}

	summary.Duration = time.Since(start)
	summary.StartTime = start
	return summary, nil
}
