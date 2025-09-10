# Documentación Técnica: cmd/campaing-app-chat-cli/main.go

## Descripción General

El archivo `main.go` del CLI implementa una aplicación de chat interactiva en terminal utilizando la biblioteca Bubble Tea. Proporciona una interfaz de usuario completa para interactuar con el servicio de chat, incluyendo autenticación, gestión de salas, envío de mensajes y streaming en tiempo real.

## Estructura del Archivo

### Importaciones

```go
import (
    "context"
    "flag"
    "fmt"
    "log"
    "os"
    "strconv"
    "strings"
    "time"
    
    authv1 "github.com/Venqis-NolaTech/campaing-app-auth-api-go/proto/generated/services/auth/v1"
    authv1client "github.com/Venqis-NolaTech/campaing-app-auth-api-go/proto/generated/services/auth/v1/client"
    chatv1 "github.com/Venqis-NolaTech/campaing-app-chat-messages-api-go/proto/generated/services/chat/v1"
    chatv1client "github.com/Venqis-NolaTech/campaing-app-chat-messages-api-go/proto/generated/services/chat/v1/client"
    "github.com/Venqis-NolaTech/campaing-app-chat-messages-api-go/utils"
    "github.com/Venqis-NolaTech/campaing-app-core-go/pkg/api"
    "github.com/Venqis-NolaTech/campaing-app-core-go/pkg/api/auth"
    "github.com/charmbracelet/bubbles/textarea"
    "github.com/charmbracelet/bubbles/viewport"
    tea "github.com/charmbracelet/bubbletea"
    "github.com/charmbracelet/lipgloss"
    "github.com/google/uuid"
    "google.golang.org/protobuf/encoding/protojson"
)
```

**Análisis de Importaciones:**

- **Estándar Go**: `context`, `flag`, `fmt`, `log`, `os`, `strconv`, `strings`, `time`
- **Servicios gRPC**: Clientes para auth y chat
- **Utilidades internas**: `utils` para encriptación
- **Core**: API y autenticación
- **UI Terminal**: Bubble Tea ecosystem para TUI
- **Utilidades**: UUID, protojson

### Variables Globales

```go
var generalParams api.GeneralParams

var userID = flag.Int("u", 0, "user id")
var remitentID = flag.Int("t", 0, "remitent id")
var roomID = flag.Int("r", 0, "room id")

var room *chatv1.Room
var me *authv1.MeResponse
```

**Análisis de Variables:**

#### Parámetros de Línea de Comandos
- **`userID`**: ID del usuario que ejecuta el CLI
- **`remitentID`**: ID del destinatario para crear chat P2P
- **`roomID`**: ID de sala existente para unirse

#### Variables de Estado Global
- **`generalParams`**: Parámetros de API (token, idioma, plataforma)
- **`room`**: Información de la sala de chat actual
- **`me`**: Información del usuario autenticado

## Función Principal

```go
func main() {
    flag.Parse()
    
    if userID == nil || *userID == 0 {
        log.Fatal("el usuario es un campo requerido")
    }
    
    token, err := auth.GenerateSessionToken(auth.SessionData{
        UserID: *userID,
        Type:   "ACCESS",
    })
    
    session, _ := auth.ValidateSessionToken(token)
    
    location, err := time.LoadLocation("Local")
    if err != nil {
        log.Fatal("Can't get timezone: ", err)
        return
    }
    
    generalParams = api.GeneralParams{
        SessionToken: token,
        Lang:         "es",
        Platform:     "cli",
        IANATimezone: location.String(),
        Session:      session,
        ClientId:     uuid.NewString(),
    }
    
    res, err := authv1client.Me(context.Background(), generalParams, &authv1.MeRequest{})
    if err != nil {
        log.Fatal("Can't get user data: ", err)
    }
    me = res
    
    if roomID == nil || *roomID == 0 {
        if remitentID == nil || *remitentID == 0 {
            log.Fatal("el campo de remitente es requerido")
        }
        res, err := chatv1client.CreateRoom(context.Background(), generalParams, &chatv1.CreateRoomRequest{
            Type: "p2p",
            Participants: []int32{
                int32(*remitentID),
            },
        })
        if err != nil {
            log.Fatal("Can't create room: ", err)
        }
        room = res.Room
    } else {
        res, err := chatv1client.GetRoom(context.Background(), generalParams, &chatv1.GetRoomRequest{
            Id: strconv.Itoa(*roomID),
        })
        if err != nil {
            log.Fatal("Can't join room: ", err)
        }
        room = res.Room
    }
    
    log.Println("Conectando al room:", room.Id)
    
    // Crea el directorio para el log si no existe
    if err := os.MkdirAll("tmp", 0755); err != nil {
        log.Fatal(err)
    }
    
    // Configura un archivo de log para depuración
    f, err := tea.LogToFile("tmp/chat-debug.log", "debug")
    if err != nil {
        log.Fatal(err)
    }
    defer f.Close()
    
    // Inicia el programa de Bubble Tea
    p := tea.NewProgram(initialModel(), tea.WithAltScreen())
    
    if _, err := p.Run(); err != nil {
        log.Fatal(err)
    }
}
```

