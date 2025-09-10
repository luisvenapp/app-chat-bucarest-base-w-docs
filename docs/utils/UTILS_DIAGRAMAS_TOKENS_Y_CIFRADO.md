# Diagramas de flujo — utils

## Validación de tokens
```mermaid
flowchart TD
    A[ValidatePublicToken] --> B[auth.GetTokenFromHeader]
    B --> C{token == publictoken?}
    C -- No --> X[error invalid token]
    C -- Sí --> D[true]
```

```mermaid
flowchart TD
    A[ValidateAuthToken(req)] --> B[api.CheckSessionFromConnectRequest]
    B --> C{ok?}
    C -- No --> X[connect.CodeUnauthenticated]
    C -- Sí --> D[retornar userID]
```

## Formateo de sala
```mermaid
flowchart TD
    A[FormatRoom] --> B{room.Type == p2p && Partner != nil}
    B -- Sí --> C[room.PhotoUrl = partner.avatar; room.Name = partner.name]
    B --> D{room.LastMessage != nil}
    D -- Sí --> E[room.LastMessageAt = LastMessage.CreatedAt]
```

## Cifrado de mensajes
```mermaid
flowchart TD
    A[GenerateKeyEncript] --> B[Random salt + iv]
    B --> C[scrypt->key]
    C --> D[makePublicEncryptUtil(AES-CBC con masterKey/IV)]
    D --> E[toBase64]
```

```mermaid
flowchart TD
    A[EncryptMessage] --> B[decodificar encriptionData]
    B --> C[obtener key/iv]
    C --> D[PKCS7 Padding]
    D --> E[AES-CBC Encrypt]
    E --> F[Base64]
```

```mermaid
flowchart TD
    A[DecryptMessage] --> B[decodificar encriptionData]
    B --> C[obtener key/iv]
    C --> D[Base64 decode mensaje]
    D --> E[AES-CBC Decrypt]
    E --> F[PKCS7 Unpadding]
```
