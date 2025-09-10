# Documentación Técnica: buf.yaml

## Descripción General

El archivo `buf.yaml` es el archivo de configuración principal de Buf para el proyecto. Define la estructura de módulos, dependencias externas, reglas de linting y políticas de breaking changes para los archivos Protocol Buffers. Utiliza la versión 2 de la configuración de Buf, proporcionando un control granular sobre la calidad y compatibilidad del código Protocol Buffers.

## Estructura del Archivo

### Versión de Configuración

```yaml
# For details on buf.yaml configuration, visit https://buf.build/docs/configuration/v2/buf-yaml
version: v2
```

**Análisis:**
- **Versión**: v2 es la versión más reciente y recomendada
- **Documentación**: Referencia oficial para configuración detallada
- **Características**: Soporte completo para módulos, workspaces y dependencias

## Configuración de Módulos

```yaml
modules:
  - path: proto
```

**Análisis Detallado:**

### Definición de Módulo
- **Path**: `proto` - Directorio que contiene los archivos .proto del proyecto
- **Estructura**: Buf tratará este directorio como un módulo independiente
- **Organización**: Separación clara entre código fuente y dependencias

### Estructura del Módulo Proto

```
proto/
├── services/
│   ├── chat/
│   │   └── v1/
│   │       └── chat.proto
│   └── tokens/
│       └── v1/
│           └── tokens.proto
├── google/
│   └── type/
│       └── datetime.proto
└── buf.yaml (este archivo)
```

## Gestión de Dependencias

```yaml
deps:
  - buf.build/googleapis/googleapis
  - buf.build/bufbuild/protovalidate
```

**Análisis de Dependencias:**

### `buf.build/googleapis/googleapis`

**Propósito:**
- **APIs de Google**: Tipos comunes y anotaciones de Google
- **Incluye**: google.protobuf, google.api, google.rpc, etc.
- **Uso**: Annotations, field behavior, HTTP mappings

**Tipos Importantes:**
```protobuf
import "google/api/annotations.proto";
import "google/protobuf/timestamp.proto";
import "google/protobuf/empty.proto";
import "google/type/datetime.proto";
```

**Ejemplos de Uso:**
```protobuf
service ChatService {
  rpc GetRooms(GetRoomsRequest) returns (GetRoomsResponse) {
    option (google.api.http) = {
      get: "/v1/rooms"
    };
  }
}

message Room {
  google.protobuf.Timestamp created_at = 1;
  google.protobuf.Timestamp updated_at = 2;
}
```

### `buf.build/bufbuild/protovalidate`

**Propósito:**
- **Validación**: Reglas de validación declarativas para mensajes
- **Características**: Validación automática en runtime
- **Integración**: Compatible con múltiples lenguajes

**Ejemplos de Uso:**
```protobuf
import "buf/validate/validate.proto";

message CreateRoomRequest {
  string name = 1 [(buf.validate.field).string.min_len = 1];
  string type = 2 [(buf.validate.field).string = {
    in: ["p2p", "group"]
  }];
  repeated int32 participants = 3 [(buf.validate.field).repeated.min_items = 1];
}
```

## Configuración de Linting

```yaml
lint:
  use:
    - BASIC
    - ENUM_VALUE_PREFIX
    - ENUM_ZERO_VALUE_SUFFIX
    - FILE_LOWER_SNAKE_CASE
    - PACKAGE_VERSION_SUFFIX
    - SERVICE_SUFFIX

  ignore:
    - proto/google/type/datetime.proto
    - proto/google/protobuf/any.proto
```

**Análisis de Reglas de Linting:**

### Reglas Habilitadas

