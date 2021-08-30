package serverLogService

import (
	"strings"

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
