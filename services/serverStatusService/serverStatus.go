package serverStatusService

import (
	"github.com/minhthong176881/Server_Management/services/serverService"
)

type ServerStatus struct {
	serverStatusService ServerStatusService
}

func NewServerStatus(serverStatusService ServerStatusService) *ServerStatus {
	return &ServerStatus{
		serverStatusService: serverStatusService,
	}
}

func (s *ServerStatus) Export() (string, error) {
	return s.serverStatusService.Export()
}

func (s *ServerStatus) Check(server *serverService.Server) (bool, error) {
	return s.serverStatusService.Check(server)
}

func (s *ServerStatus) Validate(server *serverService.Server) (bool, error) {
	return s.serverStatusService.Validate(server)
}