**Análisis Paso a Paso:**

### 1. Validación de Parámetros
```go
flag.Parse()

if userID == nil || *userID == 0 {
    log.Fatal("el usuario es un campo requerido")
}
```
- **Parsing**: Procesa argumentos de línea de comandos
- **Validación**: Asegura que se proporcione un userID válido
- **Error handling**: Termina la aplicación si falta información crítica

### 2. Generación de Token de Sesión
```go
token, err := auth.GenerateSessionToken(auth.SessionData{
    UserID: *userID,
    Type:   "ACCESS",
})

session, _ := auth.ValidateSessionToken(token)
```
- **Autenticación**: Genera token JWT para el usuario
- **Tipo**: Token de acceso para operaciones normales
- **Validación**: Verifica que el token generado sea válido

### 3. Configuración de Parámetros Generales
```go
location, err := time.LoadLocation("Local")
if err != nil {
    log.Fatal("Can't get timezone: ", err)
    return
}

generalParams = api.GeneralParams{
    SessionToken: token,
    Lang:         "es",
    Platform:     "cli",
    IANATimezone: location.String(),
    Session:      session,
    ClientId:     uuid.NewString(),
}
```

**Parámetros Configurados:**
- **SessionToken**: Token JWT para autenticación
- **Lang**: Idioma español por defecto
- **Platform**: Identificador de plataforma CLI
- **IANATimezone**: Zona horaria local del usuario
- **Session**: Datos de sesión validados
- **ClientId**: UUID único para esta instancia del CLI

### 4. Obtención de Información del Usuario
```go
res, err := authv1client.Me(context.Background(), generalParams, &authv1.MeRequest{})
if err != nil {
    log.Fatal("Can't get user data: ", err)
}
me = res
```
- **Endpoint**: Llama al servicio de auth para obtener datos del usuario
- **Información**: Nombre, teléfono, avatar, etc.
- **Uso**: Mostrar información del usuario en la interfaz

### 5. Gestión de Salas de Chat

#### Creación de Nueva Sala P2P
```go
if roomID == nil || *roomID == 0 {
    if remitentID == nil || *remitentID == 0 {
        log.Fatal("el campo de remitente es requerido")
    }
    res, err := chatv1client.CreateRoom(context.Background(), generalParams, &chatv1.CreateRoomRequest{
        Type: "p2p",
        Participants: []int32{
            int32(*remitentID),
        },
    })
    if err != nil {
        log.Fatal("Can't create room: ", err)
    }
    room = res.Room
}
```

**Lógica:**
- **Condición**: Si no se especifica roomID
- **Validación**: Requiere remitentID para crear chat P2P
- **Creación**: Llama al servicio para crear nueva sala
- **Participantes**: Incluye al remitente especificado

#### Unión a Sala Existente
```go
else {
    res, err := chatv1client.GetRoom(context.Background(), generalParams, &chatv1.GetRoomRequest{
        Id: strconv.Itoa(*roomID),
    })
    if err != nil {
        log.Fatal("Can't join room: ", err)
    }
    room = res.Room
}
```

