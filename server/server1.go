package server

// import (
// 	"context"
// 	"encoding/json"
// 	"fmt"
// 	"log"
// 	"os"
// 	"reflect"
// 	"regexp"
// 	"strconv"
// 	"strings"

// 	"github.com/360EntSecGroup-Skylar/excelize/v2"
// 	"go.mongodb.org/mongo-driver/bson"
// 	"go.mongodb.org/mongo-driver/bson/primitive"
// 	"go.mongodb.org/mongo-driver/mongo"
// 	"go.mongodb.org/mongo-driver/mongo/options"

// 	// "go.mongodb.org/mongo-driver/x/bsonx"
// 	"github.com/go-redis/redis/v8"
// 	"github.com/joho/godotenv"
// 	elastic "github.com/olivere/elastic/v7"
// 	"google.golang.org/grpc/codes"
// 	"google.golang.org/grpc/status"

// 	pbSM "github.com/minhthong176881/Server_Management/proto"
// 	// services "github.com/minhthong176881/Server_Management/services"
// )

// type Item struct {
// 	ID          primitive.ObjectID `bson:"_id,omitempty"`
// 	Ip          string             `bson:"ip"`
// 	Port        int64              `bson:"port"`
// 	Username    string             `bson:"username"`
// 	Password    string             `bson:"password"`
// 	Description string             `bson:"description"`
// 	Status      bool               `bson:"status"`
// 	Validate    bool               `bson:"validate"`
// }

// type LogItem struct {
// 	Time   string `json:"time"`
// 	Status string `json:"status"`
// }

// type UserItem struct {
// 	ID       primitive.ObjectID `bson:"_id,omitempty"`
// 	Username string             `bson:"username"`
// 	Password string             `bson:"password"`
// 	Email    string             `bson:"email"`
// }

// // Backend implements the protobuf interface
// type Backend struct {
// 	// db	*mongo.Client
// 	serverCollection *mongo.Collection
// 	userCollection   *mongo.Collection
// 	mongoCtx         context.Context
// 	esClient         *elastic.Client
// 	redisClient      *redis.Client
// }

// // New initializes a new Backend struct.
// func New() *Backend {
// 	err := godotenv.Load(".env")
// 	if err != nil {
// 		log.Fatalf("Error loading .env file")
// 	}
// 	username, password, dbName := os.Getenv("DB_USERNAME"), os.Getenv("DB_PASSWORD"), os.Getenv("DB_NAME")
// 	db, err := mongo.Connect(context.Background(), options.Client().ApplyURI("mongodb+srv://"+username+":"+password+"@cluster0.ipled.mongodb.net/"+dbName+"?retryWrites=true&w=majority"))
// 	if err != nil {
// 		log.Fatal(err)
// 	}
// 	err = db.Ping(context.Background(), nil)
// 	if err != nil {
// 		log.Fatalf("Could not connect to MongoDB: %v\n", err)
// 	} else {
// 		log.Println("Connected to Mongodb")
// 	}

// 	elasticClient, err := GetESClient()
// 	if err != nil {
// 		log.Fatalf("Could not connect to Elasticsearch: %v\n", err)
// 	}

// 	redisClient := newClient()
// 	result, err := ping(redisClient)
// 	if err != nil {
// 		log.Fatal(err)
// 	} else {
// 		log.Println(result)
// 	}

// 	return &Backend{
// 		mongoCtx:         context.Background(),
// 		serverCollection: db.Database(dbName).Collection("servers"),
// 		userCollection:   db.Database(dbName).Collection("users"),
// 		esClient:         elasticClient,
// 		redisClient:      redisClient,
// 	}
// }

// func (b *Backend) Register(_ context.Context, req *pbSM.RegisterRequest) (*pbSM.User, error) {
// 	user := req.GetUser()

// 	data := UserItem{
// 		Username: req.User.Username,
// 		Password: req.User.Password,
// 		Email:    req.User.Email,
// 	}
// 	result, err := b.userCollection.InsertOne(b.mongoCtx, data)
// 	if err != nil {
// 		return nil, nil
// 	}

// 	oid := result.InsertedID.(primitive.ObjectID)
// 	user.Id = oid.Hex()

// 	return user, nil
// }

