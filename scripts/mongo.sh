#!/bin/bash

MONGO_DB=api-lnlink-app
USERNAME=admin
PASSWORD=admin
VOLUME=~/docker/$MONGO_DB/mongo/data
BOUND=0

if [ $BOUND -eq 1 ]; then
    mkdir -p $VOLUME

    docker run --name $MONGO_DB-mongo \
        -p 27017:27017 \
        --restart unless-stopped \
        -v $VOLUME:/data/db \
        -e MONGO_INITDB_ROOT_USERNAME=$USERNAME \
        -e MONGO_INITDB_ROOT_PASSWORD=$PASSWORD \
        -d mongo
else 
    docker run --name $MONGO_DB-mongo \
        -p 27017:27017 \
        --restart unless-stopped \
        -e MONGO_INITDB_ROOT_USERNAME=$USERNAME \
        -e MONGO_INITDB_ROOT_PASSWORD=$PASSWORD \
        -d mongo
fi