**Lógica:**
- **Condición**: Si se especifica roomID
- **Obtención**: Recupera información de la sala existente
- **Validación**: Verifica que el usuario tenga acceso a la sala

### 6. Configuración de Logging
```go
if err := os.MkdirAll("tmp", 0755); err != nil {
    log.Fatal(err)
}

f, err := tea.LogToFile("tmp/chat-debug.log", "debug")
if err != nil {
    log.Fatal(err)
}
defer f.Close()
```
- **Directorio**: Crea directorio tmp para logs
- **Archivo**: Configura logging de Bubble Tea
- **Nivel**: Debug para información detallada
- **Cleanup**: Cierra archivo al terminar

### 7. Inicialización de la Interfaz
```go
p := tea.NewProgram(initialModel(), tea.WithAltScreen())

if _, err := p.Run(); err != nil {
    log.Fatal(err)
}
```
- **Programa**: Crea instancia de Bubble Tea
- **Modelo**: Inicializa con modelo inicial
- **Alt Screen**: Usa pantalla alternativa del terminal
- **Ejecución**: Inicia el loop principal de la aplicación

## Tipos de Datos para Bubble Tea

### Tipos de Mensajes

```go
// message es un tipo para los mensajes recibidos del canal de escucha.
// Usar un tipo propio nos permite distinguirlo de otros strings.
type message string

// errMessage se usa para comunicar errores al ciclo de Update.
type errMessage error
```

**Propósito:**
- **Type Safety**: Distingue entre diferentes tipos de mensajes
- **Bubble Tea Pattern**: Sigue el patrón de mensajes tipados
- **Error Handling**: Manejo específico de errores en la UI

### Estado de Sesión

```go
// sessionState contiene el estado de nuestra sesión de chat.
type sessionState struct {
    ctx         context.Context
    cancel      context.CancelFunc
    messages    []string
    messageChan chan message
    err         error
}
```

**Campos:**
- **ctx/cancel**: Control de contexto para cancelación
- **messages**: Historial de mensajes mostrados
- **messageChan**: Canal para recibir nuevos mensajes
- **err**: Estado de error de la sesión

### Modelo Principal

```go
type model struct {
    state       sessionState
    viewport    viewport.Model
    textarea    textarea.Model
    senderStyle lipgloss.Style
}
```

**Componentes:**
- **state**: Estado de la sesión de chat
- **viewport**: Área de visualización de mensajes
- **textarea**: Área de entrada de texto
- **senderStyle**: Estilo para mensajes del usuario

## Función de Inicialización del Modelo

```go
func initialModel() model {
    ta := textarea.New()
    ta.Placeholder = "Escribe tu mensaje y presiona Enter..."
    ta.Focus()
    
    ta.Prompt = "┃ "
    ta.CharLimit = 280
    ta.SetWidth(80)
    ta.SetHeight(3)
    ta.FocusedStyle.CursorLine = lipgloss.NewStyle()
    ta.ShowLineNumbers = false
    ta.KeyMap.InsertNewline.SetEnabled(false) // Deshabilita el salto de línea con Enter
    
    vp := viewport.New(80, 30)
    vp.SetContent(fmt.Sprintf("Estás en la sala %s\nPresiona Ctrl+C para salir.\n\n Información del usuario: %s", room.Id, protojson.Format(me)))
    
    // Estado inicial de la sesión
    ctx, cancel := context.WithCancel(context.Background())
    initialState := sessionState{
        ctx:         ctx,
        cancel:      cancel,
        messages:    []string{},
        messageChan: make(chan message),
    }
    
    return model{
        state:       initialState,
        viewport:    vp,
        textarea:    ta,
        senderStyle: lipgloss.NewStyle().Foreground(lipgloss.Color("5")),
    }
}
```

**Configuración del TextArea:**
- **Placeholder**: Texto de ayuda
- **Focus**: Activo por defecto
- **Prompt**: Indicador visual "┃ "
- **Límites**: 280 caracteres, 80x3 dimensiones
- **Comportamiento**: Enter envía mensaje (no nueva línea)

**Configuración del Viewport:**
- **Dimensiones**: 80x30 caracteres
- **Contenido inicial**: Información de la sala y usuario
- **Formato**: JSON pretty-printed para debugging

