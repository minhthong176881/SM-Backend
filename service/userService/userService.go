package userService

import "go.mongodb.org/mongo-driver/bson/primitive"

type UserItem struct {
	ID       primitive.ObjectID `bson:"_id,omitempty"`
	Username string             `bson:"username"`
	Password string             `bson:"password"`
	Email    string             `bson:"email"`
	Role     string             `bson:"role"`
}

type UserService interface {
	Register(user *UserItem) (string, error)
	Login(username string, password string) (string, error)
	Logout() (bool, error)
}