#### `BASIC`
**Incluye reglas fundamentales:**
- **DIRECTORY_SAME_AS_PACKAGE**: Directorio debe coincidir con package
- **PACKAGE_DEFINED**: Todo archivo debe tener package definido
- **PACKAGE_DIRECTORY_MATCH**: Package debe coincidir con estructura de directorios
- **PACKAGE_SAME_CSHARP_NAMESPACE**: Consistencia con C# namespaces
- **PACKAGE_SAME_GO_PACKAGE**: Consistencia con go_package
- **PACKAGE_SAME_JAVA_MULTIPLE_FILES**: Consistencia con Java
- **PACKAGE_SAME_JAVA_PACKAGE**: Consistencia con Java packages
- **PACKAGE_SAME_PHP_NAMESPACE**: Consistencia con PHP namespaces
- **PACKAGE_SAME_RUBY_PACKAGE**: Consistencia con Ruby packages
- **PACKAGE_SAME_SWIFT_PREFIX**: Consistencia con Swift prefixes

**Ejemplo de aplicación:**
```protobuf
// Archivo: proto/services/chat/v1/chat.proto
// Package correcto: chat.v1
package chat.v1;

option go_package = "github.com/example/proto/generated/services/chat/v1";
```

#### `ENUM_VALUE_PREFIX`
**Regla**: Valores de enum deben tener prefijo del nombre del enum

**Ejemplo correcto:**
```protobuf
enum MessageStatus {
  MESSAGE_STATUS_UNSPECIFIED = 0;
  MESSAGE_STATUS_SENDING = 1;
  MESSAGE_STATUS_SENT = 2;
  MESSAGE_STATUS_DELIVERED = 3;
  MESSAGE_STATUS_READ = 4;
}
```

**Ejemplo incorrecto:**
```protobuf
enum MessageStatus {
  UNSPECIFIED = 0;  // ❌ Falta prefijo MESSAGE_STATUS_
  SENDING = 1;      // ❌ Falta prefijo
}
```

#### `ENUM_ZERO_VALUE_SUFFIX`
**Regla**: Primer valor de enum debe terminar en `_UNSPECIFIED`

**Ejemplo correcto:**
```protobuf
enum RoomType {
  ROOM_TYPE_UNSPECIFIED = 0;  // ✅ Termina en _UNSPECIFIED
  ROOM_TYPE_P2P = 1;
  ROOM_TYPE_GROUP = 2;
}
```

#### `FILE_LOWER_SNAKE_CASE`
**Regla**: Nombres de archivos deben usar lower_snake_case

**Ejemplos correctos:**
- `chat_service.proto` ✅
- `message_types.proto` ✅
- `user_management.proto` ✅

**Ejemplos incorrectos:**
- `ChatService.proto` ❌
- `messageTypes.proto` ❌
- `user-management.proto` ❌

#### `PACKAGE_VERSION_SUFFIX`
**Regla**: Packages deben terminar con sufijo de versión

**Ejemplo correcto:**
```protobuf
package chat.v1;        // ✅ Termina en v1
package tokens.v2;      // ✅ Termina en v2
```

**Ejemplo incorrecto:**
```protobuf
package chat;           // ❌ Sin versión
package tokens.beta;    // ❌ No es vX
```

#### `SERVICE_SUFFIX`
**Regla**: Servicios deben terminar en "Service"

**Ejemplo correcto:**
```protobuf
service ChatService {     // ✅ Termina en Service
  // ...
}

service TokensService {   // ✅ Termina en Service
  // ...
}
```

**Ejemplo incorrecto:**
```protobuf
service Chat {           // ❌ No termina en Service
  // ...
}
```

### Archivos Ignorados

```yaml
ignore:
  - proto/google/type/datetime.proto
  - proto/google/protobuf/any.proto
```

**Razones para Ignorar:**
- **Archivos externos**: No controlamos estos archivos
- **Estándares diferentes**: Pueden no seguir nuestras reglas de linting
- **Compatibilidad**: Mantener compatibilidad con APIs de Google

## Configuración de Breaking Changes

```yaml
breaking:
  use:
    - FILE
```

**Análisis de Políticas de Breaking Changes:**

