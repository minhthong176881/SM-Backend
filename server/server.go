package server

import (
	"context"

	pbSM "github.com/minhthong176881/Server_Management/proto"
	services "github.com/minhthong176881/Server_Management/services"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Backend struct {
	redisService *services.RedisServerService
}

func New(redisService *services.RedisServerService) *Backend {
	return &Backend{
		redisService: redisService,
	}
}

func (b *Backend) Register(_ context.Context, req *pbSM.RegisterRequest) (*pbSM.User, error) {
	user := req.GetUser()

	data := services.User{
		Username: req.User.Username,
		Password: req.User.Password,
		Email:    req.User.Email,
	}
	result, err := b.redisService.Register(&data)
	if err != nil {
		return nil, err
	}
	user.Id = result
	return user, nil
}

func (b *Backend) Login(ctx context.Context, req *pbSM.LoginRequest) (*pbSM.LoginResponse, error) {
	logged, err := b.redisService.Login(req.GetUsername(), req.GetPassword())
	if err != nil {
		return nil, err
	}
	return &pbSM.LoginResponse{Logged: logged}, nil
}

func (b *Backend) GetServers(_ context.Context, req *pbSM.GetServersRequest) (*pbSM.GetServersResponse, error) {
	query, pageIndex, pageOffset := req.GetQuery(), req.GetPageIndex(), req.GetPageOffset()
	servers, total, err := b.redisService.GetAll(services.Query{Query: query, PageIndex: pageIndex, PageOffset: pageOffset})
	if err != nil {
		return nil, err
	}
	var pbSMServers []*pbSM.Server
	for i := 0; i < len(servers); i++ {
		server := services.ServiceToPbSM(servers[i])
		pbSMServers = append(pbSMServers, server)
	}
	return &pbSM.GetServersResponse{Servers: pbSMServers, Total: total}, nil
}

func (b *Backend) AddServer(ctx context.Context, req *pbSM.AddServerRequest) (*pbSM.Server, error) {
	server := req.GetServer()

	if server.GetIp() == "" || server.GetPort() == 0 || server.GetUsername() == "" || server.GetPassword() == "" {
		return nil, status.Error(
			codes.InvalidArgument,
			"Invalid payload!",
		)
	}

	result, err := b.redisService.Insert(&services.Server{
		Ip:          req.Server.Ip,
		Port:        req.Server.Port,
		Username:    req.Server.Username,
		Password:    req.Server.Password,
		Description: req.Server.Description,
		Status:      true,
		Validate:    true,
	})
	if err != nil {
		return nil, err
	}
	server = services.ServiceToPbSM(result)
	return server, nil
}

func (b *Backend) GetServerById(ctx context.Context, req *pbSM.GetServerByIdRequest) (*pbSM.Server, error) {
	server, err := b.redisService.GetById(req.GetId())
	if err != nil {
		return nil, err
	}
	response := services.ServiceToPbSM(server)
	return response, nil
}

func (b *Backend) UpdateServer(ctx context.Context, req *pbSM.UpdateServerRequest) (*pbSM.Server, error) {
	reqServer, err := services.PbSMToService(req.GetServer())
	if err != nil {
		return nil, err
	}
	server, err := b.redisService.Update(req.GetId(), reqServer)
	if err != nil {
		return nil, err
	}
	response := services.ServiceToPbSM(server)
	return response, nil
}

func (b *Backend) DeleteServer(ctx context.Context, req *pbSM.GetServerByIdRequest) (*pbSM.DeleteServerResponse, error) {
	err := b.redisService.Delete(req.GetId())
	if err != nil {
		return nil, err
	}
	return &pbSM.DeleteServerResponse{Deleted: 1}, nil
}

func (b *Backend) ExportServers(ctx context.Context, req *pbSM.ExportServersRequest) (*pbSM.ExportServersResponse, error) {
	downloadUrl, err := b.redisService.Export()
	if err != nil {
		return nil, err
	}
	return &pbSM.ExportServersResponse{DownloadUrl: downloadUrl}, nil
}

func (b *Backend) CheckServer(ctx context.Context, req *pbSM.GetServerByIdRequest) (*pbSM.CheckServerResponse, error) {
	status, err := b.redisService.Check(req.GetId())
	if err != nil {
		return nil, err
	}
	return &pbSM.CheckServerResponse{Status: status}, nil
}

func (b *Backend) GetServerLog(ctx context.Context, req *pbSM.GetServerLogRequest) (*pbSM.GetServerLogResponse, error) {
	logs, changeLogs, err := b.redisService.GetLog(req.GetId(), req.GetStart(), req.GetEnd(), req.GetDate(), req.GetMonth())
	if err != nil {
		return nil, err
	}
	var resLogs []*pbSM.ServerLog
	var resChangeLogs []*pbSM.ChangeLog
	for i := 0; i < len(logs); i++ {
		log := pbSM.ServerLog{}
		log.Status = logs[i].Status
		log.Time = logs[i].Time
		resLogs = append(resLogs, &log)
	}
	for j := 0; j < len(changeLogs); j++ {
		changeLog := pbSM.ChangeLog{}
		changeLog.Start = changeLogs[j].Start
		changeLog.End = changeLogs[j].End
		changeLog.Total = changeLogs[j].Total
		resChangeLogs = append(resChangeLogs, &changeLog)
	}
	return &pbSM.GetServerLogResponse{Logs: resLogs, ChangeLogs: resChangeLogs}, nil
}

func (b *Backend) ValidateServer(ctx context.Context, req *pbSM.GetServerByIdRequest) (*pbSM.ValidateServerResponse, error) {
	validate, err := b.redisService.Validate(req.GetId())
	if err != nil {
		return nil, err
	}
	return &pbSM.ValidateServerResponse{Validated: validate}, nil
}
