package serverService

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

// func (inst *MongoServerService) Register(user *User) (string, error) {
// 	result, err := inst.userCollection.InsertOne(context.Background(), user)
// 	if err != nil {
// 		return "", err
// 	}
// 	return result.InsertedID.(primitive.ObjectID).Hex(), nil
// }

// func (inst *MongoServerService) Login(username string, password string) (bool, error) {
// 	result := inst.userCollection.FindOne(context.Background(), bson.M{"username": username})
// 	data := User{}
// 	if err := result.Decode(&data); err != nil {
// 		return false, status.Errorf(codes.NotFound, fmt.Sprintf("Could not find user with Username %s: %v", username, err))
// 	}
// 	if data.Password == password {
// 		return true, nil
// 	}
// 	return false, status.Error(codes.NotFound, "Username or Password is incorrect!")
// }

func (inst *MongoServerService) GetAll(query Query) ([]*Server, int64, error) {
	var servers []*Server
	var queryDB bson.M
	if query.Query != "" {
		queryDB = bson.M{
			"$or": []bson.M{
				{"ip": bson.M{"$regex": primitive.Regex{Pattern: query.Query, Options: "i"}}},
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

func (inst *MongoServerService) Insert(server *Server) (*Server, error) {
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
		"port":        server.Port,
		"username":    server.Username,
		"password":    server.Password,
		"description": server.Description,
		"status":      server.Status,
		"validate":    server.Validate,
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

// func (inst *MongoServerService) Export() (string, error) {
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

// 	servers, _, err := inst.GetAll(Query{})
// 	if err != nil {
// 		return "", err
// 	}
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
// 	return host + "/exports/Server_list.xlsx", nil
// }

// func (inst *MongoServerService) Check(id string) (bool, error) {
// 	server, err := inst.GetById(id)
// 	if err != nil {
// 		return false, status.Errorf(codes.NotFound, fmt.Sprintf("Could not find server with Id %s: %v", id, err))
// 	}

// 	conn, err := Connect(server.Ip+":"+strconv.FormatInt(server.Port, 10), server.Username, server.Password)
// 	if err != nil {
// 		if strings.Contains(err.Error(), "ssh") && strings.Contains(err.Error(), "handshake") {
// 			server.Status = true
// 			_, _ = inst.Update(id, server)
// 			return true, nil
// 		}
// 		server.Status = false
// 		_, _ = inst.Update(id, server)
// 		return false, err
// 	}
// 	if conn != nil {
// 		server.Status = true
// 		_, _ = inst.Update(id, server)
// 		return true, nil
// 	} else {
// 		server.Status = false
// 		_, _ = inst.Update(id, server)
// 		return false, nil
// 	}
// }

// func (inst *MongoServerService) Validate(id string) (bool, error) {
// 	server, err := inst.GetById(id)
// 	if err != nil {
// 		server.Validate = false
// 		_, _ = inst.Update(id, server)
// 		return false, status.Errorf(codes.NotFound, fmt.Sprintf("Could not find server with Id %s: %v", id, err))
// 	}

// 	conn, err := Connect(server.Ip+":"+strconv.FormatInt(server.Port, 10), server.Username, server.Password)
// 	if err != nil {
// 		server.Validate = false
// 		_, _ = inst.Update(id, server)
// 		return false, err
// 	}
// 	if conn != nil {
// 		server.Validate = true
// 		_, _ = inst.Update(id, server)
// 		return true, nil
// 	} else {
// 		server.Validate = false
// 		_, _ = inst.Update(id, server)
// 		return false, nil
// 	}
// }

// func (inst *MongoServerService) GetLog(id string, start string, end string, date string, month string) ([]*LogItem, []*ChangeLogItem, error) {
// 	// serverRedis, err := ss.RedisService.redisClient.Get(ss.RedisService.redisClient.Context(), id+"_log").Result()
// 	// var elasticServer ElasticsearchServer
// 	// if (err != nil && (err.Error() == string(redis.Nil))) || serverRedis == "" {
// 	// 	log.Printf("Key %s does not exist", id+"_log")
// 	// 	elastic, err := ss.ElasticsearchService.Search(context.Background(), ss.ElasticsearchService.elasticClient, id)
// 	// 	if err != nil {
// 	// 		return nil, nil, err
// 	// 	}
// 	// 	redisVal, err := json.Marshal(elastic)
// 	// 	if err == nil {
// 	// 		ss.RedisService.redisClient.Set(ss.RedisService.redisClient.Context(), id+"_log", redisVal, 0)
// 	// 		log.Println("Set key to redis successfully")
// 	// 	}
// 	// 	elasticServer = elastic
// 	// } else if serverRedis != string(redis.Nil) {
// 	// 	err = json.Unmarshal([]byte(serverRedis), &elasticServer)
// 	// 	if err != nil {
// 	// 		return nil, nil, err
// 	// 	}
// 	// } else {
// 	// 	return nil, nil, err
// 	// }
// 	// elasticServer, err := Search(ctx, b.esClient, id)
// 	// if err != nil {
// 	// 	return nil, err
// 	// }
// 	// logs := strings.Split(elasticServer.Log, "\n")
// 	// var startIndex int
// 	// var endIndex int
// 	// var allLog []*LogItem
// 	// var responseLog []*LogItem
// 	// var changeLogs []*ChangeLogItem
// 	// for i := 0; i < len(logs)-1; i++ {
// 	// 	var log LogItem
// 	// 	log.Time = strings.Split(logs[i], " ")[0]
// 	// 	log.Status = strings.Split(logs[i], " ")[1]
// 	// 	allLog = append(allLog, &log)
// 	// }

// 	// re := regexp.MustCompile(`^\d{4}\-(0[1-9]|1[012])\-(0[1-9]|[12][0-9]|3[01])$`)
// 	// reMonth := regexp.MustCompile(`^\d{4}\-(0[1-9]|1[012])$`)

// 	// if month != "" && reMonth.MatchString(month) {
// 	// 	for i := 0; i < len(allLog); i++ {
// 	// 		if strings.Contains(FormatTime(allLog[i].Time), month) {
// 	// 			responseLog = append(responseLog, allLog[i])
// 	// 		}
// 	// 	}
// 	// 	return responseLog, changeLogs, nil
// 	// } else if month != "" && !reMonth.MatchString(month) {
// 	// 	return nil, nil, status.Errorf(codes.InvalidArgument, fmt.Sprintf("Invalid date: %s", month))
// 	// } else {
// 	// 	if date != "" && re.MatchString(date) {
// 	// 		for i := 0; i < len(allLog); i++ {
// 	// 			if strings.Contains(FormatTime(allLog[i].Time), date) {
// 	// 				responseLog = append(responseLog, allLog[i])
// 	// 			}
// 	// 		}
// 	// 		changeLogs := GetChangeLog(responseLog, changeLogs)
// 	// 		return responseLog, changeLogs, nil
// 	// 	} else if date != "" && !re.MatchString(date) {
// 	// 		return nil, nil, status.Errorf(codes.InvalidArgument, fmt.Sprintf("Invalid date: %s", date))
// 	// 	} else {
// 	// 		if (!re.MatchString(start) && start != "") || (!re.MatchString(end) && end != "") {
// 	// 			if !re.MatchString(start) && start != "" {
// 	// 				return nil, nil, status.Errorf(codes.InvalidArgument, fmt.Sprintf("Invalid date: %s", start))
// 	// 			}
// 	// 			if !re.MatchString(end) && end != "" {
// 	// 				return nil, nil, status.Errorf(codes.InvalidArgument, fmt.Sprintf("Invalid date: %s", end))
// 	// 			}
// 	// 		} else if start != "" && re.MatchString(start) {
// 	// 			for i := 0; i < len(allLog); i++ {
// 	// 				if strings.Contains(FormatTime(allLog[i].Time), start) {
// 	// 					startIndex = i
// 	// 					break
// 	// 				}
// 	// 			}
// 	// 			if end != "" && re.MatchString(end) {
// 	// 				if !CheckValidTimeRange(start, end) {
// 	// 					return nil, nil, status.Errorf(codes.InvalidArgument, fmt.Sprintf("Invalid date: %s > %s", start, end))
// 	// 				}
// 	// 				for i := len(allLog) - 1; i >= 0; i-- {
// 	// 					if strings.Contains(FormatTime(allLog[i].Time), end) {
// 	// 						endIndex = i
// 	// 						break
// 	// 					}
// 	// 				}
// 	// 				for i := startIndex; i <= endIndex; i++ {
// 	// 					responseLog = append(responseLog, allLog[i])

// 	// 				}
// 	// 				return responseLog, changeLogs, nil
// 	// 			} else {
// 	// 				for i := startIndex; i < len(allLog); i++ {
// 	// 					responseLog = append(responseLog, allLog[i])
// 	// 				}
// 	// 				return responseLog, changeLogs, nil
// 	// 			}
// 	// 		} else if start == "" && end != "" && re.MatchString(end) {
// 	// 			startIndex = 0
// 	// 			for i := len(allLog) - 1; i >= 0; i-- {
// 	// 				if strings.Contains(FormatTime(allLog[i].Time), end) {
// 	// 					endIndex = i
// 	// 					break
// 	// 				} else {
// 	// 					continue
// 	// 				}
// 	// 			}
// 	// 			for i := 0; i <= endIndex; i++ {
// 	// 				responseLog = append(responseLog, allLog[i])

// 	// 			}
// 	// 			return responseLog, changeLogs, nil
// 	// 		}
// 	// 		return allLog, changeLogs, nil
// 	// 	}
// 	// }
// 	return []*LogItem{}, []*ChangeLogItem{}, nil
// }
