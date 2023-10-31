FROM golang:1.21-alpine

# Set destination for COPY
WORKDIR /app
RUN apk update
RUN  apk add --no-cache curl tcpdump  openssh openssl  busybox-extras socat nmap nfs-utils openrc rpcbind
# Download Go modules
COPY go.mod go.sum ./
RUN go mod download

# Copy the source code. Note the slash at the end, as explained in
# https://docs.docker.com/engine/reference/builder/#copy
COPY *.go ./

# Build
RUN CGO_ENABLED=0 GOOS=linux go build -o /secret-converter
CMD [ "/secret-converter" ]