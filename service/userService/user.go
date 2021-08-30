package userService

import (
	"context"
	"fmt"

	"github.com/go-redis/redis/v8"
	"github.com/minhthong176881/Server_Management/middleware"
	"github.com/minhthong176881/Server_Management/service/serverService"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type User struct {
	mongoService *serverService.MongoServerService
	jwtManager   *middleware.JWTManager
	redisClient *redis.Client
}

func NewUser(mongoService *serverService.MongoServerService, jwtManager *middleware.JWTManager) *User {
	redisClient := serverService.NewClient()
	return &User{mongoService: mongoService, jwtManager: jwtManager, redisClient: redisClient}
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

func (u *User) Login(username string, password string) (string, error) {
	result := u.mongoService.UserCollection.FindOne(context.Background(), bson.M{"username": username})
	data := UserItem{}
	if err := result.Decode(&data); err != nil {
		return "", status.Errorf(codes.NotFound, fmt.Sprintf("Could not find user with Username %s: %v", username, err))
	}
	err := bcrypt.CompareHashAndPassword([]byte(data.Password), []byte(password))
	if err != nil {
		return "", status.Errorf(codes.Unauthenticated, fmt.Sprintf("Invalid password for user %s", username))
	}
	err = u.ClearRedisToken()
	if err != nil {
		return "", status.Errorf(codes.Internal, fmt.Sprintf("Internal error %v", err))
	}
	jwtData := middleware.UserItem {
		ID: data.ID,
		Username: data.Username,
		Email: data.Email,
		Role: data.Role,
		Password: data.Password,
	}
	token, err := u.jwtManager.Generate(&jwtData)
	if err != nil {
		return "", status.Errorf(codes.Internal, fmt.Sprintf("Internal error %v", err))
	}
	err = u.redisClient.Set(u.redisClient.Context(), "accessToken", token, 0).Err()
	if err != nil {
		return "", status.Errorf(codes.Internal, fmt.Sprintf("Internal error %v", err))
	}
	return token, nil
}

func (u *User) Logout() (bool, error) {
	err := u.ClearRedisToken()
	if err != nil {
		return false, err
	}
	return true, nil
}

func (u *User) ClearRedisToken() error {
	tokenCache, err := u.redisClient.Get(u.redisClient.Context(), "accessToken").Result()
	if err != nil && (err.Error() != string(redis.Nil)) {
		return err
	}
	if tokenCache != "" {
		err = u.redisClient.Del(u.redisClient.Context(), "accessToken").Err()
		if err != nil {
			return err
		}
	}
	return nil
}
