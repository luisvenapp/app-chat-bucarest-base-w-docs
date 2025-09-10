# Diagramas de flujo — repository/tokens

```mermaid
flowchart TD
    A[SaveToken] --> B[BeginTx]
    B --> C[INSERT messaging_token]
    C --> D[Commit]
```
