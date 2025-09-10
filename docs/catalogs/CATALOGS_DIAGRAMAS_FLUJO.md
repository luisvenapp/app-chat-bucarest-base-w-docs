# Diagramas de flujo — catalogs

Este documento resume los flujos principales del paquete catalogs.

## Inicialización y rutas especiales
```mermaid
flowchart TD
    A[Inicio del paquete catalogs] --> B{Leer variable de entorno MODE}
    B -->|"PROD"| C[IsProd = true]
    B -->|cualquier otro valor| D[IsProd = false]
    A --> E[Definir SpecialRoutes]
    E --> E1[DebugRoute: /api/chat/debug]
    E --> E2[SwaggerRoute: /api/chat/swagger]
    E --> E3[ProtosDownload: /api/chat/protos_download]
```

## Obtención de dirección del cliente gRPC
```mermaid
flowchart TD
    A[Llamar ClientAddress()] --> B[config.GetString("grpc.clientAddresses.chat-messages")]
    B --> C[Retornar dirección del cliente]
```
