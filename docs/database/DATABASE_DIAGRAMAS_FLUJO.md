# Diagramas de flujo — database

```mermaid
flowchart TD
    A[init()] --> B[Conectar a Postgres (dbpq.ConnectToNewSQLInstance)]
    B --> C{Error?}
    C -- Sí --> X[log.Fatal]
    C -- No --> D[db = instance]
    D --> E[Conectar a Cassandra/Scylla (cassandra.Connect)]
    E --> F{Error?}
    F -- Sí --> G[log.Println y continuar]
    F -- No --> H[cassandraDB = session]
    H --> I[Disponibilizar CQLDB() y DB()]
```
