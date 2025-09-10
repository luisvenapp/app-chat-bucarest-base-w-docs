# utils/generateKeyEncript.go

Resumen
- Utilidades de cifrado para mensajería. Maneja:
  - Generación de par clave/iv aleatorio (GenerateRandomKeyAndIV).
  - Empaquetado cifrado (AES-CBC) de { key, iv } con una master key/iv desde configuración (makePublicEncryptUtil) y codificación base64.
  - Cifrado/descifrado de mensajes de usuario con la clave/iv de la sala (EncryptMessage/DecryptMessage).

Flujo de claves
- La sala guarda `EncryptionData` que es: base64(hex(AES-CBC(masterKey/masterIv, json{"key","iv"}))).
- EncryptMessage/DecryptMessage:
  1) fromBase64(encriptionData) -> hex; makePublicDecryptUtil -> (key, iv) en hex.
  2) Cifran/descifran el contenido con AES-CBC(key, iv) + PKCS7.
  3) Para mensajes cifrados se usa toBase64(hex(encrypted)).

Funciones
- GenerateKeyEncript(): genera (key,iv) derivando key con scrypt(salt aleatorio), empaqueta con masterKey/masterIv y devuelve en base64.
- EncryptMessage(msg, encrData): valida entrada, aplica padding, cifra y devuelve base64.
- DecryptMessage(msg, encrData): decodifica base64->hex, valida tamaño múltiplo de 16 bytes, descifra y quita padding.
- toBase64/fromBase64: conversión hex <-> base64 de buffers.

Seguridad
- masterKey/masterIv vienen de config (chat.key/chat.iv) en hex; deben ser secretos y rotables.
- AES-CBC exige IV único y padding PKCS7. El contenido de mensajes viaja cifrado de cara al repositorio/transportes.
- Manejar errores cuando buffers no tienen longitud válida para AES.

Notas
- El empaquetado de (key,iv) permite rotación de claves de sala generando nuevos EncryptionData y compartiéndolos con clientes.