package Prometheus

import (
	"fmt"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"

	"omada_exporter_go/internal/Omada/Model/VPN"
	"omada_exporter_go/internal/Prometheus/Utils"
)

const (
	label_vpnName       string = "vpnName"
	label_vpnUserName   string = "userName"
	label_vpnInterface  string = "interfaceName"
	label_vpnServerType string = "serverType"
	label_vpnType       string = "vpnType"
	label_vpnRemoteIp   string = "remoteIp"
	label_vpnUptime     string = "uptime"
)

var vpnInfoLabels = []string{
	label_vpnName, label_vpnUserName, label_vpnInterface,
	label_vpnServerType, label_vpnType, label_vpnRemoteIp, label_vpnUptime,
}

var (
	vpn_tunnel_up_bytes_total = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "vpn_tunnel_up_bytes_total",
			Help: "Uplink traffic in bytes since the VPN tunnel connected",
		},
		vpnIdentityLabels,
	)
	vpn_tunnel_down_bytes_total = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "vpn_tunnel_down_bytes_total",
			Help: "Downlink traffic in bytes since the VPN tunnel connected",
		},
		vpnIdentityLabels,
	)
	vpn_tunnel_up_packets_total = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "vpn_tunnel_up_packets_total",
			Help: "Uplink packet count since the VPN tunnel connected",
		},
		vpnIdentityLabels,
	)
	vpn_tunnel_down_packets_total = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "vpn_tunnel_down_packets_total",
			Help: "Downlink packet count since the VPN tunnel connected",
		},
		vpnIdentityLabels,
	)
	vpn_tunnel_info = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "vpn_tunnel_info",
			Help: fmt.Sprintf("Information about an active VPN tunnel (serverType: Server/Client, vpnType: L2TP/PPTP/IPSec/OpenVPN)"),
		},
		append(vpnIdentityLabels, vpnInfoLabels...),
	)
)

func ExposeVpnMetrics(tunnels []VPN.VpnTunnel) {
	for _, t := range tunnels {
		labels := getVpnIdentityLabels(t)

		vpn_tunnel_up_bytes_total.With(labels).Set(float64(t.UpBytes))
		vpn_tunnel_down_bytes_total.With(labels).Set(float64(t.DownBytes))
		vpn_tunnel_up_packets_total.With(labels).Set(float64(t.UpPkts))
		vpn_tunnel_down_packets_total.With(labels).Set(float64(t.DownPkts))

		setVpnInfo(t, labels)
	}
}

func getVpnIdentityLabels(t VPN.VpnTunnel) prometheus.Labels {
	return prometheus.Labels{
		label_vpnId:  fmt.Sprintf("%d", t.VpnId),
		label_userId: fmt.Sprintf("%d", t.UserId),
	}
}

func setVpnInfo(t VPN.VpnTunnel, labels prometheus.Labels) {
	vpn_tunnel_info.DeletePartialMatch(labels)
	vpn_tunnel_info.With(Utils.AppendMaps(labels, map[string]string{
		label_vpnName:       t.VpnName,
		label_vpnUserName:   t.UserName,
		label_vpnInterface:  t.InterfaceName,
		label_vpnServerType: t.GetServerTypeString(),
		label_vpnType:       t.GetVpnTypeString(),
		label_vpnRemoteIp:   t.RemoteIp,
		label_vpnUptime:     t.Uptime,
	})).Set(1)
}
