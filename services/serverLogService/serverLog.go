package serverLogService

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/minhthong176881/Server_Management/services/serverService"
	"github.com/minhthong176881/Server_Management/services/serverStatusService"
	"github.com/minhthong176881/Server_Management/utils"
)

type ServerLog struct {
	serverLog ServerLogService
}

func NewServerLog(serverLog ServerLogService) *ServerLog {
	return &ServerLog{
		serverLog: serverLog,
	}
}

func (s *ServerLog) GetLog(id string, start string, end string, date string, month string) ([]*LogItem, []*ChangeLogItem, error) {
	return s.serverLog.GetLog(id, start, end, date, month)
}

func UpdateLog() error {
	ctx := context.Background()
	baseService := serverService.NewMongoServerService()
	esService := NewElasticsearchServerService(baseService)
	serverStatus := serverStatusService.NewServerStatus(baseService)
	servers, _, err := baseService.GetAll(serverService.Query{})
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
		res, err := serverStatus.Check(servers[i].ID.Hex())

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
		validateRes, err := serverStatus.Validate(servers[i].ID.Hex())
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
		_, err = baseService.Update(servers[i].ID.Hex(), servers[i])
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

func ExecuteCronJob() {
	ticker := time.NewTicker(60 * time.Second)
	for range ticker.C {
		UpdateLog()
	}
}

func GetChangeLog(logs []*LogItem, changeLogs []*ChangeLogItem) []*ChangeLogItem {
	var startIndex, endIndex int
	var start, end string
	var recursive []*LogItem
	var countOff, countOn int
	if len(logs) <= 0 {
		return changeLogs
	}

	for i := 0; i < len(logs); i++ {
		if logs[i].Status == "Off" {
			countOff++
			start = logs[i].Time
			startIndex = i
			break
		}
	}
	if countOff == 0 {
		return changeLogs
	}

	for i := startIndex + 1; i < len(logs); i++ {
		if logs[i].Status == "On" {
			countOn++
			end = logs[i].Time
			endIndex = i
			break
		}
	}
	if countOn == 0 {
		end = logs[len(logs)-1].Time
		newChangeLog := &ChangeLogItem{}
		newChangeLog.Start = start
		newChangeLog.End = end
		newChangeLog.Total = utils.CalculateTimeDiff(strings.Split(utils.FormatTime(start), " ")[1], strings.Split(utils.FormatTime(end), " ")[1])
		changeLogs = append(changeLogs, newChangeLog)
		return changeLogs
	}

	newChangeLog := &ChangeLogItem{}
	newChangeLog.Start = logs[startIndex].Time
	newChangeLog.End = logs[endIndex].Time
	newChangeLog.Total = utils.CalculateTimeDiff(strings.Split(utils.FormatTime(start), " ")[1], strings.Split(utils.FormatTime(end), " ")[1])
	changeLogs = append(changeLogs, newChangeLog)
	for i := endIndex; i < len(logs); i++ {
		recursive = append(recursive, logs[i])
	}
	return GetChangeLog(recursive, changeLogs)
}
