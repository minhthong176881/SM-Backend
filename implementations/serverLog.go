package implementations

import (
	"time"
	
	services "github.com/minhthong176881/Server_Management/services"
)

type ServerLog struct {
	serverLog services.ServerLogService
}

func NewServerLog(serverLog services.ServerLogService) *ServerLog {
	return &ServerLog{
		serverLog: serverLog,
	}
}

func (s *ServerLog) GetLog(id string, start string, end string, date string, month string) ([]*services.LogItem, []*services.ChangeLogItem, error) {
	return s.serverLog.GetLog(id, start, end, date, month)
}

func (s *ServerLog) UpdateLog() error {
	return s.serverLog.UpdateLog()
}

func (s *ServerLog) ExecuteCronJob() {
	ticker := time.NewTicker(60 * time.Second)
	for range ticker.C {
		s.UpdateLog()
		// if time.Now().Hour() == 18 && time.Now().Minute() == 0 {
		// 	SendEmail()
		// }
	}
}
