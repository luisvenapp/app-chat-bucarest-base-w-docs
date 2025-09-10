package database

import (
	"database/sql"
	"log"

	"github.com/Venqis-NolaTech/campaing-app-core-go/pkg/db/cassandra"
	"github.com/Venqis-NolaTech/campaing-app-core-go/pkg/db/postgres"
	"github.com/scylladb-solutions/gocql/v2"
)

var db *sql.DB
var cassandraDB *gocql.Session

func CQLDB() *gocql.Session {
	return cassandraDB
}

func DB() *sql.DB {
	return db
}

func init() {
	instance, err := dbpq.ConnectToNewSQLInstance(dbpq.DefaultConnectionString)
	if err != nil {
		log.Fatal("ERROR CONNECTING TO DB: ", err)
	}
	db = instance

	cassandraDB_, err := cassandra.Connect(cassandra.DefaultConnectionConfig)
	if err != nil {
		log.Println("ERROR CONNECTING TO CASSANDRA: ", err)
		return
	}
	cassandraDB = cassandraDB_
}
