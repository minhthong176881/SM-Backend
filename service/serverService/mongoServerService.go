package serverService

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type MongoServerService struct {
	ServerCollection *mongo.Collection
	UserCollection   *mongo.Collection
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
		ServerCollection: db.Database(dbName).Collection("servers"),
		UserCollection:   db.Database(dbName).Collection("users"),
	}
}

func (inst *MongoServerService) GetAll(query Query) ([]*Server, int64, error) {
	var servers []*Server
	var queryDB bson.M
	if query.Query != "" {
		queryDB = bson.M{
			"$or": []bson.M{
				{"ip": bson.M{"$regex": primitive.Regex{Pattern: query.Query, Options: "i"}}},
				{"name": bson.M{"$regex": primitive.Regex{Pattern: query.Query, Options: "i"}}},
				{"port": bson.M{"$regex": primitive.Regex{Pattern: query.Query, Options: "i"}}},
				{"username": bson.M{"$regex": primitive.Regex{Pattern: query.Query, Options: "i"}}},
				{"password": bson.M{"$regex": primitive.Regex{Pattern: query.Query, Options: "i"}}},
				{"description": bson.M{"$regex": primitive.Regex{Pattern: query.Query, Options: "i"}}},
			},
		}
	} else {
		queryDB = bson.M{}
	}
	skip := (query.PageIndex - 1) * query.PageOffset
	opts := options.FindOptions{
		Skip:  &skip,
		Limit: &query.PageOffset,
		Sort: bson.M{"created_at": -1},
	}
	total, err := inst.ServerCollection.CountDocuments(context.Background(), queryDB)
	if err != nil {
		return nil, 0, status.Errorf(codes.Internal, fmt.Sprintf("Unknown internal error: %v", err))
	}
	cursor, err := inst.ServerCollection.Find(context.Background(), queryDB, &opts)
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
		server.Name = data.Name
		server.Port = data.Port
		server.Username = data.Username
		server.Password = data.Password
		server.Description = data.Description
		server.Validate = data.Validate
		server.Status = data.Status
		server.CreatedAt = data.CreatedAt
		server.UpdatedAt = data.UpdatedAt

		servers = append(servers, &server)
	}
	return servers, int64(total), nil
}

func (inst *MongoServerService) Insert(server *Server) (*Server, error) {
	serverExist := inst.CheckServerExist(server.Ip, server.Port)
	if serverExist {
		return nil, status.Errorf(codes.AlreadyExists, fmt.Sprintf("Server %s:%d already exist", server.Ip, server.Port))
	}
	server.CreatedAt = strconv.FormatInt(time.Now().Unix(), 10)
	result, err := inst.ServerCollection.InsertOne(context.Background(), server)
	if err != nil {
		return nil, err
	}

	oid := result.InsertedID.(primitive.ObjectID)
	server.ID = oid
	return server, nil
}

func (inst *MongoServerService) GetById(id string) (*Server, error) {
	var data *Server
	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, fmt.Sprintf("Could not convert the supplied server Id to a MongoDB ObjectId: %v", err))
	}
	result := inst.ServerCollection.FindOne(context.Background(), bson.M{"_id": oid})
	if err := result.Decode(&data); err != nil {
		return nil, status.Errorf(codes.NotFound, fmt.Sprintf("Could not find server with Id %s: %v", id, err))
	}
	return data, nil
}

func (inst *MongoServerService) Update(id string, server *Server) (*Server, error) {
	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, status.Errorf(
			codes.InvalidArgument,
			fmt.Sprintf("Could not convert the supplied server Id to a MongoDB ObjectId: %v", err),
		)
	}
	update := bson.M{
		"ip":          server.Ip,
		"name":        server.Name,
		"port":        server.Port,
		"username":    server.Username,
		"password":    server.Password,
		"description": server.Description,
		"status":      server.Status,
		"validate":    server.Validate,
		"created_at":  server.CreatedAt,
		"updated_at":  server.UpdatedAt,
	}
	filter := bson.M{"_id": oid}
	result := inst.ServerCollection.FindOneAndUpdate(context.Background(), filter, bson.M{"$set": update}, options.FindOneAndUpdate().SetReturnDocument(1))
	decoded := Server{}
	err = result.Decode(&decoded)
	if err != nil {
		return nil, status.Errorf(
			codes.NotFound,
			fmt.Sprintf("Could not find server with Id: %v", err),
		)
	}
	return &decoded, nil
}

func (inst *MongoServerService) Delete(id string) error {
	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return status.Errorf(codes.InvalidArgument, fmt.Sprintf("Could not convert the supplied server Id to a MongoDB ObjectId: %v", err))
	}
	_, err = inst.ServerCollection.DeleteOne(context.Background(), bson.M{"_id": oid})
	return err
}

func (inst *MongoServerService) CheckServerExist(ip string, port int64) (bool) {
	var data *Server
	result := inst.ServerCollection.FindOne(context.Background(), bson.M{"ip": ip, "port": port})
	if err := result.Decode(&data); err != nil {
		return false
	}
	return true
}