### Regla `FILE`
**Incluye verificaciones a nivel de archivo:**

#### **FILE_NO_DELETE**
- **Protección**: No se pueden eliminar archivos .proto
- **Impacto**: Eliminar archivos rompe compatibilidad
- **Excepción**: Solo en major versions

#### **FILE_SAME_PACKAGE**
- **Protección**: Package de archivo no puede cambiar
- **Impacto**: Cambiar package rompe imports
- **Migración**: Requiere coordinación con clientes

#### **FILE_SAME_SYNTAX**
- **Protección**: Syntax (proto2/proto3) no puede cambiar
- **Impacto**: Cambio de syntax afecta generación de código
- **Estabilidad**: Mantiene consistencia de API

#### **FILE_SAME_CC_ENABLE_ARENAS**
- **Protección**: Opciones de C++ no pueden cambiar
- **Impacto**: Afecta performance en C++
- **Especializado**: Solo relevante para clientes C++

#### **FILE_SAME_CC_GENERIC_SERVICES**
- **Protección**: Configuración de servicios C++
- **Impacto**: Afecta generación de código C++

#### **FILE_SAME_CSHARP_NAMESPACE**
- **Protección**: Namespace de C# no puede cambiar
- **Impacto**: Rompe imports en código C#

#### **FILE_SAME_GO_PACKAGE**
- **Protección**: go_package no puede cambiar
- **Impacto**: Rompe imports en código Go
- **Crítico**: Muy importante para este proyecto Go

#### **FILE_SAME_JAVA_MULTIPLE_FILES**
- **Protección**: Configuración de archivos Java
- **Impacto**: Afecta estructura de clases Java

#### **FILE_SAME_JAVA_OUTER_CLASSNAME**
- **Protección**: Nombre de clase externa Java
- **Impacto**: Rompe referencias en Java

#### **FILE_SAME_JAVA_PACKAGE**
- **Protección**: Package Java no puede cambiar
- **Impacto**: Rompe imports en Java

#### **FILE_SAME_JAVA_STRING_CHECK_UTF8**
- **Protección**: Validación UTF-8 en Java
- **Impacto**: Afecta validación de strings

#### **FILE_SAME_OBJC_CLASS_PREFIX**
- **Protección**: Prefijo de clases Objective-C
- **Impacto**: Rompe código Objective-C/Swift

#### **FILE_SAME_OPTIMIZE_FOR**
- **Protección**: Optimización no puede cambiar
- **Impacto**: Afecta performance y tamaño

#### **FILE_SAME_PHP_CLASS_PREFIX**
- **Protección**: Prefijo de clases PHP
- **Impacto**: Rompe código PHP

#### **FILE_SAME_PHP_METADATA_NAMESPACE**
- **Protección**: Namespace de metadata PHP
- **Impacto**: Afecta generación PHP

#### **FILE_SAME_PHP_NAMESPACE**
- **Protección**: Namespace PHP no puede cambiar
- **Impacto**: Rompe imports PHP

#### **FILE_SAME_RUBY_PACKAGE**
- **Protección**: Package Ruby no puede cambiar
- **Impacto**: Rompe código Ruby

#### **FILE_SAME_SWIFT_PREFIX**
- **Protección**: Prefijo Swift no puede cambiar
- **Impacto**: Rompe código Swift

## Comandos de Buf

### Linting

```bash
# Ejecutar linting
buf lint

# Linting con formato específico
buf lint --format=json

# Linting de archivos específicos
buf lint proto/services/chat/v1/chat.proto
```

### Breaking Change Detection

```bash
# Comparar con main branch
buf breaking --against '.git#branch=main'

# Comparar con tag específico
buf breaking --against '.git#tag=v1.0.0'

# Comparar con commit específico
buf breaking --against '.git#commit=abc123'

# Comparar con archivo local
buf breaking --against 'proto-backup'
```

### Gestión de Dependencias

