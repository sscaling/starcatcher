#!/bin/bash

# statically link to libraries, so there is no external dependencies. I.e. this can be built from scratch
CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main .
docker build -t sscaling/starcatcher .