// func (b *Backend) Login(ctx context.Context, req *pbSM.LoginRequest) (*pbSM.LoginResponse, error) {
// 	username, password := req.GetUsername(), req.GetPassword()
// 	result := b.userCollection.FindOne(ctx, bson.M{"username": username})
// 	data := UserItem{}
// 	if err := result.Decode(&data); err != nil {
// 		return &pbSM.LoginResponse{Logged: false}, status.Errorf(codes.NotFound, fmt.Sprintf("Could not find user with Username %s: %v", req.GetUsername(), err))
// 	}
// 	if data.Password == password {
// 		return &pbSM.LoginResponse{Logged: true}, nil
// 	}
// 	return &pbSM.LoginResponse{Logged: false}, status.Error(codes.NotFound, "Username or Password is incorrect!")
// }

// func (b *Backend) GetServers(_ context.Context, req *pbSM.GetServersRequest) (*pbSM.GetServersResponse, error) {
// 	pageIndex, pageOffset := req.GetPageIndex(), req.GetPageOffset()
// 	var servers []*pbSM.Server
// 	skip := (pageIndex - 1) * pageOffset
// 	opts := options.FindOptions{
// 		Skip:  &skip,
// 		Limit: &pageOffset,
// 	}
// 	var query bson.M
// 	if req.GetQuery() != "" {
// 		query = bson.M{
// 			"$or": []bson.M{
// 				{"ip": bson.M{"$regex": primitive.Regex{Pattern: req.GetQuery(), Options: "i"}}},
// 				{"port": bson.M{"$regex": primitive.Regex{Pattern: req.GetQuery(), Options: "i"}}},
// 				{"username": bson.M{"$regex": primitive.Regex{Pattern: req.GetQuery(), Options: "i"}}},
// 				{"password": bson.M{"$regex": primitive.Regex{Pattern: req.GetQuery(), Options: "i"}}},
// 				{"description": bson.M{"$regex": primitive.Regex{Pattern: req.GetQuery(), Options: "i"}}},
// 			},
// 		}
// 	} else {
// 		query = bson.M{}
// 	}
// 	queryJson, _ := bson.Marshal(query)
// 	option := "skip=" + strconv.Itoa(int(skip)) + "&offset=" + strconv.Itoa(int(pageOffset))
// 	key := string(queryJson) + option
// 	total, err := b.serverCollection.CountDocuments(context.Background(), query)
// 	if err != nil {
// 		return nil, status.Errorf(codes.Internal, fmt.Sprintf("Unknown internal error: %v", err))
// 	}
// 	serverRedis, err := b.redisClient.Get(b.redisClient.Context(), key).Result()
// 	if err == redis.Nil {
// 		log.Printf("Key %s does not exist", key)
// 		cursor, err := b.serverCollection.Find(context.Background(), query, &opts)
// 		if err != nil {
// 			return nil, status.Errorf(codes.Internal, fmt.Sprintf("Unknown internal error: %v", err))
// 		}
// 		data := &Item{}
// 		defer cursor.Close(context.Background())
// 		for cursor.Next(context.Background()) {
// 			err := cursor.Decode(data)
// 			if err != nil {
// 				return nil, status.Errorf(codes.Unavailable, fmt.Sprintf("Could not decode data: %v", err))
// 			}
// 			server := pbSM.Server{}
// 			server.Id = data.ID.Hex()
// 			server.Ip = data.Ip
// 			server.Port = data.Port
// 			server.Username = data.Username
// 			server.Password = data.Password
// 			server.Description = data.Description
// 			server.Validate = data.Validate
// 			server.Status = data.Status

// 			servers = append(servers, &server)
// 		}
// 		redisVal, err := json.Marshal(servers)
// 		if err == nil {
// 			b.redisClient.Set(b.redisClient.Context(), key, redisVal, 0)
// 			log.Println("Set key to redis successfully")
// 		}
// 		return &pbSM.GetServersResponse{Servers: servers, Total: total}, nil
// 	} else if serverRedis != string(redis.Nil) {
// 		var redisRes []*pbSM.Server
// 		err = json.Unmarshal([]byte(serverRedis), &redisRes)
// 		if err != nil {
// 			return nil, err
// 		}
// 		return &pbSM.GetServersResponse{Servers: redisRes, Total: total}, nil
// 	} else {
// 		return nil, err
// 	}
// }

// func (b *Backend) AddServer(ctx context.Context, req *pbSM.AddServerRequest) (*pbSM.Server, error) {
// 	server := req.GetServer()

// 	if server.GetIp() == "" || server.GetPort() == 0 || server.GetUsername() == "" || server.GetPassword() == "" {
// 		return nil, status.Error(
// 			codes.InvalidArgument,
// 			"Invalid payload!",
// 		)
// 	}

