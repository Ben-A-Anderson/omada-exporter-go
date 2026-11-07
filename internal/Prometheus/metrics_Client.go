package Prometheus

import (
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"omada_exporter_go/internal"
	"omada_exporter_go/internal/Log"
	"omada_exporter_go/internal/Omada/Model/Clients"
)

const (
	clLblMac      = "macAddress"
	clLblName     = "name"
	clLblHost     = "hostName"
	clLblIP       = "ip"
	clLblVendor   = "vendor"
	clLblDevType  = "deviceType"
	clLblDevCat   = "deviceCategory"
	clLblVlan     = "vlan"
	clLblNetwork  = "networkName"
	clLblSSID     = "ssid"
	clLblWireless = "wireless"
	clLblGuest    = "guest"
	clLblApMac    = "apMac"
	clLblApName   = "apName"
	clLblRadio    = "radioId"
	clLblBand     = "band"
	clLblChannel  = "channel"
	clLblSwMac    = "switchMac"
	clLblSwName   = "switchName"
	clLblPort     = "portID"
	clLblFrom     = "fromApMac"
	clLblTo       = "toApMac"
)

const (
	clMaxRoamGap = 2 * time.Minute // longer gap = reconnect, not a roam
	clStateTTL   = 24 * time.Hour
)

type clResettable interface{ Reset() }

type clState struct {
	apMac string
	seen  time.Time
}

var (
	clReg      = prometheus.NewRegistry()
	clMu       sync.RWMutex
	clSnapshot []clResettable
	clStates   = map[string]*clState{} // touched only by the poller goroutine
)

func clGauge(name, help string, labels []string) *prometheus.GaugeVec {
	v := prometheus.NewGaugeVec(prometheus.GaugeOpts{Name: name, Help: help}, labels)
	clReg.MustRegister(v)
	clSnapshot = append(clSnapshot, v) // reset every poll
	return v
}

func clKeptGauge(name, help string, labels []string) *prometheus.GaugeVec {
	v := prometheus.NewGaugeVec(prometheus.GaugeOpts{Name: name, Help: help}, labels)
	clReg.MustRegister(v) // never reset
	return v
}

var clMacOnly = []string{clLblMac}

