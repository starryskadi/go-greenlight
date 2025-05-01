#!/bin/sh

if [ -z "$1" ]; then
  echo "Usage: ./migratedb.sh migration_name"
  exit 1
fi

migrate create -seq -ext=.sql -dir=./migrations "$1"