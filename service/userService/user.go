package userService

import (
	"context"
	"fmt"

	"github.com/go-redis/redis/v8"
	"github.com/minhthong176881/Server_Management/service/serverService"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type User struct {
	mongoService *serverService.MongoServerService
	redisClient  *redis.Client
}

func NewUser(mongoService *serverService.MongoServerService) *User {
	redisClient := serverService.NewClient()
	return &User{mongoService: mongoService, redisClient: redisClient}
}

func (u *User) Register(user *UserItem) (string, error) {
	hashPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}

	user.Password = string(hashPassword)
	result, err := u.mongoService.UserCollection.InsertOne(context.Background(), user)
	if err != nil {
		return "", err
	}
	return result.InsertedID.(primitive.ObjectID).Hex(), nil
}

func (u *User) Login(username string, password string) (*UserItem, error) {
	result := u.mongoService.UserCollection.FindOne(context.Background(), bson.M{"username": username})
	data := UserItem{}
	if err := result.Decode(&data); err != nil {
		return nil, status.Errorf(codes.NotFound, fmt.Sprintf("Could not find user with Username %s: %v", username, err))
	}
	err := bcrypt.CompareHashAndPassword([]byte(data.Password), []byte(password))
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, fmt.Sprintf("Invalid password for user %s", username))
	}
	return &data, nil
}

func (u *User) Logout(token string) {
	u.redisClient.Set(u.redisClient.Context(), token, "1", 0)
}

