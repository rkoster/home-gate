package fritzbox

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	fritzboxlib "github.com/ByteSizedMarius/go-fritzbox-api/v2"
)

//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6 -o fritzboxfakes/fake_client.go . Client

type Client interface {
	Connect() error
	RestGet(path string) ([]byte, int, error)
	SID() string
	GetLandevices() ([]Landevice, error)
	GetMonitorConfig() (MonitorConfig, error)
	GetMonitorDatasets() ([]Dataset, error)
	GetMonitorData(dataset, subset string) ([]SubsetData, error)
	BlockDevice(userUID string, block bool) error
}

type fritzboxClient struct {
	client *fritzboxlib.Client
}

func New(username, password string) Client {
	c := fritzboxlib.New(username, password)
	c.BaseUrl = "http://192.168.2.1"
	return &fritzboxClient{client: c}
}

func (c *fritzboxClient) Connect() error {
	return c.client.Connect()
}

func (c *fritzboxClient) RestGet(path string) ([]byte, int, error) {
	return c.client.RestGet(path)
}

func (c *fritzboxClient) SID() string {
	return c.client.SID()
}

func (c *fritzboxClient) GetLandevices() ([]Landevice, error) {
	jsonData, _, err := c.RestGet("/api/v0/landevice")
	if err != nil {
		return nil, err
	}

	var resp LandeviceResponse
	if err := json.Unmarshal(jsonData, &resp); err != nil {
		return nil, err
	}
	return resp.Landevice, nil
}

func (c *fritzboxClient) GetMonitorConfig() (MonitorConfig, error) {
	jsonData, _, err := c.RestGet("/api/v0/monitor/configuration")
	if err != nil {
		return MonitorConfig{}, err
	}

	var config MonitorConfig
	if err := json.Unmarshal(jsonData, &config); err != nil {
		return MonitorConfig{}, err
	}
	return config, nil
}

func (c *fritzboxClient) GetMonitorDatasets() ([]Dataset, error) {
	jsonData, _, err := c.RestGet("/api/v0/monitor/datasets")
	if err != nil {
		return nil, err
	}

	var datasets []Dataset
	if err := json.Unmarshal(jsonData, &datasets); err != nil {
		return nil, err
	}
	return datasets, nil
}

func (c *fritzboxClient) GetMonitorData(dataset, subset string) ([]SubsetData, error) {
	jsonData, _, err := c.RestGet("/api/v0/monitor/" + dataset + "/" + subset)
	if err != nil {
		return nil, err
	}

	var data []SubsetData
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, err
	}
	return data, nil
}

func (c *fritzboxClient) BlockDevice(userUID string, block bool) error {
	data := url.Values{}
	data.Set("xhr", "1")
	data.Set("sid", c.client.SID())
	data.Set("edit-profiles", "")
	data.Set("blocked", fmt.Sprintf("%t", block))
	data.Set("toBeBlocked", userUID)
	data.Set("lang", "en")
	data.Set("page", "kidLis")

	req, err := http.NewRequest("POST", c.client.BaseUrl+"/data.lua", strings.NewReader(data.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "*/*")
	req.Header.Set("Accept-Language", "en-GB,en;q=0.6")
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("Origin", strings.TrimSuffix(c.client.BaseUrl, "/"))
	req.Header.Set("Referer", c.client.BaseUrl+"/")
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
