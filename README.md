# dist_db
This repo was created by following https://jacobmartins.com/2017/01/29/practical-golang-building-a-simple-distributed-one-value-database-with-hashicorp-serf/.


## Dependencies
```
"github.com/gorilla/mux"
"github.com/hashicorp/serf/serf"
"github.com/pkg/errors"
"golang.org/x/sync/errgroup"
```

## Initial Node Call
go run dist_db.go 


## Dockerfile
```
FROM golang:1.11
WORKDIR $GOPATH/src/github.com/dist_db
COPY . .
RUN go get -d -v ./...
RUN go install -v ./...
EXPOSE 8080
# Run the executable
CMD ["dist_db"]
```

## Docker Build
```
docker build -t dist_db .
```

## Inspect Bridge - Get Gateway
```
docker network inspect bridge
```



## Run Docker Containers
```
docker run -e ADVERTISE_ADDR=172.17.0.2 -e MASTER_ADDR=172.17.0.2 -p 8080:8080 dist_db
docker run -e ADVERTISE_ADDR=172.17.0.3 -e CLUSTER_ADDR=172.17.0.2 -p 8081:8080 dist_db
docker run -e ADVERTISE_ADDR=172.17.0.4 -e CLUSTER_ADDR=172.17.0.3 -p 8082:8080 dist_db
```