package serverLogService

type LogItem struct {
	Time   string `json:"time"`
	Status string `json:"status"`
}

type ChangeLogItem struct {
	Start string
	End   string
	Total string
}

type ServerLogService interface {
	GetLog(id string, start string, end string, date string, month string) ([]*LogItem, []*ChangeLogItem, error)
}