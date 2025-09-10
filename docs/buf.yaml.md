# buf.yaml

Propósito
- Configuración de Buf (https://buf.build), que define el módulo/proyecto de Protobuf.

Contenido típico (orientativo)
- version: versión del schema de buf.
- deps: dependencias de otros módulos (por ejemplo googleapis, bufbuild protovalidate, etc.).
- build/directory: rutas donde buscar .proto.
- breaking/lint: reglas para linters y breaking change detection.

Cómo se usa en este repo
- Al ejecutar `buf generate`, `buf build`, o `buf lint`, Buf lee este archivo para saber dónde están los .proto (en proto/services/...) y cómo aplicar reglas.

Notas
- No es código ejecutable; es metadata para toolchain Protobuf.