// 	data := Item{
// 		Ip:          req.Server.Ip,
// 		Port:        req.Server.Port,
// 		Username:    req.Server.Username,
// 		Password:    req.Server.Password,
// 		Description: req.Server.Description,
// 		Status:      true,
// 		Validate:    true,
// 	}

// 	result, err := b.serverCollection.InsertOne(b.mongoCtx, data)
// 	if err != nil {
// 		return nil, err
// 	}

// 	oid := result.InsertedID.(primitive.ObjectID)
// 	server.Id = oid.Hex()

// 	err = Insert(ctx, b.esClient, ElasticsearchServer{ServerId: server.Id, Log: ""})
// 	if err != nil {
// 		log.Println("Cannot insert server to elasticsearch")
// 	}
// 	b.redisClient.FlushAll(ctx)

// 	return server, nil
// }

// func (b *Backend) GetServerById(ctx context.Context, req *pbSM.GetServerByIdRequest) (*pbSM.Server, error) {
// 	oid, err := primitive.ObjectIDFromHex(req.GetId())
// 	if err != nil {
// 		return nil, status.Errorf(codes.InvalidArgument, fmt.Sprintf("Could not convert the supplied server Id to a MongoDB ObjectId: %v", err))
// 	}
// 	data := Item{}

// 	serverRedis, err := b.redisClient.Get(b.redisClient.Context(), oid.Hex()).Result()
// 	if err == redis.Nil {
// 		log.Printf("Key %s does not exist", oid.Hex())
// 		result := b.serverCollection.FindOne(ctx, bson.M{"_id": oid})
// 		if err := result.Decode(&data); err != nil {
// 			return nil, status.Errorf(codes.NotFound, fmt.Sprintf("Could not find server with Id %s: %v", req.GetId(), err))
// 		}
// 		redisVal, err := json.Marshal(data)
// 		if err == nil {
// 			b.redisClient.Set(b.redisClient.Context(), oid.Hex(), redisVal, 0)
// 			log.Println("Set key to redis successfully")
// 		}
// 	} else if serverRedis != string(redis.Nil) {
// 		err = json.Unmarshal([]byte(serverRedis), &data)
// 		if err != nil {
// 			return nil, err
// 		}
// 	} else {
// 		return nil, err
// 	}

// 	response := pbSM.Server{}
// 	response.Id = oid.Hex()
// 	response.Ip = data.Ip
// 	response.Port = data.Port
// 	response.Username = data.Username
// 	response.Password = data.Password
// 	response.Description = data.Description
// 	response.Status = data.Status
// 	response.Validate = data.Validate
// 	return &response, nil
// }

// func (b *Backend) UpdateServer(ctx context.Context, req *pbSM.UpdateServerRequest) (*pbSM.Server, error) {
// 	server := req.GetServer()
// 	oid, err := primitive.ObjectIDFromHex(req.GetId())
// 	currentServer, _ := b.GetServerById(ctx, &pbSM.GetServerByIdRequest{Id: req.GetId()})
// 	if !reflect.DeepEqual(currentServer, server) {
// 		flushServer(b.redisClient, oid.Hex())
// 	}
// 	if err != nil {
// 		return nil, status.Errorf(
// 			codes.InvalidArgument,
// 			fmt.Sprintf("Could not convert the supplied server Id to a MongoDB ObjectId: %v", err),
// 		)
// 	}
// 	if server.GetIp() == "" || server.GetPort() == 0 || server.GetUsername() == "" || server.GetPassword() == "" {
// 		return nil, status.Error(
// 			codes.InvalidArgument,
// 			"Invalid payload!",
// 		)
// 	}
// 	update := bson.M{
// 		"ip":          server.GetIp(),
// 		"port":        server.GetPort(),
// 		"username":    server.GetUsername(),
// 		"password":    server.GetPassword(),
// 		"description": server.GetDescription(),
// 		"status":      server.GetStatus(),
// 		"validate":    server.GetValidate(),
// 	}
// 	filter := bson.M{"_id": oid}
// 	result := b.serverCollection.FindOneAndUpdate(ctx, filter, bson.M{"$set": update}, options.FindOneAndUpdate().SetReturnDocument(1))
// 	decoded := Item{}
// 	err = result.Decode(&decoded)
// 	if err != nil {
// 		return nil, status.Errorf(
// 			codes.NotFound,
// 			fmt.Sprintf("Could not find server with Id: %v", err),
// 		)
// 	}
// 	response := pbSM.Server{}
// 	response.Id = decoded.ID.Hex()
// 	response.Ip = decoded.Ip
// 	response.Port = decoded.Port
// 	response.Username = decoded.Username
// 	response.Password = decoded.Password
// 	response.Status = decoded.Status
// 	response.Validate = decoded.Validate
// 	response.Description = decoded.Description
// 	return &response, nil
// }

