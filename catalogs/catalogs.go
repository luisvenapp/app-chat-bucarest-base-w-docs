package catalogs

import (
	"os"

	"github.com/Venqis-NolaTech/campaing-app-core-go/pkg/config"
)

var (
	IsProd = os.Getenv("MODE") == "PROD"

	SpecialRoutes = struct {
		DebugRoute     string
		SwaggerRoute   string
		ProtosDownload string
	}{
		DebugRoute:     "/api/chat/debug",
		SwaggerRoute:   "/api/chat/swagger",
		ProtosDownload: "/api/chat/protos_download",
	}
)

func ClientAddress() string {
	return config.GetString("grpc.clientAddresses.chat-messages")
}
