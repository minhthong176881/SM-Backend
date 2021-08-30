package server

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/360EntSecGroup-Skylar/excelize/v2"
	"github.com/joho/godotenv"
	"github.com/minhthong176881/Server_Management/middleware"
	pbSM "github.com/minhthong176881/Server_Management/proto"
	"github.com/minhthong176881/Server_Management/service/serverLogService"
	"github.com/minhthong176881/Server_Management/service/serverService"
	"github.com/minhthong176881/Server_Management/service/serverStatusService"
	"github.com/minhthong176881/Server_Management/service/userService"
	"github.com/minhthong176881/Server_Management/utils"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type Backend struct {
	baseService  serverService.ServerService
	serverLog    serverLogService.ServerLogService
	serverStatus serverStatusService.ServerStatusService
	user         userService.UserService
	jwtManager   *middleware.JWTManager
}

func New(base serverService.ServerService, serverLog serverLogService.ServerLogService, serverStatus serverStatusService.ServerStatusService, user userService.UserService, jwtManager *middleware.JWTManager) *Backend {
	return &Backend{
		baseService:  base,
		serverLog:    serverLog,
		serverStatus: serverStatus,
		user:         user,
		jwtManager:   jwtManager,
	}
}

func (b *Backend) Register(_ context.Context, req *pbSM.RegisterRequest) (*pbSM.User, error) {
	user := req.GetUser()

	data := userService.UserItem{
		Username: req.User.Username,
		Password: req.User.Password,
		Email:    req.User.Email,
		Role:     req.User.Role,
	}
	result, err := b.user.Register(&data)
	if err != nil {
		return nil, err
	}
	user.Id = result
	return user, nil
}

func (b *Backend) Login(ctx context.Context, req *pbSM.LoginRequest) (*pbSM.LoginResponse, error) {
	data, err := b.user.Login(req.GetUsername(), req.GetPassword())
	if err != nil {
		return nil, err
	}
	
	jwtData := middleware.UserItem {
		ID: data.ID,
		Username: data.Username,
		Email: data.Email,
		Role: data.Role,
		Password: data.Password,
	}
	token, err := b.jwtManager.Generate(&jwtData)
	if err != nil {
		return nil, status.Errorf(codes.Internal, fmt.Sprintf("Internal error %v", err))
	}
	return &pbSM.LoginResponse{AccessToken: token}, nil
}

func (b *Backend) Logout(ctx context.Context, req *pbSM.LogoutRequest) (*pbSM.LogoutResponse, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, status.Errorf(codes.Unauthenticated, "metadata is not provided")
	}

	values := md["authorization"]
	if len(values) == 0 { // no auth header
		return nil, status.Errorf(codes.Unauthenticated, "authorization token is not provided")
	}
	loggedOut, err := b.user.Logout(values[0])
	if err != nil {
		return nil, err
	}
	return &pbSM.LogoutResponse{LoggedOut: loggedOut}, nil
}