// func (b *Backend) DeleteServer(ctx context.Context, req *pbSM.GetServerByIdRequest) (*pbSM.DeleteServerResponse, error) {
// 	oid, err := primitive.ObjectIDFromHex(req.GetId())
// 	if err != nil {
// 		return nil, status.Errorf(codes.InvalidArgument, fmt.Sprintf("Could not convert the supplied server Id to a MongoDB ObjectId: %v", err))
// 	}
// 	_, err1 := b.serverCollection.DeleteOne(ctx, bson.M{"_id": oid})
// 	err2 := Delete(ctx, b.esClient, oid.Hex())
// 	if err1 != nil {
// 		return &pbSM.DeleteServerResponse{Deleted: 0}, status.Errorf(codes.NotFound, fmt.Sprintf("Could not delete server with Id %s: %v", req.GetId(), err1))
// 	} else if err2 != nil {
// 		return &pbSM.DeleteServerResponse{Deleted: 0}, status.Errorf(codes.NotFound, fmt.Sprintf("Could not delete server with Id %s: %v", req.GetId(), err2))
// 	}
// 	flushServer(b.redisClient, oid.Hex())
// 	return &pbSM.DeleteServerResponse{Deleted: 1}, nil
// }

// func (b *Backend) ExportServers(ctx context.Context, req *pbSM.ExportServersRequest) (*pbSM.ExportServersResponse, error) {
// 	var myTableName = "Server list"
// 	f := excelize.NewFile()
// 	f.DeleteSheet("Sheet1")
// 	index := f.NewSheet(myTableName)
// 	_ = f.SetCellValue(myTableName, "A2", "Server")
// 	_ = f.SetCellValue(myTableName, "B2", "IP")
// 	_ = f.SetCellValue(myTableName, "C2", "Username")
// 	_ = f.SetCellValue(myTableName, "D2", "Password")
// 	_ = f.SetCellValue(myTableName, "E2", "Status")
// 	_ = f.SetCellValue(myTableName, "F2", "Password validate")
// 	_ = f.SetCellValue(myTableName, "G2", "Description")

// 	serverResponse, err := b.GetServers(ctx, &pbSM.GetServersRequest{})
// 	if err != nil {
// 		return nil, err
// 	}
// 	servers := serverResponse.Servers
// 	for i := 3; i < len(servers)+3; i++ {
// 		num := strconv.FormatInt(int64(i), 10)
// 		var status string
// 		if servers[i-3].Status {
// 			status = "On"
// 		} else {
// 			status = "Off"
// 		}
// 		var validate string
// 		if servers[i-3].Validate {
// 			validate = "Valid"
// 		} else {
// 			validate = "Invalid"
// 		}
// 		_ = f.SetCellValue(myTableName, "A"+num, i-2)
// 		_ = f.SetCellValue(myTableName, "B"+num, servers[i-3].Ip)
// 		_ = f.SetCellValue(myTableName, "C"+num, servers[i-3].Username)
// 		_ = f.SetCellValue(myTableName, "D"+num, servers[i-3].Password)
// 		_ = f.SetCellValue(myTableName, "E"+num, status)
// 		_ = f.SetCellValue(myTableName, "F"+num, validate)
// 		_ = f.SetCellValue(myTableName, "G"+num, servers[i-3].Description)
// 	}
// 	f.SetActiveSheet(index)
// 	f.Path = "public/OpenAPI/exports/Server_list.xlsx"
// 	_ = f.Save()

// 	err = godotenv.Load(".env")
// 	if err != nil {
// 		log.Fatalf("Error loading .env file")
// 	}
// 	host := os.Getenv("HOST")

// 	return &pbSM.ExportServersResponse{DownloadUrl: host + "/exports/Server_list.xlsx"}, nil
// }

// func (b *Backend) CheckServer(ctx context.Context, req *pbSM.GetServerByIdRequest) (*pbSM.CheckServerResponse, error) {
// 	server, err := b.GetServerById(ctx, &pbSM.GetServerByIdRequest{Id: req.GetId()})
// 	if err != nil {
// 		return nil, status.Errorf(codes.NotFound, fmt.Sprintf("Could not find server with Id %s: %v", req.GetId(), err))
// 	}

