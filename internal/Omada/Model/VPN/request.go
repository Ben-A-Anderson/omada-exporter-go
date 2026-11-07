package VPN

import (
	"omada_exporter_go/internal/Log"
	"omada_exporter_go/internal/Omada/HttpClient/ApiClient"
)

func Get() (*[]VpnTunnel, error) {
	Log.Debug("Fetching VPN tunnel stats")

	client := ApiClient.GetInstance()

	result, err := ApiClient.Get[VpnTunnel](
		client,
		path_OpenApiVpnTunnelStats,
		map[string]string{},
		map[string]string{},
		true, // usePagination
	)
	if err != nil {
		return nil, Log.Error(err, "Failed to get VPN tunnel stats")
	}

	Log.Debug("Fetched %d VPN tunnels", len(*result))

	return result, nil
}
