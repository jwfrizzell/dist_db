FROM golang:1.11
WORKDIR $GOPATH/src/github.com/dist_db
COPY . .
RUN go get -d -v ./...
RUN go install -v ./...
EXPOSE 8080
# Run the executable
CMD ["dist_db"]