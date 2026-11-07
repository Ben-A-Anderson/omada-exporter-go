package Clients

import (
	"omada_exporter_go/internal/Log"
	"omada_exporter_go/internal/Omada/HttpClient/ApiClient"
)

const path_OpenApiClientsList = "/openapi/v1/{omadaID}/sites/{siteID}/clients"

type Client struct {
	MacAddress     string  `json:"mac"`
	Name           string  `json:"name"`
	HostName       string  `json:"hostName"`
	Vendor         string  `json:"vendor"`
	DeviceType     string  `json:"deviceType"`
	DeviceCategory string  `json:"deviceCategory"`
	IP             string  `json:"ip"`
	Wireless       bool    `json:"wireless"`
	Active         bool    `json:"active"`
	Guest          bool    `json:"guest"`
	Vid            float64 `json:"vid"`
	NetworkName    string  `json:"networkName"`

	// wireless
	SSID        string  `json:"ssid"`
	ApName      string  `json:"apName"`
	ApMac       string  `json:"apMac"`
	RadioID     float64 `json:"radioId"`
	Channel     float64 `json:"channel"`
	RSSI        float64 `json:"rssi"`
	SNR         float64 `json:"snr"`
	SignalLevel float64 `json:"signalLevel"`
	SignalRank  float64 `json:"signalRank"`
	HealthScore float64 `json:"healthScore"`
	RxRate      float64 `json:"rxRate"`
	TxRate      float64 `json:"txRate"`
	PowerSave   bool    `json:"powerSave"`

	// wired
	SwitchName string  `json:"switchName"`
	SwitchMac  string  `json:"switchMac"`
	Port       float64 `json:"port"`

	// counters / state
	TrafficDown    float64 `json:"trafficDown"`
	TrafficUp      float64 `json:"trafficUp"`
	DownPacket     float64 `json:"downPacket"`
	UpPacket       float64 `json:"upPacket"`
	Activity       float64 `json:"activity"`
	UploadActivity float64 `json:"uploadActivity"`
	Uptime         float64 `json:"uptime"`
	LastSeen       float64 `json:"lastSeen"` // milliseconds
}

func Get() (*[]Client, error) {
	Log.Debug("Fetching clients data")
	result, err := ApiClient.Get[Client](ApiClient.GetInstance(), path_OpenApiClientsList, nil, nil, true)
	if err != nil {
		return nil, Log.Error(err, "Failed to get clients data")
	}
	return result, nil
}