// 	conn, err := Connect(server.Ip+":"+strconv.FormatInt(server.Port, 10), server.Username, server.Password)
// 	if err != nil {
// 		if strings.Contains(err.Error(), "ssh") && strings.Contains(err.Error(), "handshake") {
// 			server.Status = true
// 			_, _ = b.UpdateServer(ctx, &pbSM.UpdateServerRequest{Id: server.Id, Server: server})
// 			return &pbSM.CheckServerResponse{Status: true}, nil
// 		}
// 		server.Status = false
// 		_, _ = b.UpdateServer(ctx, &pbSM.UpdateServerRequest{Id: server.Id, Server: server})
// 		return nil, err
// 	}
// 	if conn != nil {
// 		server.Status = true
// 		_, _ = b.UpdateServer(ctx, &pbSM.UpdateServerRequest{Id: server.Id, Server: server})
// 		return &pbSM.CheckServerResponse{Status: true}, nil
// 	} else {
// 		server.Status = false
// 		_, _ = b.UpdateServer(ctx, &pbSM.UpdateServerRequest{Id: server.Id, Server: server})
// 		return &pbSM.CheckServerResponse{Status: false}, nil
// 	}
// }

// func (b *Backend) GetServerLog(ctx context.Context, req *pbSM.GetServerLogRequest) (*pbSM.GetServerLogResponse, error) {
// 	serverRedis, err := b.redisClient.Get(b.redisClient.Context(), req.GetId()+"_log").Result()
// 	var elasticServer ElasticsearchServer
// 	if err == redis.Nil {
// 		log.Printf("Key %s does not exist", req.GetId()+"_log")
// 		elastic, err := Search(ctx, b.esClient, req.GetId())
// 		if err != nil {
// 			return nil, err
// 		}
// 		redisVal, err := json.Marshal(elastic)
// 		if err == nil {
// 			b.redisClient.Set(b.redisClient.Context(), req.GetId()+"_log", redisVal, 0)
// 			log.Println("Set key to redis successfully")
// 		}
// 		elasticServer = elastic
// 	} else if serverRedis != string(redis.Nil) {
// 		err = json.Unmarshal([]byte(serverRedis), &elasticServer)
// 		if err != nil {
// 			return nil, err
// 		}
// 	} else {
// 		return nil, err
// 	}
// 	// elasticServer, err := Search(ctx, b.esClient, req.GetId())
// 	// if err != nil {
// 	// 	return nil, err
// 	// }
// 	logs := strings.Split(elasticServer.Log, "\n")
// 	var startIndex int
// 	var endIndex int
// 	var allLog []*pbSM.ServerLog
// 	var responseLog []*pbSM.ServerLog
// 	var changeLogs []*pbSM.ChangeLog

// 	for i := 0; i < len(logs)-1; i++ {
// 		var log pbSM.ServerLog
// 		log.Time = strings.Split(logs[i], " ")[0]
// 		log.Status = strings.Split(logs[i], " ")[1]
// 		allLog = append(allLog, &log)
// 	}

// 	re := regexp.MustCompile(`^\d{4}\-(0[1-9]|1[012])\-(0[1-9]|[12][0-9]|3[01])$`)
// 	reMonth := regexp.MustCompile(`^\d{4}\-(0[1-9]|1[012])$`)

