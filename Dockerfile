FROM golang:1.26.4

WORKDIR /app

COPY go.mod ./
COPY go.sum ./

RUN go mod download

COPY . .

RUN cd cmd && go build -o ../server

CMD [ "/app/server" ]
