FROM golang:1.17-alpine AS build
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
COPY --from=build /app/Server_Management /app/Server_Management
COPY --from=build /app/.env /app/.env
# COPY --from=build /app/cert.pem /app/cert.pem 
# COPY --from=build /app/key.pem /app/key.pem

EXPOSE 11000
CMD ./Server_Management