package main

import (
	"flag"
	"fmt"
	"log"

	fritzbox "home-gate/internal/fritzbox"
	"home-gate/internal/policy"
)



type DataSource struct {
	LandeviceUID   string `json:"landeviceUid"`
	Type           string `json:"type"`
	DataSourceName string `json:"dataSourceName"`
	Unit           string `json:"unit"`
}

type Subset struct {
	Duration       float64 `json:"duration"`
	SampleInterval float64 `json:"sampleInterval"`
	UID            string  `json:"UID"`
}

type SubsetData struct {
	Timestamp      string    `json:"timestamp"`
	DataSourceName string    `json:"dataSourceName"`
	Measurements   []float64 `json:"measurements"`
}

type LandeviceResponse struct {
	Landevice []Landevice `json:"landevice"`
}

type DataPoint struct {
	Time   int64         `json:"time"`
	Values []interface{} `json:"values"`
}

type Landevice struct {
	UID          string `json:"UID"`
	FriendlyName string `json:"friendly_name"`
	MAC          string `json:"mac"`
	Active       string `json:"active"`
	UserUIDs     string `json:"user_UIDs"`
	Blocked      string `json:"blocked"`
}

type MonitorConfig struct {
	DisplayHomenetDevices string `json:"displayHomenetDevices"`
}



func processDeviceHour(name string, rcvMeasurements, sndMeasurements []float64, intervalSeconds float64) {
	var totalRcv, totalSnd int64
	for _, val := range rcvMeasurements {
		totalRcv += int64(val * intervalSeconds)
	}
	for _, val := range sndMeasurements {
		totalSnd += int64(val * intervalSeconds)
	}

	fmt.Printf("%s usage in last hour:\n", name)
	fmt.Printf("Downstream: %d bytes\n", totalRcv)
	fmt.Printf("Upstream: %d bytes\n", totalSnd)
}

