package main

import (
	"embed"

	"log"

	"github.com/Venqis-NolaTech/campaing-app-chat-messages-api-go/catalogs"
	"github.com/Venqis-NolaTech/campaing-app-chat-messages-api-go/handlers"
	"github.com/Venqis-NolaTech/campaing-app-chat-messages-api-go/proto"
	"github.com/Venqis-NolaTech/campaing-app-core-go/pkg/config"
	"github.com/Venqis-NolaTech/campaing-app-core-go/pkg/server"
)

var (
	//go:embed proto
	protoFilesFs embed.FS

	address = config.GetString("server.address")
)

func main() {
	server.InitEnvironment()

	server.InitRedis()
	server.InitNats()

	srv := server.NewServer(
		server.WithProdMode(catalogs.IsProd),
		server.WithDebugRoute(catalogs.SpecialRoutes.DebugRoute),
		server.WithSwagger(catalogs.SpecialRoutes.SwaggerRoute, proto.SwaggerJsonDoc),
		server.WithServices(handlers.RegisterServicesFns),
		server.WithProtosDownload(protoFilesFs, catalogs.SpecialRoutes.ProtosDownload, "proto"),
	)

	log.Printf("Initializing gRPC server on address: %s\n", address)
	if err := srv.Listen(address); err != nil {
		log.Fatal(err)
	}
}
