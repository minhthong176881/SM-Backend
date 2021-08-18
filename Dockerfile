FROM golang:1.17-alpine
WORKDIR /app
COPY . .
# RUN apk update
# RUN apk add make
# RUN apk add curl
# RUN apk add bash
# RUN make install
# RUN go mod tidy
RUN go build 
# EXPOSE 11000
# CMD [ "ls" ]

FROM golang:1.17-alpine
WORKDIR /app
COPY --from=0 /app/Server_Management /app/Server_Management
COPY --from=0 /app/.env /app/.env
COPY --from=0 /app/cert.pem /app/cert.pem 
COPY --from=0 /app/key.pem /app/key.pem

EXPOSE 11000
CMD ./Server_Management