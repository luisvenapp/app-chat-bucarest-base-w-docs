# Documentación Técnica: database/database.go

## Descripción General

El archivo `database.go` es responsable de la inicialización y gestión de las conexiones a las bases de datos utilizadas por la aplicación. Implementa un patrón de inicialización automática mediante la función `init()` y proporciona acceso global a las instancias de base de datos PostgreSQL y Cassandra/ScyllaDB.

## Estructura del Archivo

### Importaciones

```go
import (
    "database/sql"
    "log"
    
    "github.com/Venqis-NolaTech/campaing-app-core-go/pkg/db/cassandra"
    "github.com/Venqis-NolaTech/campaing-app-core-go/pkg/db/postgres"
    "github.com/scylladb-solutions/gocql/v2"
)
```

**Análisis de Importaciones:**

- **`database/sql`**: Interfaz estándar de Go para bases de datos SQL
- **`log`**: Logging básico para errores críticos
- **`cassandra`**: Módulo del core para conexiones a Cassandra/ScyllaDB
- **`postgres`**: Módulo del core para conexiones a PostgreSQL (alias `dbpq`)
- **`gocql/v2`**: Driver oficial para Cassandra/ScyllaDB

### Variables Globales

```go
var db *sql.DB
var cassandraDB *gocql.Session
```

**Análisis de Variables:**

#### `db *sql.DB`
- **Propósito**: Instancia global de conexión a PostgreSQL
- **Tipo**: Puntero a `sql.DB` (interfaz estándar de Go)
- **Uso**: Base de datos principal para datos relacionales
- **Scope**: Accesible globalmente a través de la función `DB()`

#### `cassandraDB *gocql.Session`
- **Propósito**: Instancia global de sesión a Cassandra/ScyllaDB
- **Tipo**: Puntero a `gocql.Session`
- **Uso**: Base de datos NoSQL para mensajes y datos de alta velocidad
- **Scope**: Accesible globalmente a través de la función `CQLDB()`

### Funciones de Acceso

#### Función CQLDB

```go
func CQLDB() *gocql.Session {
    return cassandraDB
}
```

**Análisis Detallado:**

- **Propósito**: Proporciona acceso thread-safe a la sesión de Cassandra
- **Patrón**: Getter function para variable global
- **Retorno**: Puntero a la sesión activa de Cassandra
- **Uso típico**:
  ```go
  session := database.CQLDB()
  query := session.Query("SELECT * FROM messages WHERE room_id = ?", roomID)
  ```

#### Función DB

```go
func DB() *sql.DB {
    return db
}
```

**Análisis Detallado:**

- **Propósito**: Proporciona acceso thread-safe a la conexión de PostgreSQL
- **Patrón**: Getter function para variable global
- **Retorno**: Puntero a la instancia de base de datos SQL
- **Uso típico**:
  ```go
  database := database.DB()
  rows, err := database.Query("SELECT * FROM users WHERE id = $1", userID)
  ```

### Función de Inicialización

```go
func init() {
    instance, err := dbpq.ConnectToNewSQLInstance(dbpq.DefaultConnectionString)
    if err != nil {
        log.Fatal("ERROR CONNECTING TO DB: ", err)
    }
    db = instance
    
    cassandraDB_, err := cassandra.Connect(cassandra.DefaultConnectionConfig)
    if err != nil {
        log.Println("ERROR CONNECTING TO CASSANDRA: ", err)
        return
    }
    cassandraDB = cassandraDB_
}
```

**Análisis Paso a Paso:**

#### 1. Inicialización de PostgreSQL

```go
instance, err := dbpq.ConnectToNewSQLInstance(dbpq.DefaultConnectionString)
if err != nil {
    log.Fatal("ERROR CONNECTING TO DB: ", err)
}
db = instance
```

**Proceso de Conexión:**

- **`dbpq.ConnectToNewSQLInstance()`**: Crea nueva instancia de conexión
- **`dbpq.DefaultConnectionString`**: Utiliza string de conexión por defecto
- **Manejo de errores**: `log.Fatal()` termina la aplicación si falla
- **Asignación**: La instancia se asigna a la variable global `db`

**Configuración Típica del Connection String:**
```
postgres://username:password@localhost:5432/chat_db?sslmode=disable
```

**Características de la Conexión:**
- **Pool de conexiones**: `sql.DB` maneja automáticamente un pool
- **Thread-safe**: Seguro para uso concurrente
- **Reconexión automática**: Maneja desconexiones temporales
- **Configuración**: Timeouts, max connections, etc. desde el core

#### 2. Inicialización de Cassandra/ScyllaDB

```go
cassandraDB_, err := cassandra.Connect(cassandra.DefaultConnectionConfig)
if err != nil {
    log.Println("ERROR CONNECTING TO CASSANDRA: ", err)
    return
}
cassandraDB = cassandraDB_
```

**Proceso de Conexión:**

- **`cassandra.Connect()`**: Establece sesión con Cassandra
- **`cassandra.DefaultConnectionConfig`**: Configuración por defecto
- **Manejo de errores**: `log.Println()` registra error pero no termina la app
- **Graceful degradation**: La aplicación puede funcionar sin Cassandra
- **Asignación**: La sesión se asigna a la variable global `cassandraDB`

