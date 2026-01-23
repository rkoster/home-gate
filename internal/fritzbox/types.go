package fritzbox

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

type Landevice struct {
	UID          string `json:"UID"`
	FriendlyName string `json:"friendly_name"`
	MAC          string `json:"mac"`
	Active       string `json:"active"`
	UserUIDs     string `json:"user_UIDs"`
	Blocked      string `json:"blocked"`
}

type LandeviceResponse struct {
	Landevice []Landevice `json:"landevice"`
}

type MonitorConfig struct {
	DisplayHomenetDevices string `json:"displayHomenetDevices"`
}
