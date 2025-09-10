# Documentación Técnica: repository/rooms/helpers.go

## Descripción General

El archivo `helpers.go` contiene funciones utilitarias específicas del dominio de salas de chat. Proporciona funcionalidades auxiliares para el procesamiento de datos, incluyendo ordenamiento de IDs de usuarios y normalización de texto para búsquedas y comparaciones.

## Estructura del Archivo

### Importaciones

```go
import (
    "unicode"
    
    "golang.org/x/text/runes"
    "golang.org/x/text/transform"
    "golang.org/x/text/unicode/norm"
)
```

**Análisis de Importaciones:**

- **`unicode`**: Paquete estándar para clasificación de caracteres Unicode
- **`golang.org/x/text/runes`**: Utilidades para manipulación de runas
- **`golang.org/x/text/transform`**: Framework para transformaciones de texto
- **`golang.org/x/text/unicode/norm`**: Normalización Unicode (NFD, NFC, etc.)

## Función sortUserIDs

```go
func sortUserIDs(id1, id2 int) (int, int) {
    if id1 < id2 {
        return id1, id2
    }
    return id2, id1
}
```

**Análisis Detallado:**

### Propósito de la Función
- **Ordenamiento determinístico**: Asegura orden consistente de IDs de usuarios
- **Normalización**: Crea una representación canónica de pares de usuarios
- **Uso principal**: Generación de claves para salas P2P (persona-a-persona)

### Lógica de Implementación

#### Comparación Simple
```go
if id1 < id2 {
    return id1, id2
}
return id2, id1
```

**Proceso:**
1. **Comparación**: Evalúa si `id1` es menor que `id2`
2. **Orden ascendente**: Si es menor, retorna en orden original
3. **Orden inverso**: Si es mayor o igual, retorna en orden inverso
4. **Resultado**: Siempre retorna el menor ID primero

### Casos de Uso

#### 1. Generación de Claves para Salas P2P

```go
func generateP2PRoomKey(userID1, userID2 int) string {
    smaller, larger := sortUserIDs(userID1, userID2)
    return fmt.Sprintf("p2p_%d_%d", smaller, larger)
}

// Ejemplos:
// generateP2PRoomKey(123, 456) → "p2p_123_456"
// generateP2PRoomKey(456, 123) → "p2p_123_456" (mismo resultado)
```

#### 2. Búsqueda de Salas Existentes

```go
func findExistingP2PRoom(ctx context.Context, userID1, userID2 int) (*Room, error) {
    smaller, larger := sortUserIDs(userID1, userID2)
    
    query := `
        SELECT room_id FROM p2p_rooms 
        WHERE user1_id = $1 AND user2_id = $2
    `
    
    var roomID string
    err := db.QueryRowContext(ctx, query, smaller, larger).Scan(&roomID)
    if err != nil {
        return nil, err
    }
    
    return getRoomByID(ctx, roomID)
}
```

#### 3. Prevención de Salas Duplicadas

```go
func createP2PRoom(ctx context.Context, userID1, userID2 int) (*Room, error) {
    smaller, larger := sortUserIDs(userID1, userID2)
    
    // Verificar si ya existe
    existing, err := findExistingP2PRoom(ctx, smaller, larger)
    if err == nil && existing != nil {
        return existing, nil // Retornar sala existente
    }
    
    // Crear nueva sala
    return createNewP2PRoom(ctx, smaller, larger)
}
```

### Ventajas del Approach

#### 1. **Consistencia**
- **Determinístico**: Mismo input siempre produce mismo output
- **Independiente del orden**: `sortUserIDs(A, B) == sortUserIDs(B, A)`
- **Canónico**: Una sola representación para cada par de usuarios

#### 2. **Performance**
- **O(1)**: Complejidad constante
- **Sin allocations**: No crea nuevas estructuras de datos
- **Minimal overhead**: Solo una comparación

#### 3. **Simplicidad**
- **Fácil de entender**: Lógica clara y directa
- **Fácil de testear**: Casos de prueba simples
- **Sin dependencias**: No requiere librerías externas

## Función removeAccents

```go
func removeAccents(s string) (string, error) {
    t := transform.Chain(norm.NFD, runes.Remove(runes.In(unicode.Mn)), norm.NFC)
    result, _, err := transform.String(t, s)
    return result, err
}
```

**Análisis Detallado:**

### Propósito de la Función
- **Normalización de texto**: Elimina acentos y diacríticos de strings
- **Búsqueda mejorada**: Facilita búsquedas insensibles a acentos
- **Comparación de texto**: Permite comparaciones normalizadas