## Funciones de Streaming

### Listener de Mensajes

```go
func listenForMessages(ctx context.Context, msgChan chan<- message) {
    req, _ := api.NewRequest(generalParams, &chatv1.StreamMessagesRequest{})
    client := chatv1client.GetChatServiceClient()
    
    stream, err := client.StreamMessages(ctx, req)
    if err != nil {
        log.Fatal("Can't stream messages: ", err)
    }
    defer stream.Close()
    
    for stream.Receive() {
        if err := stream.Err(); err != nil {
            log.Println("Error with stream:", err)
            continue
        }
        log.Println("Event received for user:", me.Id, protojson.Format(stream.Msg()))
        
        msg := stream.Msg().GetMessage()
        if msg != nil && msg.RoomId == room.Id && uint32(msg.GetSenderId()) != me.Id {
            contact := msg.GetContactName()
            content, err := utils.DecryptMessage(msg.Content, room.EncryptionData)
            if err != nil {
                log.Println("Error decripting message:", err)
            }
            msgChan <- message(fmt.Sprintf("%s: %s", contact, content))
        }
    }
    
    log.Println("Cerrando el listener de mensajes.")
}
```

**Análisis del Streaming:**

#### 1. Configuración del Stream
```go
req, _ := api.NewRequest(generalParams, &chatv1.StreamMessagesRequest{})
client := chatv1client.GetChatServiceClient()
stream, err := client.StreamMessages(ctx, req)
```
- **Request**: Crea request con parámetros de autenticación
- **Cliente**: Obtiene cliente gRPC configurado
- **Stream**: Establece conexión de streaming bidireccional

#### 2. Loop de Recepción
```go
for stream.Receive() {
    if err := stream.Err(); err != nil {
        log.Println("Error with stream:", err)
        continue
    }
    // Procesar mensaje
}
```
- **Receive**: Bloquea hasta recibir mensaje
- **Error handling**: Continúa en caso de errores no fatales
- **Logging**: Registra errores para debugging

#### 3. Filtrado de Mensajes
```go
msg := stream.Msg().GetMessage()
if msg != nil && msg.RoomId == room.Id && uint32(msg.GetSenderId()) != me.Id {
    // Procesar mensaje válido
}
```

**Condiciones de Filtro:**
- **Mensaje válido**: No es nil
- **Sala correcta**: Pertenece a la sala actual
- **No es propio**: No fue enviado por el usuario actual

#### 4. Desencriptación y Formateo
```go
contact := msg.GetContactName()
content, err := utils.DecryptMessage(msg.Content, room.EncryptionData)
if err != nil {
    log.Println("Error decripting message:", err)
}
msgChan <- message(fmt.Sprintf("%s: %s", contact, content))
```
- **Contacto**: Obtiene nombre del remitente
- **Desencriptación**: Desencripta contenido del mensaje
- **Formato**: Combina nombre y contenido
- **Envío**: Envía al canal para actualizar UI

### Envío de Mensajes

```go
func sendMessage(content string) {
    message, err := utils.EncryptMessage(content, room.EncryptionData)
    if err != nil {
        fmt.Println("Error enviando mensaje: ", err)
        return
    }
    _, err = chatv1client.SendMessage(context.Background(), generalParams, &chatv1.SendMessageRequest{
        Type:         "user",
        RoomId:       room.Id,
        Content:      message,
        ContactName:  &me.Name,
        ContactPhone: &me.Phone,
    })
    if err != nil {
        fmt.Println("Error enviando mensaje: ", err)
        return
    }
    log.Printf("Mensaje a enviar: '%s'", content)
}
```

**Proceso de Envío:**
1. **Encriptación**: Encripta contenido con claves de la sala
2. **Request**: Crea request con datos del mensaje
3. **Metadatos**: Incluye información de contacto
4. **Envío**: Llama al servicio gRPC
5. **Logging**: Registra mensaje enviado

## Implementación de Bubble Tea

### Función Init

```go
func (m model) Init() tea.Cmd {
    go listenForMessages(m.state.ctx, m.state.messageChan)
    return m.waitForMessage()
}
```
- **Goroutine**: Inicia listener en background
- **Comando**: Retorna comando para esperar mensajes

