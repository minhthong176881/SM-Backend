# grpc-gateway-boilerplate
All the boilerplate you need to get started with writing grpc-gateway powered
REST services in Go.

## Running

Running `main.go` starts a web server on https://0.0.0.0:11000/. You can configure
the port used with the `$PORT` environment variable, and to serve on HTTP set
`$SERVE_HTTP=true`.

```
$ go run main.go
```

An OpenAPI UI is served on https://0.0.0.0:11000/.

## Getting started

After cloning the repo, there are a couple of initial steps;

1. Install the generate dependencies with `make install`.
   This will install `buf`, `protoc-gen-go`, `protoc-gen-go-grpc`, `protoc-gen-grpc-gateway` and
   `protoc-gen-openapiv2` which are necessary for us to generate the Go and swagger files.
1. If you forked this repo, or cloned it into a different directory from the github structure,
   you will need to correct the import paths. Here's a nice `find` one-liner for accomplishing this
   (replace `yourscmprovider.com/youruser/yourrepo` with your cloned repo path):
   ```bash
   $ find . -path ./vendor -prune -o -type f \( -name '*.go' -o -name '*.proto' \) -exec sed -i -e "s;github.com/minhthong176881/Server_Management;yourscmprovider.com/youruser/yourrepo;g" {} +
   ```
1. Finally, generate the files with `make generate`.

Now you can run the web server with `go run main.go`.

## Build docker
1. Change elasticsearch_host in .env to `http://elasticsearch:9200` and redis_host to `redis:6379`

2. Build image: `docker build -t server-management`

2. Run `docker-compose up`

## Certificate
1. Add cert.pem and key.pem into root foler (same directory with file `main.go`)
2. Modify ABSOLUTE_PATH in .env file

