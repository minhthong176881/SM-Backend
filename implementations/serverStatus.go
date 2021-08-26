package implementations

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/360EntSecGroup-Skylar/excelize/v2"
	"github.com/joho/godotenv"
	services "github.com/minhthong176881/Server_Management/services"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type ServerStatus struct {
	baseService services.ServerService
}

func NewServerStatus(baseService services.ServerService) *ServerStatus {
	return &ServerStatus{
		baseService: baseService,
	}
}

func (s *ServerStatus) Export() (string, error) {
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

	servers, _, err := s.baseService.GetAll(services.Query{})
	if err != nil {
		return "", err
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
	return host + "/exports/Server_list.xlsx", nil
}

func (s *ServerStatus) Check(id string) (bool, error) {
	server, err := s.baseService.GetById(id)
	if err != nil {
		return false, status.Errorf(codes.NotFound, fmt.Sprintf("Could not find server with Id %s: %v", id, err))
	}
	conn, err := services.Connect(server.Ip+":"+strconv.FormatInt(server.Port, 10), server.Username, server.Password)
	if err != nil {
		if strings.Contains(err.Error(), "ssh") && strings.Contains(err.Error(), "handshake") {
			server.Status = true
			_, _ = s.baseService.Update(id, server)
			return true, nil
		}
		server.Status = false
		_, _ = s.baseService.Update(id, server)
		return false, err
	}
	if conn != nil {
		server.Status = true
		_, _ = s.baseService.Update(id, server)
		return true, nil
	} else {
		server.Status = false
		_, _ = s.baseService.Update(id, server)
		return false, nil
	}
}

func (s *ServerStatus) Validate(id string) (bool, error) {
	server, err := s.baseService.GetById(id)
	if err != nil {
		server.Validate = false
		_, _ = s.baseService.Update(id, server)
		return false, status.Errorf(codes.NotFound, fmt.Sprintf("Could not find server with Id %s: %v", id, err))
	}

	conn, err := services.Connect(server.Ip+":"+strconv.FormatInt(server.Port, 10), server.Username, server.Password)
	if err != nil {
		server.Validate = false
		_, _ = s.baseService.Update(id, server)
		return false, err
	}
	if conn != nil {
		server.Validate = true
		_, _ = s.baseService.Update(id, server)
		return true, nil
	} else {
		server.Validate = false
		_, _ = s.baseService.Update(id, server)
		return false, nil
	}
}