### Función Update

```go
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    var (
        tiCmd tea.Cmd
        vpCmd tea.Cmd
        cmd   tea.Cmd
    )
    
    m.textarea, tiCmd = m.textarea.Update(msg)
    m.viewport, vpCmd = m.viewport.Update(msg)
    
    switch msg := msg.(type) {
    case tea.KeyMsg:
        switch msg.Type {
        case tea.KeyCtrlC, tea.KeyEsc:
            m.state.cancel()
            return m, tea.Quit
        case tea.KeyEnter:
            userInput := m.textarea.Value()
            if userInput != "" {
                sendMessage(userInput)
                m.state.messages = append(m.state.messages, m.senderStyle.Render("Tú: ")+userInput)
                m.viewport.SetContent(strings.Join(m.state.messages, "\n"))
                m.textarea.Reset()
                m.viewport.GotoBottom()
            }
        }
    
    case message:
        m.state.messages = append(m.state.messages, string(msg))
        m.viewport.SetContent(strings.Join(m.state.messages, "\n"))
        m.viewport.GotoBottom()
        cmd = m.waitForMessage()
    
    case errMessage:
        m.state.err = msg
        return m, nil
    }
    
    return m, tea.Batch(tiCmd, vpCmd, cmd)
}
```

**Manejo de Eventos:**

#### Teclas de Control
- **Ctrl+C/Esc**: Cancela contexto y sale de la aplicación
- **Enter**: Envía mensaje y actualiza interfaz

#### Mensajes Recibidos
- **Agregar a historial**: Añade mensaje a la lista
- **Actualizar viewport**: Refresca la visualización
- **Scroll automático**: Va al final de la conversación
- **Continuar escuchando**: Programa próximo mensaje

#### Errores
- **Almacenar error**: Guarda en estado para mostrar
- **No terminar**: Permite recuperación

### Función View

```go
func (m model) View() string {
    if m.state.err != nil {
        return fmt.Sprintf("Error: %v", m.state.err)
    }
    
    return fmt.Sprintf(`%s

%s`,
        m.viewport.View(),
        m.textarea.View(),
    ) + "\n"
}
```

**Layout:**
- **Viewport**: Área superior para mensajes
- **TextArea**: Área inferior para entrada
- **Error handling**: Muestra errores si existen

### Función de Espera

```go
func (m *model) waitForMessage() tea.Cmd {
    return func() tea.Msg {
        return <-m.state.messageChan
    }
}
```
- **Comando**: Convierte canal en comando de Bubble Tea
- **Bloqueo**: Espera hasta recibir mensaje
- **Conversión**: Convierte mensaje del canal a mensaje de Bubble Tea

## Casos de Uso

### Uso Básico - Chat P2P
```bash
# Crear chat con usuario 456
./chat-cli -u 123 -t 456
```

### Uso Avanzado - Unirse a Sala Existente
```bash
# Unirse a sala específica
./chat-cli -u 123 -r 789
```

## Consideraciones de Seguridad

1. **Encriptación**: Todos los mensajes se encriptan antes del envío
2. **Autenticación**: Token JWT para todas las operaciones
3. **Validación**: Verificación de permisos de sala
4. **Logging**: Información sensible no se registra en logs

## Performance y Optimización

1. **Streaming**: Conexión persistente para tiempo real
2. **Goroutines**: Procesamiento asíncrono de mensajes
3. **Buffering**: Canal con buffer para mensajes
4. **Memory**: Gestión eficiente de historial de mensajes

## Mejores Prácticas Implementadas

1. **Separation of Concerns**: UI separada de lógica de negocio
2. **Error Handling**: Manejo robusto de errores
3. **Resource Management**: Cleanup apropiado de recursos
4. **User Experience**: Interfaz intuitiva y responsiva
5. **Security**: Encriptación end-to-end
6. **Logging**: Información detallada para debugging

Este CLI proporciona una interfaz completa y funcional para interactuar con el sistema de chat, demostrando todas las capacidades de la API de manera práctica y user-friendly.