package serverStatusService

import (
	"github.com/minhthong176881/Server_Management/service/serverService"
)

type ServerStatusService interface {
	Check(server *serverService.Server) (bool, error)
	Validate(server *serverService.Server) (bool, error)
}