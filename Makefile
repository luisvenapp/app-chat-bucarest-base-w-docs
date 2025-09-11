SHELL := /bin/bash

.PHONY: up down logs build migrate-cassandra

build:
	docker compose build

up:
	docker compose up -d

logs:
	docker compose logs -f app

migrate-cassandra:
	# Espera a que Scylla esté listo y aplica CQL
	docker compose exec -T scylla sh -lc 'until cqlsh -e "DESCRIBE KEYSPACES" 127.0.0.1 9042 >/dev/null 2>&1; do echo waiting for scylla; sleep 5; done; cqlsh -e "SOURCE \'/migrations/cassandra/0001_init.cql\';" 127.0.0.1 9042'