func processDeviceDay(name string, rcvMeasurements, sndMeasurements []float64, activityThreshold float64, parsedPolicy map[string]int) {
	dailyActiveCount := 0
	for i := 0; i < len(rcvMeasurements); i++ {
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

	numIntervals := 48
	start := len(rcvMeasurements) - numIntervals
	if start < 0 {
		start = 0
		numIntervals = len(rcvMeasurements)
	}

	activeCount := 0
	for i := start; i < len(rcvMeasurements); i++ {
		rcv := rcvMeasurements[i]
		snd := 0.0
		if i < len(sndMeasurements) {
			snd = sndMeasurements[i]
		}
		if rcv > activityThreshold || snd > activityThreshold {
			activeCount++
		}
	}
	activeMinutes := activeCount * 15

	fmt.Printf("%s activity in last 12 hours:\n", name)
	fmt.Printf("Active: %d minutes (%d/%d intervals)\n", activeMinutes, activeCount, numIntervals)
	fmt.Printf("Daily total: %d minutes (%d/96 intervals)\n", dailyActiveMinutes, dailyActiveCount)
	if parsedPolicy != nil {
		todayAllowed := policy.GetTodayAllowed(parsedPolicy)
		if todayAllowed > 0 {
			if dailyActiveMinutes > todayAllowed {
				fmt.Printf("Exceeded daily limit: %d/%d minutes\n", dailyActiveMinutes, todayAllowed)
			} else {
				fmt.Printf("Within daily limit: %d/%d minutes\n", dailyActiveMinutes, todayAllowed)
			}
		}
	}
}

func processDevices(client fritzbox.Client, targetMACs, targetNames []string, parsedPolicy map[string]int, period string, activityThreshold float64, landevices []fritzbox.Landevice, macToUserUID map[string]string) {
	datasets, err := client.GetMonitorDatasets()
	if err != nil {
		log.Fatalf("Failed to fetch datasets: %v", err)
	}

	var macaddrsDataset *fritzbox.Dataset
	for _, ds := range datasets {
		if ds.UID == "macaddrs" {
			macaddrsDataset = &ds
			break
		}
	}
	if macaddrsDataset == nil {
		log.Fatal("macaddrs dataset not found")
	}

	var subsetUID string
	var intervalSeconds float64
	switch period {
	case "hour":
		subsetUID = "subset0001"
		intervalSeconds = 60
	case "day":
		subsetUID = "subset0002"
		intervalSeconds = 900
	default:
		log.Fatalf("Invalid period: %s", period)
	}

	response, err := client.GetMonitorData("macaddrs", subsetUID)
	if err != nil {
		log.Fatalf("Failed to fetch mac data: %v", err)
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
			fmt.Printf("MAC %s not found\n", name)
			continue
		}

		if period == "hour" {
			processDeviceHour(name, rcvMeasurements, sndMeasurements, intervalSeconds)
		} else {
			processDeviceDay(name, rcvMeasurements, sndMeasurements, activityThreshold, parsedPolicy)
		}
		fmt.Println()
	}
}

	var macaddrsDataset *fritzbox.Dataset
	for _, ds := range datasets {
		if ds.UID == "macaddrs" {
			macaddrsDataset = &ds
			break
		}
	}
	if macaddrsDataset == nil {
		log.Fatal("macaddrs dataset not found")
	}

	var subsetUID string
	var intervalSeconds float64
	switch period {
	case "hour":
		subsetUID = "subset0001"
		intervalSeconds = 60
	case "day":
		subsetUID = "subset0002"
		intervalSeconds = 900
	default:
		log.Fatalf("Invalid period: %s", period)
	}

	response, err := client.GetMonitorData("macaddrs", subsetUID)
	if err != nil {
		log.Fatalf("Failed to fetch mac data: %v", err)
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
			fmt.Printf("MAC %s not found\n", name)
			continue
		}

		if period == "hour" {
			var totalRcv, totalSnd int64
			for _, val := range rcvMeasurements {
				totalRcv += int64(val * intervalSeconds)
			}
			for _, val := range sndMeasurements {
				totalSnd += int64(val * intervalSeconds)
			}

			fmt.Printf("%s usage in last hour:\n", name)
			fmt.Printf("Downstream: %d bytes\n", totalRcv)
			fmt.Printf("Upstream: %d bytes\n", totalSnd)
		} else {
			dailyActiveCount := 0
			for i := 0; i < len(rcvMeasurements); i++ {
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

			numIntervals := 48
			start := len(rcvMeasurements) - numIntervals
			if start < 0 {
				start = 0
				numIntervals = len(rcvMeasurements)
			}

			activeCount := 0
			for i := start; i < len(rcvMeasurements); i++ {
				rcv := rcvMeasurements[i]
				snd := 0.0
				if i < len(sndMeasurements) {
					snd = sndMeasurements[i]
				}
				if rcv > activityThreshold || snd > activityThreshold {
					activeCount++
				}
			}
			activeMinutes := activeCount * 15

			fmt.Printf("%s activity in last 12 hours:\n", name)
			fmt.Printf("Active: %d minutes (%d/%d intervals)\n", activeMinutes, activeCount, numIntervals)
			fmt.Printf("Daily total: %d minutes (%d/96 intervals)\n", dailyActiveMinutes, dailyActiveCount)
			if parsedPolicy != nil {
				todayAllowed := getTodayAllowed(parsedPolicy)
				if todayAllowed > 0 {
					if dailyActiveMinutes > todayAllowed {
						fmt.Printf("Exceeded daily limit: %d/%d minutes\n", dailyActiveMinutes, todayAllowed)
					} else {
						fmt.Printf("Within daily limit: %d/%d minutes\n", dailyActiveMinutes, todayAllowed)
					}
				}
			}
		}
	}
}
			policy[match[1]] = min
		}
	}
	if len(policy) == 0 {
		return nil, fmt.Errorf("no valid policy entries found")
	}
	return policy, nil
}