**Configuración Típica:**
```go
DefaultConnectionConfig = gocql.ClusterConfig{
    Hosts:    []string{"127.0.0.1"},
    Keyspace: "chat_keyspace",
    Port:     9042,
}
```

## Arquitectura de Bases de Datos

### PostgreSQL - Base de Datos Principal

**Uso en la Aplicación:**
- **Usuarios y autenticación**
- **Metadatos de salas de chat**
- **Configuraciones de usuario**
- **Relaciones entre entidades**
- **Datos transaccionales**

**Ventajas:**
- **ACID compliance**: Transacciones confiables
- **Relaciones complejas**: JOINs y foreign keys
- **Consistencia**: Datos críticos siempre consistentes
- **Madurez**: Ecosistema robusto y bien documentado

### Cassandra/ScyllaDB - Base de Datos de Mensajes

**Uso en la Aplicación:**
- **Mensajes de chat**: Almacenamiento de alto volumen
- **Historial de mensajes**: Datos time-series
- **Metadatos de mensajes**: Estados, reacciones, etc.
- **Datos de sesión temporal**

**Ventajas:**
- **Escalabilidad horizontal**: Maneja millones de mensajes
- **Alto rendimiento**: Escrituras y lecturas rápidas
- **Disponibilidad**: Tolerancia a fallos de nodos
- **Time-series**: Optimizado para datos temporales

## Patrones de Diseño Implementados

### 1. Singleton Pattern
- Una sola instancia de cada conexión de base de datos
- Acceso global controlado a través de funciones getter
- Inicialización única en tiempo de carga del paquete

### 2. Dependency Injection (Implícito)
- Las funciones `DB()` y `CQLDB()` actúan como inyectores de dependencias
- Los repositorios reciben las instancias a través de estas funciones
- Facilita testing con mocks

### 3. Fail-Fast Pattern (PostgreSQL)
- Si PostgreSQL falla, la aplicación no puede continuar
- `log.Fatal()` termina inmediatamente la ejecución
- Evita estados inconsistentes

### 4. Graceful Degradation (Cassandra)
- Si Cassandra falla, la aplicación continúa funcionando
- Funcionalidad reducida pero operativa
- Permite recuperación sin restart completo

## Consideraciones de Rendimiento

### Pool de Conexiones PostgreSQL
```go
// Configuración típica en el core
db.SetMaxOpenConns(25)
db.SetMaxIdleConns(5)
db.SetConnMaxLifetime(5 * time.Minute)
```

### Sesión Cassandra
```go
// Configuración típica en el core
cluster.NumConns = 2
cluster.Timeout = 10 * time.Second
cluster.ConnectTimeout = 10 * time.Second
```

## Manejo de Errores y Recuperación

### Estrategias Implementadas

1. **PostgreSQL - Crítico**:
   - Error de conexión → Aplicación termina
   - Permite restart rápido y limpio
   - Evita corrupción de datos

2. **Cassandra - No Crítico**:
   - Error de conexión → Log y continúa
   - Funcionalidad degradada
   - Posible reconexión posterior

### Monitoreo de Conexiones

```go
// Ejemplo de uso en handlers
func (h *handler) checkDBHealth() error {
    if err := database.DB().Ping(); err != nil {
        return fmt.Errorf("PostgreSQL unhealthy: %w", err)
    }
    
    if database.CQLDB() == nil {
        return fmt.Errorf("Cassandra unavailable")
    }
    
    return nil
}
```

## Seguridad

### 1. Configuración Externa
- Connection strings desde variables de entorno
- No hardcoding de credenciales
- Rotación de passwords sin recompilación

### 2. SSL/TLS
- Conexiones encriptadas en producción
- Validación de certificados
- Configuración por entorno

### 3. Principio de Menor Privilegio
- Usuarios de base de datos con permisos mínimos
- Separación de roles por funcionalidad
- Auditoría de accesos

## Testing y Mocking

### Estrategias para Testing

```go
// Mock para testing
type MockDB struct {
    *sql.DB
}

func (m *MockDB) Query(query string, args ...interface{}) (*sql.Rows, error) {
    // Implementación mock
}

// En tests
func TestWithMockDB(t *testing.T) {
    originalDB := db
    defer func() { db = originalDB }()
    
    db = &MockDB{}
    // Test logic
}
```

## Mejores Prácticas Implementadas

1. **Inicialización Automática**: Función `init()` para setup transparente
2. **Acceso Controlado**: Funciones getter en lugar de variables públicas
3. **Manejo Diferenciado de Errores**: Crítico vs no crítico
4. **Separación de Responsabilidades**: PostgreSQL para datos críticos, Cassandra para volumen
5. **Thread Safety**: Todas las operaciones son thread-safe
6. **Configuración Externa**: No hardcoding de configuraciones

Este archivo es fundamental para la persistencia de datos de la aplicación, proporcionando una base sólida y escalable para el almacenamiento de información crítica y de alto volumen.