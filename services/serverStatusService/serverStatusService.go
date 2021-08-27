package serverStatusService

import (
	"github.com/minhthong176881/Server_Management/services/serverService"
)

type ServerStatusService interface {
	Export() (string, error)
	Check(server *serverService.Server) (bool, error)
	Validate(server *serverService.Server) (bool, error)
}