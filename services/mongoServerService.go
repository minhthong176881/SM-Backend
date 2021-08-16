package services

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type MongoServerService struct {
	serverCollection *mongo.Collection
	userCollection   *mongo.Collection
}

func NewMongoServerService() *MongoServerService {
	err := godotenv.Load(".env")
	if err != nil {
		log.Fatalf("Error loading .env file")
	}
	username, password, dbName := os.Getenv("DB_USERNAME"), os.Getenv("DB_PASSWORD"), os.Getenv("DB_NAME")
	db, err := mongo.Connect(context.Background(), options.Client().ApplyURI("mongodb+srv://"+username+":"+password+"@cluster0.ipled.mongodb.net/"+dbName+"?retryWrites=true&w=majority"))
	if err != nil {
		log.Fatal(err)
	}
	err = db.Ping(context.Background(), nil)
	if err != nil {
		log.Fatalf("Could not connect to MongoDB: %v\n", err)
	} else {
		log.Println("Connected to Mongodb")
	}
	return &MongoServerService{
		serverCollection: db.Database(dbName).Collection("servers"),
		userCollection:   db.Database(dbName).Collection("users"),
	}
}

func (mongo *MongoServerService) Register(user *User) (string, error) {
	result, err := mongo.userCollection.InsertOne(context.Background(), user)
	if err != nil {
		return "", err
	}
	return result.InsertedID.(primitive.ObjectID).Hex(), nil
}

func (mongo *MongoServerService) Login(username string, password string) (bool, error) {
	result := mongo.userCollection.FindOne(context.Background(), bson.M{"username": username})
	data := User{}
	if err := result.Decode(&data); err != nil {
		return false, status.Errorf(codes.NotFound, fmt.Sprintf("Could not find user with Username %s: %v", username, err))
	}
	if data.Password == password {
		return true, nil
	}
	return false, status.Error(codes.NotFound, "Username or Password is incorrect!")
}

func (mongo *MongoServerService) GetAll(query bson.M, skip int64, limit int64) ([]*Server, int64, error) {
	var servers []*Server
	opts := options.FindOptions{
		Skip:  &skip,
		Limit: &limit,
	}
	total, err := mongo.serverCollection.CountDocuments(context.Background(), query)
	if err != nil {
		return nil, 0, status.Errorf(codes.Internal, fmt.Sprintf("Unknown internal error: %v", err))
	}
	cursor, err := mongo.serverCollection.Find(context.Background(), query, &opts)
	if err != nil {
		return nil, 0, status.Errorf(codes.Internal, fmt.Sprintf("Unknown internal error: %v", err))
	}
	data := &Server{}
	defer cursor.Close(context.Background())
	for cursor.Next(context.Background()) {
		err := cursor.Decode(data)
		if err != nil {
			return nil, 0, status.Errorf(codes.Unavailable, fmt.Sprintf("Could not decode data: %v", err))
		}
		server := Server{}
		server.ID = data.ID
		server.Ip = data.Ip
		server.Port = data.Port
		server.Username = data.Username
		server.Password = data.Password
		server.Description = data.Description
		server.Validate = data.Validate
		server.Status = data.Status

		servers = append(servers, &server)
	}
	return servers, int64(total), nil
}

func (mongo *MongoServerService) Insert(server *Server) (string, error) {
	result, err := mongo.serverCollection.InsertOne(context.Background(), server)
	if err != nil {
		return "", err
	}

	oid := result.InsertedID.(primitive.ObjectID)
	return oid.Hex(), nil
}

func (mongo *MongoServerService) GetById(id string) (*Server, error) {
	var data *Server
	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, fmt.Sprintf("Could not convert the supplied server Id to a MongoDB ObjectId: %v", err))
	}
	result := mongo.serverCollection.FindOne(context.Background(), bson.M{"_id": oid})
	if err := result.Decode(&data); err != nil {
		return nil, status.Errorf(codes.NotFound, fmt.Sprintf("Could not find server with Id %s: %v", id, err))
	}
	return data, nil
}

func (mongo *MongoServerService) Update(update bson.M, filter bson.M) (*Server, error) {
	result := mongo.serverCollection.FindOneAndUpdate(context.Background(), filter, bson.M{"$set": update}, options.FindOneAndUpdate().SetReturnDocument(1))
	decoded := Server{}
	err := result.Decode(&decoded)
	if err != nil {
		return nil, status.Errorf(
			codes.NotFound,
			fmt.Sprintf("Could not find server with Id: %v", err),
		)
	}
	return &decoded, nil
}

func (mongo *MongoServerService) Delete(id string) error {
	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return status.Errorf(codes.InvalidArgument, fmt.Sprintf("Could not convert the supplied server Id to a MongoDB ObjectId: %v", err))
	}
	_, err = mongo.serverCollection.DeleteOne(context.Background(), bson.M{"_id": oid})
	return err
}
