package VPN

type VpnTunnel struct {
	VpnId         int32  `json:"vpnId"`
	VpnName       string `json:"vpnName"`
	UserId        int32  `json:"userId"`
	UserName      string `json:"userName"`
	InterfaceName string `json:"interfaceName"`
	ServerType    int32  `json:"serverType"` // 0: VPN Server, 1: VPN Client
	VpnType       int32  `json:"vpnType"`    // 0: L2TP, 1: PPTP, 2: IPSec, 3: OpenVPN
	RemoteIp      string `json:"remoteIp"`
	DownPkts      int64  `json:"downPkts"`
	DownBytes     int64  `json:"downBytes"`
	UpPkts        int64  `json:"upPkts"`
	UpBytes       int64  `json:"upBytes"`
	Uptime        string `json:"uptime"`
}

const path_OpenApiVpnTunnelStats = "/openapi/v1/{omadaID}/sites/{siteID}/setting/vpn/stats/tunnel"

func (t VpnTunnel) GetServerTypeString() string {
	switch t.ServerType {
	case 0:
		return "Server"
	case 1:
		return "Client"
	default:
		return "Unknown"
	}
}

func (t VpnTunnel) GetVpnTypeString() string {
	switch t.VpnType {
	case 0:
		return "L2TP"
	case 1:
		return "PPTP"
	case 2:
		return "IPSec"
	case 3:
		return "OpenVPN"
	default:
		return "Unknown"
	}
}
