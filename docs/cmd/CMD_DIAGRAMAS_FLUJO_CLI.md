# Diagramas de flujo — cmd

Este documento resume los flujos de la aplicación CLI campaing-app-chat-cli.

## Flujo principal de ejecución
```mermaid
flowchart TD
    A[Inicio main()] --> B[Parsear flags -u -t -r]
    B --> C{userID válido?}
    C -- No --> X[log.Fatal usuario requerido]
    C -- Sí --> D[Generar token de sesión]
    D --> E[Validar token y obtener session]
    E --> F[Configurar GeneralParams]
    F --> G[authv1client.Me para datos del usuario]
    G --> H{roomID proporcionado?}
    H -- Sí --> I[chatv1client.GetRoom]
    H -- No --> J{remitentID válido?}
    J -- No --> X
    J -- Sí --> K[chatv1client.CreateRoom]
    I --> L[room listo]
    K --> L[room listo]
    L --> M[Preparar logging tmp/chat-debug.log]
    M --> N[Crear Bubble Tea Program]
    N --> O[Run()]
```

## Recepción de mensajes por streaming
```mermaid
flowchart TD
    A[listenForMessages] --> B[api.NewRequest(GeneralParams)]
    B --> C[client := chatv1client.GetChatServiceClient()]
    C --> D[client.StreamMessages]
    D --> E{Receive loop}
    E --> F[Validar errores]
    E --> G[Filtrar por room.Id y sender != me]
    G --> H[utils.DecryptMessage]
    H --> I[Enviar al canal messageChan]
```

## Envío de mensajes
```mermaid
flowchart TD
    A[sendMessage(content)] --> B[utils.EncryptMessage]
    B --> C[chatv1client.SendMessage]
    C --> D[Log de mensaje enviado]
```

## Ciclo Bubble Tea (Update)
```mermaid
flowchart TD
    A[Update(msg)] --> B{Key?
Ctrl+C/Esc?}
    B -- Sí --> C[cancel ctx y Quit]
    B -- No --> D{Enter?}
    D -- Sí --> E[Tomar textarea.Value]
    E --> F[sendMessage]
    F --> G[Append mensaje propio y reset UI]
    D -- No --> H{message recibido?}
    H -- Sí --> I[Append y scroll]
```
