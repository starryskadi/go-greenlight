#!/bin/sh

export $(cat .env | xargs)
migrate -path=./migrations -database=$GREENLIGHT_DB_DSN $@