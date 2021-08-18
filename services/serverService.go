package services

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

type User struct {
	ID       primitive.ObjectID `bson:"_id,omitempty"`
	Username string             `bson:"username"`
	Password string             `bson:"password"`
	Email    string             `bson:"email"`
}

type LogItem struct {
	Time   string `json:"time"`
	Status string `json:"status"`
}

type ChangeLogItem struct {
	Start string
	End   string
	Total string
}

type Query struct {
	Query      string `json:"query"`
	PageIndex  int64  `json:"pageIndex"`
	PageOffset int64  `json:"pageOffset"`
}

type ServerService interface {
	Register(user *User) (string, error)
	Login(username string, password string) (bool, error)
	GetAll(query Query) ([]*Server, int64, error)
	GetById(id string) (*Server, error)
	Insert(server *Server) (*Server, error)
	Update(id string, server *Server) (*Server, error)
	Delete(id string) error
	Export() (string, error)
	Check(id string) (bool, error)
	Validate(id string) (bool, error)
	GetLog(id string, start string, end string, date string, month string) ([]*LogItem, []*ChangeLogItem, error)
}