func processDevices(client *fritzbox.Client, targetMACs, targetNames []string, parsedPolicy map[string]int, period string, activityThreshold float64) {
	// Fetch datasets
	datasetsJSON, _, err := client.RestGet("/api/v0/monitor/datasets")
	if err != nil {
		log.Fatalf("Failed to fetch datasets: %v", err)
	}

	var datasets []Dataset
	if err := json.Unmarshal(datasetsJSON, &datasets); err != nil {
		log.Fatalf("Failed to parse datasets: %v", err)
	}

	var macaddrsDataset *Dataset
	for _, ds := range datasets {
		if ds.UID == "macaddrs" {
			macaddrsDataset = &ds
			break
		}
	}
	if macaddrsDataset == nil {
		log.Fatal("macaddrs dataset not found")
	}

	var subsetUID string
	var intervalSeconds float64
	switch period {
	case "hour":
		subsetUID = "subset0001"
		intervalSeconds = 60
	case "day":
		subsetUID = "subset0002"
		intervalSeconds = 900
	default:
		log.Fatalf("Invalid period: %s", period)
	}

	dataJSON, _, err := client.RestGet("/api/v0/monitor/macaddrs/" + subsetUID)
	if err != nil {
		log.Fatalf("Failed to fetch mac data: %v", err)
	}

	var response []SubsetData
	if err := json.Unmarshal(dataJSON, &response); err != nil {
		log.Fatalf("Failed to parse mac data: %v", err)
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
			fmt.Printf("MAC %s not found\n", name)
			continue
		}

		if period == "hour" {
			var totalRcv, totalSnd int64
			for _, val := range rcvMeasurements {
				totalRcv += int64(val * intervalSeconds)
			}
			for _, val := range sndMeasurements {
				totalSnd += int64(val * intervalSeconds)
			}

			fmt.Printf("%s usage in last hour:\n", name)
			fmt.Printf("Downstream: %d bytes\n", totalRcv)
			fmt.Printf("Upstream: %d bytes\n", totalSnd)
		} else {
			dailyActiveCount := 0
			for i := 0; i < len(rcvMeasurements); i++ {
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

			numIntervals := 48
			start := len(rcvMeasurements) - numIntervals
			if start < 0 {
				start = 0
				numIntervals = len(rcvMeasurements)
			}

			activeCount := 0
			for i := start; i < len(rcvMeasurements); i++ {
				rcv := rcvMeasurements[i]
				snd := 0.0
				if i < len(sndMeasurements) {
					snd = sndMeasurements[i]
				}
				if rcv > activityThreshold || snd > activityThreshold {
					activeCount++
				}
			}
			activeMinutes := activeCount * 15

			fmt.Printf("%s activity in last 12 hours:\n", name)
			fmt.Printf("Active: %d minutes (%d/%d intervals)\n", activeMinutes, activeCount, numIntervals)
			fmt.Printf("Daily total: %d minutes (%d/96 intervals)\n", dailyActiveMinutes, dailyActiveCount)
			if parsedPolicy != nil {
				todayAllowed := getTodayAllowed(parsedPolicy)
				if todayAllowed > 0 {
					if dailyActiveMinutes > todayAllowed {
						fmt.Printf("Exceeded daily limit: %d/%d minutes\n", dailyActiveMinutes, todayAllowed)
					} else {
						fmt.Printf("Within daily limit: %d/%d minutes\n", dailyActiveMinutes, todayAllowed)
					}
				}
			}
		}
	}
}

