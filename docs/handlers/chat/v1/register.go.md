# handlers/chat/v1/register.go

- Expone RegisterServiceHandler() que crea un Vanguard Service para ChatService.
- NewChatServiceHandler(NewHandler(), options...) asocia la implementación del handler con middlewares/opciones comunes del servidor (ServiceHandlerOptions del core).
- El servidor principal recoge esta función desde handlers/handlers.go.