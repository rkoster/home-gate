package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"strings"

	fritzbox "github.com/ByteSizedMarius/go-fritzbox-api/v2"
)

type Dataset struct {
	DataSources []DataSource `json:"dataSources"`
	Type        string       `json:"type"`
	Subsets     []Subset     `json:"subsets"`
	UID         string       `json:"UID"`
}

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

type DataPoint struct {
	Time   int64         `json:"time"`
	Values []interface{} `json:"values"`
}

func main() {
	username := flag.String("username", "", "Fritzbox username")
	password := flag.String("password", "", "Fritzbox password")
	mac := flag.String("mac", "", "MAC address to query usage for")
	period := flag.String("period", "hour", "Period to query: hour or day")
	flag.Parse()

	if *username == "" || *password == "" || *mac == "" {
		log.Fatal("username, password, and mac flags are required")
	}

	// Normalize MAC: remove colons and lowercase
	normalizedMac := strings.ToLower(strings.ReplaceAll(*mac, ":", ""))

	client := fritzbox.New(*username, *password)
	client.BaseUrl = "http://192.168.2.1"

	if err := client.Connect(); err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
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
		log.Fatalf("MAC %s not found in subset data", *mac)
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

		fmt.Printf("MAC %s usage in last hour:\n", *mac)
		fmt.Printf("Downstream: %d bytes\n", totalRcv)
		fmt.Printf("Upstream: %d bytes\n", totalSnd)
	} else { // day
		// Count active intervals: where either rcv or snd > 0
		activeCount := 0
		for i := range rcvMeasurements {
			if rcvMeasurements[i] > 0 || (i < len(sndMeasurements) && sndMeasurements[i] > 0) {
				activeCount++
			}
		}
		activeMinutes := activeCount * 15 // 15 min intervals

		fmt.Printf("MAC %s activity in last day:\n", *mac)
		fmt.Printf("Active for %d minutes (%d out of %d intervals)\n", activeMinutes, activeCount, len(rcvMeasurements))
	}
}
