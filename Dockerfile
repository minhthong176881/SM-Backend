FROM golang:1.16-alpine

WORKDIR /app

COPY . .

RUN apk update
# RUN apk add --upgrade elasticsearch

RUN apk add make
RUN apk add curl
RUN apk add bash

RUN make install

RUN go mod tidy

EXPOSE 11000

CMD [ "go", "run", "main.go" ]