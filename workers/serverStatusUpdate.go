package workers

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/360EntSecGroup-Skylar/excelize/v2"
	"github.com/joho/godotenv"
	"github.com/minhthong176881/Server_Management/services/serverLogService"
	"github.com/minhthong176881/Server_Management/services/serverService"
	"github.com/minhthong176881/Server_Management/utils"
)

type ServerStatusUpdateWorker struct {
	serverService       serverService.ServerService
	serverLogService    serverLogService.ServerLogService
}

func NewServerStatusUpdateWorker(serverService serverService.ServerService, serverLogService serverLogService.ServerLogService) *ServerStatusUpdateWorker {
	return &ServerStatusUpdateWorker{
		serverService:       serverService,
		serverLogService:    serverLogService,
	}
}

func (w *ServerStatusUpdateWorker) Export() (string, error) {
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

	servers, _, err := w.serverService.GetAll(serverService.Query{})
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

func (w *ServerStatusUpdateWorker) Check(server *serverService.Server) (bool, error) {
	conn, err := utils.Connect(server.Ip+":"+strconv.FormatInt(server.Port, 10), server.Username, server.Password)
	if err != nil {
		if strings.Contains(err.Error(), "ssh") && strings.Contains(err.Error(), "handshake") {
			server.Status = true
			_, _ = w.serverService.Update(server.ID.Hex(), server)
			return true, nil
		}
		server.Status = false
		_, _ = w.serverService.Update(server.ID.Hex(), server)
		return false, err
	}
	if conn != nil {
		server.Status = true
		_, _ = w.serverService.Update(server.ID.Hex(), server)
		return true, nil
	} else {
		server.Status = false
		_, _ = w.serverService.Update(server.ID.Hex(), server)
		return false, nil
	}
}

func (w *ServerStatusUpdateWorker) Validate(server *serverService.Server) (bool, error) {
	conn, err := utils.Connect(server.Ip+":"+strconv.FormatInt(server.Port, 10), server.Username, server.Password)
	if err != nil {
		server.Validate = false
		_, _ = w.serverService.Update(server.ID.Hex(), server)
		return false, err
	}
	if conn != nil {
		server.Validate = true
		_, _ = w.serverService.Update(server.ID.Hex(), server)
		return true, nil
	} else {
		server.Validate = false
		_, _ = w.serverService.Update(server.ID.Hex(), server)
		return false, nil
	}
}


func (w *ServerStatusUpdateWorker) UpdateLog() error {
	ctx := context.Background()
	esService := serverLogService.NewElasticsearchServerService()
	servers, _, err := w.serverService.GetAll(serverService.Query{})
	if err != nil {
		return err
	}
	var changeLog []string
	currentTime := time.Now().Unix()
	timeStampString := strconv.FormatInt(currentTime, 10)
	for i := 0; i < len(servers); i++ {
		// Check status
		elasticServer, err := esService.Search(ctx, esService.ElasticClient, servers[i].ID.Hex())
		if err != nil {
			return err
		}
		res, err := w.Check(servers[i])

		if err != nil {
			elasticServer.Log += timeStampString + " Off\n"
			servers[i].Status = false
		}

		if res {
			elasticServer.Log += timeStampString + " On\n"
			servers[i].Status = true
		} else {
			elasticServer.Log += timeStampString + " Off\n"
			servers[i].Status = false
		}

		// Validate password
		validateRes, err := w.Validate(servers[i])
		if err != nil {
			servers[i].Validate = false
		}
		if validateRes {
			servers[i].Validate = false
		} else {
			servers[i].Validate = true
		}
		err = esService.Update(ctx, esService.ElasticClient, servers[i].ID.Hex(), elasticServer.Log)
		if err != nil {
			fmt.Println("Failed to update elastic server")
			return err
		}
		_, err = w.serverService.Update(servers[i].ID.Hex(), servers[i])
		if err != nil {
			fmt.Println("Failed to update server: ", err)
			return err
		}
		logs := strings.Split(elasticServer.Log, "\n")
		if len(logs) >= 3 {
			if strings.Split(logs[len(logs)-2], " ")[1] != strings.Split(logs[len(logs)-3], " ")[1] {
				changeLog = append(changeLog, servers[i].Ip+": "+logs[len(logs)-2])
			}
		}
	}

	if len(changeLog) > 0 {
		utils.SendEmail(changeLog)
	}
	return nil
}

func (w *ServerStatusUpdateWorker) ExecuteCronJob() {
	ticker := time.NewTicker(60 * time.Second)
	for range ticker.C {
		w.UpdateLog()
	}
}
