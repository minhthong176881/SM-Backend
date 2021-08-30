package serverService

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Server struct {
	ID          primitive.ObjectID `bson:"_id,omitempty"`
	Ip          string             `bson:"ip"`
	Port        int64              `bson:"port"`
	Username    string             `bson:"username"`
	Password    string             `bson:"password"`
	Description string             `bson:"description"`
	Status      bool               `bson:"status"`
	Validate    bool               `bson:"validate"`
}

type GetAllResponse struct {
	Servers []*Server `json:"servers"`
	Total   int64     `json:"total"`
}

type Query struct {
	Query      string `json:"query"`
	PageIndex  int64  `json:"pageIndex"`
	PageOffset int64  `json:"pageOffset"`
}

type ServerService interface {
	GetAll(query Query) ([]*Server, int64, error)
	GetById(id string) (*Server, error)
	Insert(server *Server) (*Server, error)
	Update(id string, server *Server) (*Server, error)
	Delete(id string) error
}