var (
	client_info = clGauge("client_info", "Client information (always 1)", []string{
		clLblMac, clLblName, clLblHost, clLblIP, clLblVendor, clLblDevType, clLblDevCat,
		clLblVlan, clLblNetwork, clLblSSID, clLblWireless, clLblGuest})
	client_ap_info = clGauge("client_ap_info", "AP a wireless client is associated with (always 1)",
		[]string{clLblMac, clLblApMac, clLblApName, clLblRadio, clLblBand, clLblChannel, clLblSSID})
	client_port_info = clGauge("client_port_info", "Switch port a wired client is connected to (always 1)",
		[]string{clLblMac, clLblSwMac, clLblSwName, clLblPort})

	client_active                         = clGauge("client_active", "1 if the client is active", clMacOnly)
	client_uptime_seconds                 = clGauge("client_uptime_seconds", "Client connection time in seconds", clMacOnly)
	client_last_seen_timestamp_seconds    = clGauge("client_last_seen_timestamp_seconds", "Controller last-seen time (unix seconds)", clMacOnly)
	client_traffic_down_bytes_total       = clGauge("client_traffic_down_bytes_total", "Bytes downloaded by the client", clMacOnly)
	client_traffic_up_bytes_total         = clGauge("client_traffic_up_bytes_total", "Bytes uploaded by the client", clMacOnly)
	client_packets_down_total             = clGauge("client_packets_down_total", "Packets downloaded by the client", clMacOnly)
	client_packets_up_total               = clGauge("client_packets_up_total", "Packets uploaded by the client", clMacOnly)
	client_activity_down_bytes_per_second = clGauge("client_activity_down_bytes_per_second", "Current download rate", clMacOnly)
	client_activity_up_bytes_per_second   = clGauge("client_activity_up_bytes_per_second", "Current upload rate", clMacOnly)

	client_rssi_dbm             = clGauge("client_rssi_dbm", "Wireless client RSSI in dBm", clMacOnly)
	client_snr_db               = clGauge("client_snr_db", "Wireless client SNR in dB", clMacOnly)
	client_signal_level_percent = clGauge("client_signal_level_percent", "Wireless signal level (0-100)", clMacOnly)
	client_signal_rank          = clGauge("client_signal_rank", "Wireless signal rank (1-5)", clMacOnly)
	client_health_score         = clGauge("client_health_score", "Controller health score", clMacOnly)
	client_rx_rate_kbps         = clGauge("client_rx_rate_kbps", "Negotiated Rx rate in kbps", clMacOnly)
	client_tx_rate_kbps         = clGauge("client_tx_rate_kbps", "Negotiated Tx rate in kbps", clMacOnly)
	client_power_save           = clGauge("client_power_save", "1 if client is in power save", clMacOnly)

	// persistent (not reset between polls)
	client_roams_total = func() *prometheus.CounterVec {
		v := prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "client_roams_total", Help: "AP-to-AP roams observed for the client"}, clMacOnly)
		clReg.MustRegister(v)
		return v
	}()
	client_last_roam_timestamp_seconds = clKeptGauge("client_last_roam_timestamp_seconds", "Unix time of the last observed roam", clMacOnly)
	client_last_roam_info              = clKeptGauge("client_last_roam_info", "From/to AP of the last observed roam (always 1)",
		[]string{clLblMac, clLblFrom, clLblTo})

	client_poll_duration_seconds = func() prometheus.Gauge {
		g := prometheus.NewGauge(prometheus.GaugeOpts{Name: "client_poll_duration_seconds", Help: "Duration of the last client poll"})
		clReg.MustRegister(g)
		return g
	}()
	client_poll_clients = func() prometheus.Gauge {
		g := prometheus.NewGauge(prometheus.GaugeOpts{Name: "client_poll_clients", Help: "Clients returned by the last poll"})
		clReg.MustRegister(g)
		return g
	}()
	client_poll_last_success_timestamp_seconds = func() prometheus.Gauge {
		g := prometheus.NewGauge(prometheus.GaugeOpts{Name: "client_poll_last_success_timestamp_seconds", Help: "Unix time of the last successful poll"})
		clReg.MustRegister(g)
		return g
	}()
	client_poll_errors_total = func() prometheus.Counter {
		c := prometheus.NewCounter(prometheus.CounterOpts{Name: "client_poll_errors_total", Help: "Failed client polls"})
		clReg.MustRegister(c)
		return c
	}()
)

func clBool(b bool) float64 {
	if b {
		return 1
	}
	return 0
}

func clBand(radioID int) string {
	switch radioID {
	case 0:
		return "2.4 GHz"
	case 1:
		return "5 GHz"
	default:
		return "radio-" + strconv.Itoa(radioID)
	}
}

func clExpose(c *Clients.Client) {
	m := prometheus.Labels{clLblMac: c.MacAddress}

	client_info.With(prometheus.Labels{
		clLblMac: c.MacAddress, clLblName: c.Name, clLblHost: c.HostName, clLblIP: c.IP,
		clLblVendor: c.Vendor, clLblDevType: c.DeviceType, clLblDevCat: c.DeviceCategory,
		clLblVlan: strconv.Itoa(int(c.Vid)), clLblNetwork: c.NetworkName, clLblSSID: c.SSID,
		clLblWireless: strconv.FormatBool(c.Wireless), clLblGuest: strconv.FormatBool(c.Guest),
	}).Set(1)

	client_active.With(m).Set(clBool(c.Active))
	client_uptime_seconds.With(m).Set(c.Uptime)
	client_last_seen_timestamp_seconds.With(m).Set(c.LastSeen / 1000)
	client_traffic_down_bytes_total.With(m).Set(c.TrafficDown)
	client_traffic_up_bytes_total.With(m).Set(c.TrafficUp)
	client_packets_down_total.With(m).Set(c.DownPacket)
	client_packets_up_total.With(m).Set(c.UpPacket)
	client_activity_down_bytes_per_second.With(m).Set(c.Activity)
	client_activity_up_bytes_per_second.With(m).Set(c.UploadActivity)

	if c.Wireless {
		client_ap_info.With(prometheus.Labels{
			clLblMac: c.MacAddress, clLblApMac: c.ApMac, clLblApName: c.ApName,
			clLblRadio: strconv.Itoa(int(c.RadioID)), clLblBand: clBand(int(c.RadioID)),
			clLblChannel: strconv.Itoa(int(c.Channel)), clLblSSID: c.SSID,
		}).Set(1)
		client_rssi_dbm.With(m).Set(c.RSSI)
		client_snr_db.With(m).Set(c.SNR)
		client_signal_level_percent.With(m).Set(c.SignalLevel)
		client_signal_rank.With(m).Set(c.SignalRank)
		client_health_score.With(m).Set(c.HealthScore)
		client_rx_rate_kbps.With(m).Set(c.RxRate)
		client_tx_rate_kbps.With(m).Set(c.TxRate)
		client_power_save.With(m).Set(clBool(c.PowerSave))
	} else {
		client_port_info.With(prometheus.Labels{
			clLblMac: c.MacAddress, clLblSwMac: c.SwitchMac, clLblSwName: c.SwitchName,
			clLblPort: strconv.Itoa(int(c.Port)),
		}).Set(1)
	}
}

