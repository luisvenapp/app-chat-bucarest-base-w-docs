# proto/embed.go

Propósito
- Empaqueta el archivo OpenAPI generado (generated/openapi.yaml) dentro del binario mediante go:embed.

Uso
- La variable pública `SwaggerJsonDoc []byte` queda disponible para servir documentación Swagger/OpenAPI sin depender del sistema de archivos en producción.