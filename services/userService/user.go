package userService

import (
	"context"
	"fmt"

	serverService "github.com/minhthong176881/Server_Management/services/serverService"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type User struct {
	mongoService *serverService.MongoServerService
}

func NewUser(mongoService *serverService.MongoServerService) *User {
	return &User{mongoService: mongoService}
}

func (u *User) Register(user *UserItem) (string, error) {
	result, err := u.mongoService.UserCollection.InsertOne(context.Background(), user)
	if err != nil {
		return "", err
	}
	return result.InsertedID.(primitive.ObjectID).Hex(), nil
}

func (u *User) Login(username string, password string) (bool, error) {
	result := u.mongoService.UserCollection.FindOne(context.Background(), bson.M{"username": username})
	data := UserItem{}
	if err := result.Decode(&data); err != nil {
		return false, status.Errorf(codes.NotFound, fmt.Sprintf("Could not find user with Username %s: %v", username, err))
	}
	if data.Password == password {
		return true, nil
	}
	return false, status.Error(codes.NotFound, "Username or Password is incorrect!")
}