func clTrackRoam(c *Clients.Client, now time.Time) {
	if !c.Wireless || c.ApMac == "" {
		return
	}
	m := prometheus.Labels{clLblMac: c.MacAddress}
	prev, ok := clStates[c.MacAddress]
	if !ok {
		clStates[c.MacAddress] = &clState{apMac: c.ApMac, seen: now}
		client_roams_total.With(m).Add(0) // create the series at 0 so increase() sees the first roam
		return
	}
	if prev.apMac != c.ApMac && now.Sub(prev.seen) <= clMaxRoamGap {
		client_roams_total.With(m).Inc()
		client_last_roam_timestamp_seconds.With(m).Set(float64(now.Unix()))
		client_last_roam_info.DeletePartialMatch(m)
		client_last_roam_info.With(prometheus.Labels{clLblMac: c.MacAddress, clLblFrom: prev.apMac, clLblTo: c.ApMac}).Set(1)
		Log.Info("Client roamed: %s (%s) %s -> %s", c.Name, c.MacAddress, prev.apMac, c.ApMac)
	}
	prev.apMac = c.ApMac
	prev.seen = now
}

func clPrune(now time.Time) {
	for mac, st := range clStates {
		if now.Sub(st.seen) > clStateTTL {
			delete(clStates, mac)
			m := prometheus.Labels{clLblMac: mac}
			client_roams_total.Delete(m)
			client_last_roam_timestamp_seconds.Delete(m)
			client_last_roam_info.DeletePartialMatch(m)
		}
	}
}

func clPollOnce() {
	start := time.Now()
	list, err := Clients.Get() // network call happens outside the lock
	if err != nil {
		client_poll_errors_total.Inc()
		return
	}
	now := time.Now()

	clMu.Lock()
	defer clMu.Unlock()
	for _, v := range clSnapshot {
		v.Reset()
	}
	n := 0
	for i := range *list {
		c := &(*list)[i]
		if c.MacAddress == "" {
			continue
		}
		clExpose(c)
		clTrackRoam(c, now)
		n++
	}
	clPrune(now)
	client_poll_clients.Set(float64(n))
	client_poll_duration_seconds.Set(time.Since(start).Seconds())
	client_poll_last_success_timestamp_seconds.Set(float64(now.Unix()))
}

func StartClientPoller() {
	interval := internal.GetConfig().Clients.PollInterval
	if interval < time.Second {
		interval = time.Second
	}
	Log.Info("Starting client poller, interval %s", interval)
	go func() {
		clPollOnce()
		t := time.NewTicker(interval)
		defer t.Stop()
		for range t.C {
			clPollOnce()
		}
	}()
}

func ClientMetricsHandler() http.Handler {
	h := promhttp.HandlerFor(clReg, promhttp.HandlerOpts{})
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		clMu.RLock()
		defer clMu.RUnlock()
		h.ServeHTTP(w, r)
	})
}