```bash
# Actualizar dependencias
buf mod update

# Verificar dependencias
buf mod verify

# Limpiar cache de dependencias
buf mod clear-cache
```

## Integración con CI/CD

### GitHub Actions

```yaml
name: Protocol Buffers CI
on: [push, pull_request]

jobs:
  proto-lint:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
        with:
          fetch-depth: 0  # Necesario para breaking changes
      
      - uses: bufbuild/buf-setup-action@v1
        with:
          version: '1.28.1'
      
      - name: Lint Protocol Buffers
        run: buf lint
      
      - name: Check for breaking changes
        if: github.event_name == 'pull_request'
        run: buf breaking --against 'https://github.com/${{ github.repository }}.git#branch=${{ github.base_ref }}'
```

### Pre-commit Hooks

```yaml
# .pre-commit-config.yaml
repos:
  - repo: https://github.com/bufbuild/buf
    rev: v1.28.1
    hooks:
      - id: buf-lint
      - id: buf-breaking
        args: ['--against', '.git#branch=main']
```

## Configuración Avanzada

### Workspace Configuration

```yaml
# buf.work.yaml (para múltiples módulos)
version: v2
directories:
  - proto
  - proto-external
  - proto-internal
```

### Configuración por Entorno

```yaml
# buf.dev.yaml (desarrollo)
version: v2
modules:
  - path: proto
lint:
  use:
    - BASIC
  # Reglas más permisivas para desarrollo

# buf.prod.yaml (producción)
version: v2
modules:
  - path: proto
lint:
  use:
    - BASIC
    - ENUM_VALUE_PREFIX
    - ENUM_ZERO_VALUE_SUFFIX
    - FILE_LOWER_SNAKE_CASE
    - PACKAGE_VERSION_SUFFIX
    - SERVICE_SUFFIX
  # Reglas estrictas para producción
```

## Mejores Prácticas Implementadas

### 1. **Linting Comprehensivo**
- Reglas básicas para consistencia
- Convenciones de naming específicas
- Validación de estructura de archivos

### 2. **Breaking Change Protection**
- Protección a nivel de archivo
- Verificación automática en CI
- Prevención de cambios incompatibles

### 3. **Gestión de Dependencias**
- Dependencias versionadas
- APIs estándar de Google
- Validación automática

### 4. **Organización Modular**
- Estructura clara de módulos
- Separación de responsabilidades
- Escalabilidad para múltiples servicios

### 5. **Configuración Versionada**
- Configuración en control de versiones
- Reproducibilidad entre entornos
- Evolución controlada de reglas

## Troubleshooting

### Problemas Comunes

#### 1. **Errores de Linting**
```bash
# Ver detalles del error
buf lint --error-format=json

# Ignorar temporalmente
buf lint --disable-symlinks
```

#### 2. **Breaking Changes Falsos Positivos**
```bash
# Verificar diferencias específicas
buf breaking --against '.git#branch=main' --format=json

# Excluir archivos específicos
buf breaking --against '.git#branch=main' --exclude-path proto/experimental
```

#### 3. **Dependencias No Encontradas**
```bash
# Limpiar y actualizar
buf mod clear-cache
buf mod update
```

### Debugging

```bash
# Modo verbose
buf lint --verbose

# Debug de configuración
buf config ls-files

# Verificar módulos
buf ls-files
```

## Monitoreo y Mantenimiento

### Métricas de Calidad

```bash
# Contar violaciones de linting
buf lint --format=json | jq '.issues | length'

# Verificar cobertura de reglas
buf lint --list-rules
```

### Actualizaciones

```bash
# Verificar nuevas versiones de dependencias
buf registry deps list

# Actualizar a versión específica
# Editar buf.yaml manualmente
```

Este archivo `buf.yaml` establece una base sólida para la gestión de Protocol Buffers, asegurando calidad, consistencia y compatibilidad en el desarrollo de APIs gRPC.