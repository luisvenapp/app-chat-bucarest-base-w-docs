# handlers/tokens/v1/handler.go

Propósito
- Servicio TokensService: persiste tokens de notificaciones (push/APNs/FCM/VoIP) asociados al usuario autenticado.

Método
- SaveToken(ctx, req)
  - Valida sesión con utils.ValidateAuthToken.
  - Construye repositorio SQLTokensRepository(database.DB()).
  - Inserta el token con metadata (platform, version, device, lang, flags) y user_id.
  - Devuelve { success: true }.

Notas
- Mantiene la separación handler/repository; cualquier validación extra o deduplicación podría incorporarse en el repo si se requiere.