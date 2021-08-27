package serverStatusService

type ServerStatusService interface {
	Export() (string, error)
	Check(id string) (bool, error)
	Validate(id string) (bool, error)
}