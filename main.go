package main

import (
	"io/ioutil"
	"net"
	"os"
	// "time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/grpclog"

	"github.com/minhthong176881/Server_Management/gateway"
	"github.com/minhthong176881/Server_Management/insecure"
	pbSM "github.com/minhthong176881/Server_Management/proto"
	server "github.com/minhthong176881/Server_Management/server"
	"github.com/minhthong176881/Server_Management/services/serverLogService"
	"github.com/minhthong176881/Server_Management/services/serverService"
	"github.com/minhthong176881/Server_Management/services/serverStatusService"
	"github.com/minhthong176881/Server_Management/services/userService"
	"github.com/minhthong176881/Server_Management/workers"
)

func main() {
	// Adds gRPC internal logs. This is quite verbose, so adjust as desired!
	log := grpclog.NewLoggerV2(os.Stdout, ioutil.Discard, ioutil.Discard)
	grpclog.SetLoggerV2(log)

	addr := "0.0.0.0:10000"
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalln("Failed to listen:", err)
	}

	s := grpc.NewServer(
		// TODO: Replace with your own certificate!
		grpc.Creds(credentials.NewServerTLSFromCert(&insecure.Cert)),
	)
	if err != nil {
		log.Fatalln(err)
	}

	mongoServerService := serverService.NewMongoServerService()
	redisServerService := serverService.NewRedisServerService(mongoServerService)
	user := userService.NewUser(mongoServerService)
	elasticsearchServerService := serverLogService.NewElasticsearchServerService()
	serverLog := serverLogService.NewServerLog(elasticsearchServerService)
	serverStatusUpdateWorker := workers.NewServerStatusUpdateWorker(redisServerService, serverLog)
	serverStatus := serverStatusService.NewServerStatus(serverStatusUpdateWorker)
	// time.Sleep(30 * time.Second)

	pbSM.RegisterSMServiceServer(s, server.New(redisServerService, serverLog, serverStatus, user))

	// Serve gRPC Server
	log.Info("Serving gRPC on https://", addr)
	go func() {
		log.Fatal(s.Serve(lis))
	}()

	go func() {
		serverStatusUpdateWorker.ExecuteCronJob()
	}()

	err = gateway.Run("dns:///" + addr)
	log.Fatalln(err)
}
