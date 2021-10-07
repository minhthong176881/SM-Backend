FROM golang:1.17-alpine AS build
WORKDIR /app
COPY . .
RUN go build 

FROM golang:1.17-alpine
WORKDIR /app
COPY --from=build /app/Server_Management /app/Server_Management
COPY --from=build /app/.env /app/.env
# COPY --from=build /app/key.pem /app/key.pem
# COPY --from=build /app/cert.pem /app/cert.pem

EXPOSE 11000
CMD ./Server_Management