### Proceso de Transformación

#### 1. Construcción de la Cadena de Transformación
```go
t := transform.Chain(norm.NFD, runes.Remove(runes.In(unicode.Mn)), norm.NFC)
```

**Análisis de cada paso:**

##### `norm.NFD` (Normalization Form Decomposed)
- **Propósito**: Descompone caracteres acentuados en caracteres base + diacríticos
- **Ejemplo**: `"café"` → `"cafe" + "́"` (e + acute accent)
- **Unicode**: Convierte a forma canónica descompuesta

##### `runes.Remove(runes.In(unicode.Mn))`
- **`unicode.Mn`**: Categoría Unicode "Mark, Nonspacing" (diacríticos)
- **`runes.In(unicode.Mn)`**: Predicate que identifica diacríticos
- **`runes.Remove(...)`**: Elimina runas que coinciden con el predicate
- **Resultado**: Solo quedan caracteres base, sin diacríticos

##### `norm.NFC` (Normalization Form Composed)
- **Propósito**: Recompone caracteres en forma canónica
- **Resultado**: String normalizado sin acentos
- **Consistencia**: Asegura representación estándar

#### 2. Aplicación de la Transformación
```go
result, _, err := transform.String(t, s)
```

- **Input**: String original con posibles acentos
- **Transform**: Aplica la cadena de transformaciones
- **Output**: String sin acentos, bytes transformados, error
- **Return**: Solo string resultado y error

### Ejemplos de Transformación

```go
// Ejemplos de uso
removeAccents("café")      // → "cafe", nil
removeAccents("niño")      // → "nino", nil
removeAccents("résumé")    // → "resume", nil
removeAccents("naïve")     // → "naive", nil
removeAccents("Zürich")    // → "Zurich", nil
removeAccents("São Paulo") // → "Sao Paulo", nil
```

### Casos de Uso en el Sistema de Chat

#### 1. Búsqueda de Usuarios
```go
func searchUsers(ctx context.Context, query string) ([]*User, error) {
    normalizedQuery, err := removeAccents(strings.ToLower(query))
    if err != nil {
        return nil, err
    }
    
    sql := `
        SELECT id, name, phone FROM users 
        WHERE LOWER(remove_accents(name)) LIKE $1
    `
    
    return queryUsers(ctx, sql, "%"+normalizedQuery+"%")
}
```

#### 2. Comparación de Nombres
```go
func compareNames(name1, name2 string) (bool, error) {
    norm1, err := removeAccents(strings.ToLower(name1))
    if err != nil {
        return false, err
    }
    
    norm2, err := removeAccents(strings.ToLower(name2))
    if err != nil {
        return false, err
    }
    
    return norm1 == norm2, nil
}
```

#### 3. Indexación para Búsqueda
```go
func createSearchIndex(ctx context.Context, user *User) error {
    normalizedName, err := removeAccents(strings.ToLower(user.Name))
    if err != nil {
        return err
    }
    
    query := `
        INSERT INTO user_search_index (user_id, normalized_name)
        VALUES ($1, $2)
        ON CONFLICT (user_id) DO UPDATE SET normalized_name = EXCLUDED.normalized_name
    `
    
    _, err = db.ExecContext(ctx, query, user.ID, normalizedName)
    return err
}
```

### Consideraciones de Performance

#### 1. **Caching de Resultados**
```go
var accentCache = make(map[string]string)
var accentCacheMutex sync.RWMutex

func removeAccentsCached(s string) (string, error) {
    accentCacheMutex.RLock()
    if result, exists := accentCache[s]; exists {
        accentCacheMutex.RUnlock()
        return result, nil
    }
    accentCacheMutex.RUnlock()
    
    result, err := removeAccents(s)
    if err != nil {
        return "", err
    }
    
    accentCacheMutex.Lock()
    accentCache[s] = result
    accentCacheMutex.Unlock()
    
    return result, nil
}
```

#### 2. **Precomputación en Base de Datos**
```sql
-- Agregar columna normalizada
ALTER TABLE users ADD COLUMN normalized_name TEXT;

-- Trigger para mantener sincronizada
CREATE OR REPLACE FUNCTION update_normalized_name()
RETURNS TRIGGER AS $$
BEGIN
    NEW.normalized_name = remove_accents(LOWER(NEW.name));
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER users_normalize_name
    BEFORE INSERT OR UPDATE ON users
    FOR EACH ROW
    EXECUTE FUNCTION update_normalized_name();
```

