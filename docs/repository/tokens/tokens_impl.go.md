# repository/tokens/tokens_impl.go (SQLTokensRepository)

Guardar token (SaveToken)
- Transacción simple que inserta en public.messaging_token:
  - token, platform, platform_version, device, lang, is_voip, debug, user_id, created_at=NOW().
- Sin retorno de entidad; se usa sólo como registro para notificaciones push.

Notas
- Puede ampliarse con upsert/deduplicación si su plataforma lo requiere para evitar múltiples tokens por dispositivo.