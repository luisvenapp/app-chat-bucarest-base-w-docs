# cmd/campaing-app-chat-cli/main.go

Descripción general
- Cliente CLI/TUI (Bubble Tea) para probar/usar el servicio de chat.
- Crea o abre una sala, se suscribe al stream de eventos y permite enviar/recibir mensajes en tiempo real con cifrado de extremo a extremo a nivel de contenido usando utils.EncryptMessage/DecryptMessage.

Flags
- -u (user id) [requerido]: ID del usuario autenticado que opera el cliente.
- -t (remitent id): ID del usuario con quien crear chat P2P si no se pasa -r.
- -r (room id): ID de sala existente a la que conectarse.

Flujo de ejecución
1) Autenticación
   - Genera un token de sesión con auth.GenerateSessionToken y obtiene la sesión (auth.ValidateSessionToken).
   - Construye api.GeneralParams con datos de sesión, lang, platform, timezone y clientId (uuid).
   - Obtiene datos del usuario (Me) mediante authv1client.Me.
2) Selección de sala
   - Si se pasa -r, consulta chatv1client.GetRoom y la abre.
   - Si no, y hay -t, crea sala P2P con chatv1client.CreateRoom.
3) Interfaz TUI
   - Usa bubbles viewport y textarea (sin salto de línea con Enter) y alt-screen.
   - Muestra información de usuario y sala.
4) Recepción de mensajes
   - Goroutine listenForMessages: llama client.StreamMessages(ctx, req) y recibe eventos.
   - Filtra eventos MessageEvent con Message de la misma sala y que no sean del propio remitente.
   - Descifra el contenido con utils.DecryptMessage(room.EncryptionData) y los encola al canal interno.
5) Envío de mensajes
   - sendMessage: cifra el texto plano con utils.EncryptMessage usando room.EncryptionData y llama a chatv1client.SendMessage (type=user, room_id, contact metadata opcional).
   - Muestra el propio mensaje en el viewport (prefijo "Tú:").
6) Ciclo de actualización
   - Update gestiona teclas: Ctrl+C/Esc sale; Enter envía.
   - Render (View) combina viewport + textarea.

Puntos clave
- Cifrado: utils.EncryptMessage/DecryptMessage emplean AES-CBC con padding PKCS7; la clave de sala viene empaquetada en room.EncryptionData (ver utils/generateKeyEncript.go).
- Stream: stream.Receive() itera y, por cada evento, valida tipo y sala antes de mostrar.
- Logging: crea tmp/chat-debug.log con tea.LogToFile.

Dependencias externas
- auth API: generación y validación de token + Me.
- chatv1client: cliente ConnectRPC del servicio de chat.
- Bubble Tea/Bubbles/Lipgloss para TUI.

Ejemplos de uso
- Crear P2P con el usuario 2:  go run ./cmd/campaing-app-chat-cli -u 1 -t 2
- Abrir sala existente 123:    go run ./cmd/campaing-app-chat-cli -u 1 -r 123

Consideraciones
- Requiere backends en ejecución (auth, chat, nats, db). El CLI usa la misma configuración que el resto del proyecto para endpoints/clientes.
- La UI es minimalista: Enter envía, no hay nuevas líneas.
- Se usa clientId para gestionar consumidores durables de JetStream y heartbeats de conexión (el handler del server envía eventos Connected cada 15s).