// 	if req.GetMonth() != "" && reMonth.MatchString(req.GetMonth()) {
// 		for i := 0; i < len(allLog); i++ {
// 			if strings.Contains(FormatTime(allLog[i].Time), req.GetMonth()) {
// 				responseLog = append(responseLog, allLog[i])
// 			}
// 		}
// 		return &pbSM.GetServerLogResponse{Logs: responseLog, ChangeLogs: changeLogs}, nil
// 	} else if req.GetMonth() != "" && !reMonth.MatchString(req.GetMonth()) {
// 		return nil, status.Errorf(codes.InvalidArgument, fmt.Sprintf("Invalid date: %s", req.GetMonth()))
// 	} else {
// 		if req.GetDate() != "" && re.MatchString(req.GetDate()) {
// 			for i := 0; i < len(allLog); i++ {
// 				if strings.Contains(FormatTime(allLog[i].Time), req.GetDate()) {
// 					responseLog = append(responseLog, allLog[i])
// 				}
// 			}
// 			changeLogs := GetChangeLog(responseLog, changeLogs)
// 			return &pbSM.GetServerLogResponse{Logs: responseLog, ChangeLogs: changeLogs}, nil
// 		} else if req.GetDate() != "" && !re.MatchString(req.GetDate()) {
// 			return nil, status.Errorf(codes.InvalidArgument, fmt.Sprintf("Invalid date: %s", req.GetDate()))
// 		} else {
// 			if (!re.MatchString(req.GetStart()) && req.GetStart() != "") || (!re.MatchString(req.GetEnd()) && req.GetEnd() != "") {
// 				if !re.MatchString(req.GetStart()) && req.GetStart() != "" {
// 					return nil, status.Errorf(codes.InvalidArgument, fmt.Sprintf("Invalid date: %s", req.GetStart()))
// 				}
// 				if !re.MatchString(req.GetEnd()) && req.GetEnd() != "" {
// 					return nil, status.Errorf(codes.InvalidArgument, fmt.Sprintf("Invalid date: %s", req.GetEnd()))
// 				}
// 			} else if req.GetStart() != "" && re.MatchString(req.GetStart()) {
// 				for i := 0; i < len(allLog); i++ {
// 					if strings.Contains(FormatTime(allLog[i].Time), req.GetStart()) {
// 						startIndex = i
// 						break
// 					}
// 				}
// 				if req.GetEnd() != "" && re.MatchString(req.GetEnd()) {
// 					if !CheckValidTimeRange(req.GetStart(), req.GetEnd()) {
// 						return nil, status.Errorf(codes.InvalidArgument, fmt.Sprintf("Invalid date: %s > %s", req.GetStart(), req.GetEnd()))
// 					}
// 					for i := len(allLog) - 1; i >= 0; i-- {
// 						if strings.Contains(FormatTime(allLog[i].Time), req.GetEnd()) {
// 							endIndex = i
// 							break
// 						}
// 					}
// 					for i := startIndex; i <= endIndex; i++ {
// 						responseLog = append(responseLog, allLog[i])

// 					}
// 					return &pbSM.GetServerLogResponse{Logs: responseLog, ChangeLogs: changeLogs}, nil
// 				} else {
// 					for i := startIndex; i < len(allLog); i++ {
// 						responseLog = append(responseLog, allLog[i])
// 					}
// 					return &pbSM.GetServerLogResponse{Logs: responseLog, ChangeLogs: changeLogs}, nil
// 				}
// 			} else if req.GetStart() == "" && req.GetEnd() != "" && re.MatchString(req.GetEnd()) {
// 				startIndex = 0
// 				for i := len(allLog) - 1; i >= 0; i-- {
// 					if strings.Contains(FormatTime(allLog[i].Time), req.GetEnd()) {
// 						endIndex = i
// 						break
// 					} else {
// 						continue
// 					}
// 				}
// 				for i := 0; i <= endIndex; i++ {
// 					responseLog = append(responseLog, allLog[i])

// 				}
// 				return &pbSM.GetServerLogResponse{Logs: responseLog, ChangeLogs: changeLogs}, nil
// 			}
// 			return &pbSM.GetServerLogResponse{Logs: allLog, ChangeLogs: changeLogs}, nil
// 		}
// 	}
// }

// func (b *Backend) ValidateServer(ctx context.Context, req *pbSM.GetServerByIdRequest) (*pbSM.ValidateServerResponse, error) {
// 	server, err := b.GetServerById(ctx, &pbSM.GetServerByIdRequest{Id: req.GetId()})
// 	if err != nil {
// 		server.Validate = false
// 		_, _ = b.UpdateServer(ctx, &pbSM.UpdateServerRequest{Id: server.Id, Server: server})
// 		return nil, status.Errorf(codes.NotFound, fmt.Sprintf("Could not find server with Id %s: %v", req.GetId(), err))
// 	}
// 	conn, err := Connect(server.Ip+":"+strconv.FormatInt(server.Port, 10), server.Username, server.Password)
// 	if err != nil {
// 		server.Validate = false
// 		_, _ = b.UpdateServer(ctx, &pbSM.UpdateServerRequest{Id: server.Id, Server: server})
// 		return nil, err
// 	}
// 	if conn != nil {
// 		server.Validate = true
// 		_, _ = b.UpdateServer(ctx, &pbSM.UpdateServerRequest{Id: server.Id, Server: server})
// 		return &pbSM.ValidateServerResponse{Validated: true}, nil
// 	} else {
// 		server.Validate = false
// 		_, _ = b.UpdateServer(ctx, &pbSM.UpdateServerRequest{Id: server.Id, Server: server})
// 		return &pbSM.ValidateServerResponse{Validated: false}, nil
// 	}
// }