func blockUnblock(client *fritzbox.Client, sid, userUID string, block bool) error {
	data := url.Values{}
	data.Set("xhr", "1")
	data.Set("sid", sid)
	data.Set("edit-profiles", "")
	data.Set("blocked", fmt.Sprintf("%t", block))
	data.Set("toBeBlocked", userUID)
	data.Set("lang", "en")
	data.Set("page", "kidLis")

	req, err := http.NewRequest("POST", client.BaseUrl+"/data.lua", strings.NewReader(data.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "*/*")
	req.Header.Set("Accept-Language", "en-GB,en;q=0.6")
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("Origin", strings.TrimSuffix(client.BaseUrl, "/"))
	req.Header.Set("Referer", client.BaseUrl+"/")
	req.Header.Set("Sec-GPC", "1")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/143.0.0.0 Safari/537.36")

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	httpClient := &http.Client{Transport: tr}
	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	return nil
}

func main() {
	username := flag.String("username", "", "Fritzbox username")
	password := flag.String("password", "", "Fritzbox password")
	mac := flag.String("mac", "", "MAC address to query usage for (optional, uses configured if empty)")
	period := flag.String("period", "day", "Period to query: hour or day")
	activityThreshold := flag.Float64("activity-threshold", 0, "Minimum Byte/s to consider interval active (default 0)")
	policy := flag.String("policy", "", "Policy string for allowed minutes per day range (e.g., MO-TH90FR120SA-SU180)")
	action := flag.String("action", "", "Action to perform: block or unblock")
	target := flag.String("target", "", "MAC address to block/unblock")
	enforce := flag.Bool("enforce", false, "Enforce policy by blocking devices that exceed limits")
	flag.Parse()

	if *username == "" || *password == "" {
		log.Fatal("username and password flags are required")
	}

	client := fritzbox.New(*username, *password)
	client.BaseUrl = "http://192.168.2.1"

	if err := client.Connect(); err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer client.Close()

	var parsedPolicy map[string]int
	if *policy != "" {
		var err error
		parsedPolicy, err = policy.Parse(*policy)
		if err != nil {
			log.Fatalf("Failed to parse policy: %v", err)
		}
	}

	// Fetch landevices
	landevicesJSON, _, err := client.RestGet("/api/v0/landevice")
	if err != nil {
		log.Fatalf("Failed to fetch landevices: %v", err)
	}

	var landevices LandeviceResponse
	if err := json.Unmarshal(landevicesJSON, &landevices); err != nil {
		log.Fatalf("Failed to parse landevices: %v", err)
	}

	// Map UID to device and MAC to userUID
	uidToDevice := make(map[string]Landevice)
	macToUserUID := make(map[string]string)
	for _, dev := range landevices.Landevice {
		uidToDevice[dev.UID] = dev
		if dev.UserUIDs != "" {
			macToUserUID[strings.ToLower(strings.ReplaceAll(dev.MAC, ":", ""))] = dev.UserUIDs
		}
	}

	var targetMACs = make([]string, 0)
	var targetNames = make([]string, 0)

	if *mac != "" {
		normalizedMac := strings.ToLower(strings.ReplaceAll(*mac, ":", ""))
		targetMACs = []string{normalizedMac}
		targetNames = []string{*mac}
	} else {
		configJSON, _, err := client.RestGet("/api/v0/monitor/configuration")
		if err != nil {
			log.Fatalf("Failed to fetch monitor config: %v", err)
		}

		var config MonitorConfig
		if err := json.Unmarshal(configJSON, &config); err != nil {
			log.Fatalf("Failed to parse monitor config: %v", err)
		}

		uids := strings.Split(config.DisplayHomenetDevices, ",")

		for _, uid := range uids {
			if dev, ok := uidToDevice[uid]; ok {
				normalizedMac := strings.ToLower(strings.ReplaceAll(dev.MAC, ":", ""))
				targetMACs = append(targetMACs, normalizedMac)
				targetNames = append(targetNames, dev.FriendlyName)
			}
		}
	}

	// Fetch datasets
	datasetsJSON, _, err := client.RestGet("/api/v0/monitor/datasets")
	if err != nil {
		log.Fatalf("Failed to fetch datasets: %v", err)
	}

	var datasets []Dataset
	if err := json.Unmarshal(datasetsJSON, &datasets); err != nil {
		log.Fatalf("Failed to parse datasets: %v", err)
	}

	var macaddrsDataset *Dataset
	for _, ds := range datasets {
		if ds.UID == "macaddrs" {
			macaddrsDataset = &ds
			break
		}
	}
	if macaddrsDataset == nil {
		log.Fatal("macaddrs dataset not found")
	}

	var subsetUID string
	var intervalSeconds float64
	switch *period {
	case "hour":
		subsetUID = "subset0001"
		intervalSeconds = 60
	case "day":
		subsetUID = "subset0002"
		intervalSeconds = 900
	default:
		log.Fatalf("Invalid period: %s. Use 'hour' or 'day'", *period)
	}

	dataJSON, _, err := client.RestGet("/api/v0/monitor/macaddrs/" + subsetUID)
	if err != nil {
		log.Fatalf("Failed to fetch mac data: %v", err)
	}

	var response []SubsetData
	if err := json.Unmarshal(dataJSON, &response); err != nil {
		log.Fatalf("Failed to parse mac data: %v", err)
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
			fmt.Printf("MAC %s not found\n", name)
			continue
		}

		if *period == "hour" {
			var totalRcv, totalSnd int64
			for _, val := range rcvMeasurements {
				totalRcv += int64(val * intervalSeconds)
			}
			for _, val := range sndMeasurements {
				totalSnd += int64(val * intervalSeconds)
			}

			fmt.Printf("%s usage in last hour:\n", name)
			fmt.Printf("Downstream: %d bytes\n", totalRcv)
			fmt.Printf("Upstream: %d bytes\n", totalSnd)
		} else {
			dailyActiveCount := 0
			for i := 0; i < len(rcvMeasurements); i++ {
				rcv := rcvMeasurements[i]
				snd := 0.0
				if i < len(sndMeasurements) {
					snd = sndMeasurements[i]
				}
				if rcv > *activityThreshold || snd > *activityThreshold {
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
				isActive := rcv > *activityThreshold || snd > *activityThreshold
				activity = append(activity, isActive)
				if isActive {
					activeCount++
				}
			}
			activeMinutes := activeCount * 15

			now := time.Now()
			minutesPastMidnight := now.Hour()*60 + now.Minute()
			intervalsPastMidnight := minutesPastMidnight / 15
			dayStartPos := numIntervals - intervalsPastMidnight
			if dayStartPos >= 0 && dayStartPos < numIntervals {
				if dayStartPos < len(activity) {
					activity[dayStartPos] = true
				}
			}

			var viz strings.Builder
			for i, active := range activity {
				if dayStartPos >= 0 && i == dayStartPos {
					viz.WriteString("|")
				} else if active {
					viz.WriteString("\033[31m*\033[0m")
				} else {
					viz.WriteString("\033[2m.\033[0m")
				}
			}

			fmt.Printf("%s activity in last 12 hours:\n", name)
			fmt.Printf("Active: %d minutes (%d/%d intervals)\n", activeMinutes, activeCount, numIntervals)
			fmt.Printf("Daily total: %d minutes (%d/96 intervals)\n", dailyActiveMinutes, dailyActiveCount)
			if parsedPolicy != nil {
				todayAllowed := getTodayAllowed(parsedPolicy)
				if todayAllowed > 0 {
					if dailyActiveMinutes > todayAllowed {
						fmt.Printf("Exceeded daily limit: %d/%d minutes\n", dailyActiveMinutes, todayAllowed)
					} else {
						fmt.Printf("Within daily limit: %d/%d minutes\n", dailyActiveMinutes, todayAllowed)
					}
				}
			}
			fmt.Printf("Timeline: %s\n", viz.String())
		}
		fmt.Println()
	}
}
	}

	client := fritzbox.New(*username, *password)
	client.BaseUrl = "http://192.168.2.1"

	fmt.Println("Connecting to Fritz!Box")
	if err := client.Connect(); err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer client.Close()
	fmt.Println("Connected")

	var parsedPolicy map[string]int

	}
	defer client.Close()

	if *policy != "" {
		policyMap, err = policy.Parse(*policy)
		if err != nil {
			log.Fatalf("Failed to parse policy: %v", err)
		}
	}

	fmt.Println("Fetching landevices")
	// Fetch landevices always needed for user UIDs
	landevicesJSON, _, err := client.RestGet("/api/v0/landevice")
	if err != nil {
		log.Fatalf("Failed to fetch landevices: %v", err)
	}
	fmt.Printf("Fetched landevices: %d bytes\n", len(landevicesJSON))

	var landevices LandeviceResponse
	if err := json.Unmarshal(landevicesJSON, &landevices); err != nil {
		log.Fatalf("Failed to parse landevices: %v", err)
	}
	fmt.Printf("Parsed %d devices\n", len(landevices.Landevice))

	// Map UID to device and MAC to userUID
	uidToDevice := make(map[string]Landevice)
	macToUserUID := make(map[string]string)
	for _, dev := range landevices.Landevice {
		uidToDevice[dev.UID] = dev
		if dev.UserUIDs != "" {
			macToUserUID[strings.ToLower(strings.ReplaceAll(dev.MAC, ":", ""))] = dev.UserUIDs
		}
	}

	var targetMACs = make([]string, 0)
	var targetNames = make([]string, 0)

	if *mac != "" {
		// Normalize MAC: remove colons and lowercase
		normalizedMac := strings.ToLower(strings.ReplaceAll(*mac, ":", ""))
		targetMACs = []string{normalizedMac}
		targetNames = []string{*mac} // Use original for display
	} else {
		fmt.Println("No MAC specified, fetching configured devices")
		// Fetch configured devices
		configJSON, _, err := client.RestGet("/api/v0/monitor/configuration")
		if err != nil {
			log.Fatalf("Failed to fetch monitor config: %v", err)
		}
		fmt.Printf("Config JSON: %s\n", string(configJSON))

		var config MonitorConfig
		if err := json.Unmarshal(configJSON, &config); err != nil {
			log.Fatalf("Failed to parse monitor config: %v", err)
		}

		uids := strings.Split(config.DisplayHomenetDevices, ",")
		fmt.Printf("Configured UIDs: %v\n", uids)

		for _, uid := range uids {
			if dev, ok := uidToDevice[uid]; ok {
				normalizedMac := strings.ToLower(strings.ReplaceAll(dev.MAC, ":", ""))
				targetMACs = append(targetMACs, normalizedMac)
				targetNames = append(targetNames, dev.FriendlyName)
				fmt.Printf("Added device: %s (%s)\n", dev.FriendlyName, normalizedMac)
			}
		}
		fmt.Printf("Total target devices: %d\n", len(targetMACs))
	}
	fmt.Printf("Final target devices: %d\n", len(targetNames))
	defer client.Close()

	// Fetch datasets
	datasetsJSON, _, err := client.RestGet("/api/v0/monitor/datasets")
	if err != nil {
		log.Fatalf("Failed to fetch datasets: %v", err)
	}

	var datasets []Dataset
	if err := json.Unmarshal(datasetsJSON, &datasets); err != nil {
		log.Fatalf("Failed to parse datasets: %v", err)
	}

	// Find macaddrs dataset
	var macaddrsDataset *Dataset
	for _, ds := range datasets {
		if ds.UID == "macaddrs" {
			macaddrsDataset = &ds
			break
		}
	}
	if macaddrsDataset == nil {
		log.Fatal("macaddrs dataset not found")
	}

	var subsetUID string
	var intervalSeconds float64
	switch *period {
	case "hour":
		subsetUID = "subset0001"
		intervalSeconds = 60 // 1 min
	case "day":
		subsetUID = "subset0002"
		intervalSeconds = 900 // 15 min
	default:
		log.Fatalf("Invalid period: %s. Use 'hour' or 'day'", *period)
	}

	// Fetch subset data
	dataJSON, _, err := client.RestGet("/api/v0/monitor/macaddrs/" + subsetUID)
	if err != nil {
		log.Fatalf("Failed to fetch mac data: %v", err)
	}

	var response []SubsetData
	if err := json.Unmarshal(dataJSON, &response); err != nil {
		log.Fatalf("Failed to parse mac data: %v", err)
	}

	// Dummy use to avoid unused warnings
	_ = mac
	_ = action
	_ = target

	// Process each target MAC
	for idx, normalizedMac := range targetMACs {
		name := targetNames[idx]

		// Find data for rcv and snd
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
			fmt.Printf("MAC %s (%s) not found in subset data\n", name, normalizedMac)
			continue
		}

		if *period == "hour" {
			// Calculate totals: each measurement is Byte/s for intervalSeconds
			var totalRcv, totalSnd int64
			for _, val := range rcvMeasurements {
				totalRcv += int64(val * intervalSeconds)
			}
			for _, val := range sndMeasurements {
				totalSnd += int64(val * intervalSeconds)
			}

			fmt.Printf("%s (%s) usage in last hour:\n", name, strings.ToUpper(normalizedMac))
			fmt.Printf("Downstream: %d bytes\n", totalRcv)
			fmt.Printf("Upstream: %d bytes\n", totalSnd)
			fmt.Println()
		} else { // day
			// Sum active minutes over full day (96 intervals)
			dailyActiveCount := 0
			for i := 0; i < len(rcvMeasurements); i++ {
				rcv := rcvMeasurements[i]
				snd := 0.0
				if i < len(sndMeasurements) {
					snd = sndMeasurements[i]
				}
				if rcv > *activityThreshold || snd > *activityThreshold {
					dailyActiveCount++
				}
			}
			dailyActiveMinutes := dailyActiveCount * 15

			// Use last 48 intervals for 12-hour timeline
			numIntervals := 48
			start := len(rcvMeasurements) - numIntervals
			if start < 0 {
				start = 0
				numIntervals = len(rcvMeasurements)
			}

			// Count active in last 12 hours for timeline
			activeCount := 0
			var activity []bool
			for i := start; i < len(rcvMeasurements); i++ {
				rcv := rcvMeasurements[i]
				snd := 0.0
				if i < len(sndMeasurements) {
					snd = sndMeasurements[i]
				}
				isActive := rcv > *activityThreshold || snd > *activityThreshold
				activity = append(activity, isActive)
				if isActive {
					activeCount++
				}
			}
			activeMinutes := activeCount * 15

			// Add | for start of day in timeline
			now := time.Now()
			minutesPastMidnight := now.Hour()*60 + now.Minute()
			intervalsPastMidnight := minutesPastMidnight / 15
			dayStartPos := numIntervals - intervalsPastMidnight
			if dayStartPos >= 0 && dayStartPos < numIntervals {
				// Insert | at position
				if dayStartPos < len(activity) {
					activity[dayStartPos] = true // Mark as special, but for now, just note
				}
			}

			// ASCII visualization
			var viz strings.Builder
			for i, active := range activity {
				if dayStartPos >= 0 && i == dayStartPos {
					viz.WriteString("|")
				} else if active {
					viz.WriteString("\033[31m*\033[0m") // Red for active
				} else {
					viz.WriteString("\033[2m.\033[0m") // Dim for inactive
				}
			}

			fmt.Printf("%s (%s) activity in last 12 hours:\n", name, strings.ToUpper(normalizedMac))
			fmt.Printf("Active: %d minutes (%d/%d intervals, threshold %.1f Byte/s)\n", activeMinutes, activeCount, numIntervals, *activityThreshold)
			fmt.Printf("Daily total: %d minutes (%d/96 intervals)\n", dailyActiveMinutes, dailyActiveCount)
			if parsedPolicy != nil {
				todayAllowed := getTodayAllowed(parsedPolicy)
				if todayAllowed > 0 {
					if activeMinutes > todayAllowed {
						fmt.Printf("⚠️  Exceeded daily limit: %d/%d minutes\n", activeMinutes, todayAllowed)
					} else {
						fmt.Printf("✅ Within daily limit: %d/%d minutes\n", activeMinutes, todayAllowed)
					}
				}
			}
				}
			}
			fmt.Printf("Timeline (each char = 15 min, | = day start): %s\n", viz.String())



			fmt.Println()
		}
	}
}