## Testing

### Unit Tests para sortUserIDs

```go
func TestSortUserIDs(t *testing.T) {
    tests := []struct {
        name     string
        id1, id2 int
        want1, want2 int
    }{
        {
            name: "first smaller",
            id1: 123, id2: 456,
            want1: 123, want2: 456,
        },
        {
            name: "second smaller", 
            id1: 456, id2: 123,
            want1: 123, want2: 456,
        },
        {
            name: "equal IDs",
            id1: 123, id2: 123,
            want1: 123, want2: 123,
        },
        {
            name: "negative IDs",
            id1: -1, id2: -5,
            want1: -5, want2: -1,
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got1, got2 := sortUserIDs(tt.id1, tt.id2)
            assert.Equal(t, tt.want1, got1)
            assert.Equal(t, tt.want2, got2)
        })
    }
}
```

### Unit Tests para removeAccents

```go
func TestRemoveAccents(t *testing.T) {
    tests := []struct {
        name     string
        input    string
        expected string
        hasError bool
    }{
        {
            name: "spanish accents",
            input: "café niño",
            expected: "cafe nino",
            hasError: false,
        },
        {
            name: "french accents",
            input: "résumé naïve",
            expected: "resume naive", 
            hasError: false,
        },
        {
            name: "german umlauts",
            input: "Zürich Müller",
            expected: "Zurich Muller",
            hasError: false,
        },
        {
            name: "no accents",
            input: "hello world",
            expected: "hello world",
            hasError: false,
        },
        {
            name: "empty string",
            input: "",
            expected: "",
            hasError: false,
        },
        {
            name: "mixed languages",
            input: "São Paulo café",
            expected: "Sao Paulo cafe",
            hasError: false,
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result, err := removeAccents(tt.input)
            
            if tt.hasError {
                assert.Error(t, err)
            } else {
                assert.NoError(t, err)
                assert.Equal(t, tt.expected, result)
            }
        })
    }
}
```

### Benchmark Tests

```go
func BenchmarkSortUserIDs(b *testing.B) {
    for i := 0; i < b.N; i++ {
        sortUserIDs(123, 456)
    }
}

func BenchmarkRemoveAccents(b *testing.B) {
    testString := "café niño résumé"
    b.ResetTimer()
    
    for i := 0; i < b.N; i++ {
        removeAccents(testString)
    }
}

func BenchmarkRemoveAccentsLong(b *testing.B) {
    testString := strings.Repeat("café niño résumé ", 100)
    b.ResetTimer()
    
    for i := 0; i < b.N; i++ {
        removeAccents(testString)
    }
}
```

## Consideraciones de Internacionalización

### Soporte para Diferentes Idiomas

#### 1. **Idiomas Soportados**
- **Español**: café → cafe, niño → nino
- **Francés**: résumé → resume, naïve → naive  
- **Alemán**: Zürich → Zurich, Müller → Muller
- **Portugués**: São Paulo → Sao Paulo
- **Italiano**: città → citta

#### 2. **Limitaciones**
- **Caracteres no latinos**: No maneja caracteres árabes, chinos, etc.
- **Ligaduras**: No procesa ligaduras como æ, œ
- **Casos especiales**: Algunos caracteres pueden requerir tratamiento especial

### Extensiones Futuras

```go
func normalizeForSearch(s string) (string, error) {
    // 1. Convertir a minúsculas
    s = strings.ToLower(s)
    
    // 2. Remover acentos
    s, err := removeAccents(s)
    if err != nil {
        return "", err
    }
    
    // 3. Remover espacios extra
    s = strings.TrimSpace(s)
    s = regexp.MustCompile(`\s+`).ReplaceAllString(s, " ")
    
    // 4. Remover caracteres especiales
    s = regexp.MustCompile(`[^\p{L}\p{N}\s]`).ReplaceAllString(s, "")
    
    return s, nil
}
```

## Mejores Prácticas Implementadas

1. **Single Responsibility**: Cada función tiene una responsabilidad específica
2. **Error Handling**: Manejo explícito de errores en transformaciones
3. **Unicode Compliance**: Uso correcto de estándares Unicode
4. **Performance**: Implementaciones eficientes para operaciones frecuentes
5. **Testability**: Funciones puras fáciles de testear
6. **Internationalization**: Soporte para múltiples idiomas con acentos

Este archivo proporciona utilidades fundamentales para el procesamiento de datos en el sistema de chat, especialmente importantes para la gestión de salas P2P y la búsqueda de usuarios en entornos multiidioma.