func (b *Backend) GetServers(_ context.Context, req *pbSM.GetServersRequest) (*pbSM.GetServersResponse, error) {
	query, pageIndex, pageOffset := req.GetQuery(), req.GetPageIndex(), req.GetPageOffset()
	servers, total, err := b.baseService.GetAll(serverService.Query{Query: query, PageIndex: pageIndex, PageOffset: pageOffset})
	if err != nil {
		return nil, err
	}
	var pbSMServers []*pbSM.Server
	for i := 0; i < len(servers); i++ {
		server := utils.ServiceToPbSM(servers[i])
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

	result, err := b.baseService.Insert(&serverService.Server{
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
	server = utils.ServiceToPbSM(result)
	return server, nil
}

func (b *Backend) GetServerById(ctx context.Context, req *pbSM.GetServerByIdRequest) (*pbSM.Server, error) {
	server, err := b.baseService.GetById(req.GetId())
	if err != nil {
		return nil, err
	}
	response := utils.ServiceToPbSM(server)
	return response, nil
}

func (b *Backend) UpdateServer(ctx context.Context, req *pbSM.UpdateServerRequest) (*pbSM.Server, error) {
	reqServer, err := utils.PbSMToService(req.GetServer())
	if err != nil {
		return nil, err
	}
	server, err := b.baseService.Update(req.GetId(), reqServer)
	if err != nil {
		return nil, err
	}
	response := utils.ServiceToPbSM(server)
	return response, nil
}

func (b *Backend) DeleteServer(ctx context.Context, req *pbSM.GetServerByIdRequest) (*pbSM.DeleteServerResponse, error) {
	err := b.baseService.Delete(req.GetId())
	if err != nil {
		return nil, err
	}
	return &pbSM.DeleteServerResponse{Deleted: 1}, nil
}

func (b *Backend) ExportServers(ctx context.Context, req *pbSM.ExportServersRequest) (*pbSM.ExportServersResponse, error) {
	var myTableName = "Server list"
	f := excelize.NewFile()
	f.DeleteSheet("Sheet1")
	index := f.NewSheet(myTableName)
	_ = f.SetCellValue(myTableName, "A2", "Server")
	_ = f.SetCellValue(myTableName, "B2", "IP")
	_ = f.SetCellValue(myTableName, "C2", "Username")
	_ = f.SetCellValue(myTableName, "D2", "Password")
	_ = f.SetCellValue(myTableName, "E2", "Status")
	_ = f.SetCellValue(myTableName, "F2", "Password validate")
	_ = f.SetCellValue(myTableName, "G2", "Description")

	servers, _, err := b.baseService.GetAll(serverService.Query{})
	if err != nil {
		return nil, err
	}
	for i := 3; i < len(servers)+3; i++ {
		num := strconv.FormatInt(int64(i), 10)
		var status string
		if servers[i-3].Status {
			status = "On"
		} else {
			status = "Off"
		}
		var validate string
		if servers[i-3].Validate {
			validate = "Valid"
		} else {
			validate = "Invalid"
		}
		_ = f.SetCellValue(myTableName, "A"+num, i-2)
		_ = f.SetCellValue(myTableName, "B"+num, servers[i-3].Ip)
		_ = f.SetCellValue(myTableName, "C"+num, servers[i-3].Username)
		_ = f.SetCellValue(myTableName, "D"+num, servers[i-3].Password)
		_ = f.SetCellValue(myTableName, "E"+num, status)
		_ = f.SetCellValue(myTableName, "F"+num, validate)
		_ = f.SetCellValue(myTableName, "G"+num, servers[i-3].Description)
	}
	f.SetActiveSheet(index)
	f.Path = "public/OpenAPI/exports/Server_list.xlsx"
	_ = f.Save()

	err = godotenv.Load(".env")
	if err != nil {
		log.Fatalf("Error loading .env file")
	}
	host := os.Getenv("HOST")
	downloadUrl := host + "/exports/Server_list.xlsx"
	return &pbSM.ExportServersResponse{DownloadUrl: downloadUrl}, nil
}

func (b *Backend) CheckServer(ctx context.Context, req *pbSM.GetServerByIdRequest) (*pbSM.CheckServerResponse, error) {
	server, err := b.baseService.GetById(req.GetId())
	if err != nil {
		return nil, err
	}
	status, err := b.serverStatus.Check(server)
	if err != nil {
		return nil, err
	}
	return &pbSM.CheckServerResponse{Status: status}, nil
}

func (b *Backend) GetServerLog(ctx context.Context, req *pbSM.GetServerLogRequest) (*pbSM.GetServerLogResponse, error) {
	logs, changeLogs, err := b.serverLog.GetLog(req.GetId(), req.GetStart(), req.GetEnd(), req.GetDate(), req.GetMonth())
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
	server, err := b.baseService.GetById(req.GetId())
	if err != nil {
		return nil, err
	}
	validate, err := b.serverStatus.Validate(server)
	if err != nil {
		return nil, err
	}
	return &pbSM.ValidateServerResponse{Validated: validate}, nil
}
