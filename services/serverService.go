package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"reflect"
	"regexp"
	"strconv"
	"strings"

	"github.com/360EntSecGroup-Skylar/excelize/v2"
	"github.com/go-redis/redis"
	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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

type ServerServiceInterface interface {
	Register(user User) (string, error)
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

type ServerService struct {
	MongoService         *MongoServerService
	RedisService         *RedisServerService
	ElasticsearchService *ElasticsearchServerService
}

func NewServerService(ElasticsearchService *ElasticsearchServerService) ServerService {
	// MongoService := NewMongoServerService()
	// RedisService := NewRedisServerService(*MongoService)
	// ElasticsearchService := NewElasticsearchServerService(*RedisService)
	return ServerService{
		MongoService:         ElasticsearchService.redisService.mongoService,
		RedisService:         ElasticsearchService.redisService,
		ElasticsearchService: ElasticsearchService,
	}
}

func (ss *ServerService) Register(user User) (string, error) {
	return ss.MongoService.Register(&user)
}

func (ss *ServerService) Login(username string, password string) (bool, error) {
	return ss.MongoService.Login(username, password)
}

func (ss *ServerService) GetAll(query Query) ([]*Server, int64, error) {
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
	queryJson, _ := bson.Marshal(query)
	option := "skip=" + strconv.Itoa(int((query.PageIndex-1)*query.PageOffset)) + "&offset=" + strconv.Itoa(int(query.PageOffset))
	key := string(queryJson) + option

	total, err := ss.MongoService.serverCollection.CountDocuments(context.Background(), query)
	if err != nil {
		return nil, 0, status.Errorf(codes.Internal, fmt.Sprintf("Unknown internal error: %v", err))
	}

	serverRedis, err := ss.RedisService.redisClient.Get(ss.RedisService.redisClient.Context(), key).Result()
	if (err != nil && (err.Error() == string(redis.Nil))) || serverRedis == "" {
		log.Printf("Key %s does not exist.", key)
		servers, total, err := ss.MongoService.GetAll(queryDB, (query.PageIndex-1)*query.PageOffset, query.PageOffset)
		if err != nil {
			fmt.Println(err)
			return nil, 0, err
		}
		redisVal, err := json.Marshal(servers)
		if err == nil {
			ss.RedisService.redisClient.Set(ss.RedisService.redisClient.Context(), key, redisVal, 0)
			log.Println("Set key to redis successfully")
		}
		return servers, total, err
	} else if serverRedis != string(redis.Nil) {
		var redisRes []*Server
		err = json.Unmarshal([]byte(serverRedis), &redisRes)
		if err != nil {
			fmt.Println(err)
			return nil, 0, err
		}
		return redisRes, total, nil
	}
	return nil, 0, err
}

func (ss *ServerService) GetById(id string) (*Server, error) {
	serverRedis, err := ss.RedisService.redisClient.Get(ss.RedisService.redisClient.Context(), id).Result()
	var data *Server
	if (err != nil && (err.Error() == string(redis.Nil))) || serverRedis == "" {
		log.Printf("Key %s does not exist", id)
		result, err := ss.MongoService.GetById(id)
		if err != nil {
			return nil, err
		}
		redisVal, err := json.Marshal(result)
		if err == nil {
			ss.RedisService.redisClient.Set(ss.RedisService.redisClient.Context(), id, redisVal, 0)
			log.Println("Set key to redis successfully")
		}
		return result, nil
	} else if serverRedis != string(redis.Nil) {
		err = json.Unmarshal([]byte(serverRedis), &data)
		if err != nil {
			return nil, err
		}
		return data, nil
	} else {
		return nil, err
	}
}

func (ss *ServerService) Insert(server *Server) (*Server, error) {
	id, err := ss.MongoService.Insert(server)
	if err != nil {
		return nil, err
	}
	serverId, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, status.Errorf(
			codes.InvalidArgument,
			fmt.Sprintf("Could not convert the supplied server Id to a MongoDB ObjectId: %v", err),
		)
	}
	server.ID = serverId
	err = ss.ElasticsearchService.Insert(context.Background(), ss.ElasticsearchService.elasticClient, ElasticsearchServer{ServerId: server.ID.Hex(), Log: ""})
	if err != nil {
		log.Println("Cannot insert server to elasticsearch")
	}
	ss.RedisService.redisClient.FlushAll(context.Background())
	return server, nil
}

func (ss *ServerService) Update(id string, server *Server) (*Server, error) {
	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, status.Errorf(
			codes.InvalidArgument,
			fmt.Sprintf("Could not convert the supplied server Id to a MongoDB ObjectId: %v", err),
		)
	}

	current, _ := ss.GetById(id)
	if !reflect.DeepEqual(current, server) {
		ss.RedisService.flushServer(ss.RedisService.redisClient, id)
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
	return ss.MongoService.Update(update, filter)
}

func (ss *ServerService) Delete(id string) error {
	err := ss.ElasticsearchService.Delete(context.Background(), ss.ElasticsearchService.elasticClient, id)
	if err != nil {
		return err
	}
	ss.RedisService.flushServer(ss.RedisService.redisClient, id)
	return ss.MongoService.Delete(id)
}

func (ss *ServerService) Export() (string, error) {
	var myTableName = "Server list"
	f := excelize.NewFile()
	f.DeleteSheet("Sheet1")
	index := f.NewSheet(myTableName)
	_ = f.SetCellValue(myTableName, "A2", "Server")
	_ = f.SetCellValue(myTableName, "B2", "IP")
	_ = f.SetCellValue(myTableName, "C2", "Username")
	_ = f.SetCellValue(myTableName, "D2", "Password")
	_ = f.SetCellValue(myTableName, "E2", "Status")
	_ = f.SetCellValue(myTableName, "F2", "Password validate")
	_ = f.SetCellValue(myTableName, "G2", "Description")

	servers, _, err := ss.GetAll(Query{})
	if err != nil {
		return "", err
	}
	for i := 3; i < len(servers)+3; i++ {
		num := strconv.FormatInt(int64(i), 10)
		var status string
		if servers[i-3].Status {
			status = "On"
		} else {
			status = "Off"
		}
		var validate string
		if servers[i-3].Validate {
			validate = "Valid"
		} else {
			validate = "Invalid"
		}
		_ = f.SetCellValue(myTableName, "A"+num, i-2)
		_ = f.SetCellValue(myTableName, "B"+num, servers[i-3].Ip)
		_ = f.SetCellValue(myTableName, "C"+num, servers[i-3].Username)
		_ = f.SetCellValue(myTableName, "D"+num, servers[i-3].Password)
		_ = f.SetCellValue(myTableName, "E"+num, status)
		_ = f.SetCellValue(myTableName, "F"+num, validate)
		_ = f.SetCellValue(myTableName, "G"+num, servers[i-3].Description)
	}
	f.SetActiveSheet(index)
	f.Path = "public/OpenAPI/exports/Server_list.xlsx"
	_ = f.Save()

	err = godotenv.Load(".env")
	if err != nil {
		log.Fatalf("Error loading .env file")
	}
	host := os.Getenv("HOST")
	return host + "/exports/Server_list.xlsx", nil
}

func (ss *ServerService) Check(id string) (bool, error) {
	server, err := ss.GetById(id)
	if err != nil {
		return false, status.Errorf(codes.NotFound, fmt.Sprintf("Could not find server with Id %s: %v", id, err))
	}

	conn, err := Connect(server.Ip+":"+strconv.FormatInt(server.Port, 10), server.Username, server.Password)
	if err != nil {
		if strings.Contains(err.Error(), "ssh") && strings.Contains(err.Error(), "handshake") {
			server.Status = true
			_, _ = ss.Update(id, server)
			return true, nil
		}
		server.Status = false
		_, _ = ss.Update(id, server)
		return false, err
	}
	if conn != nil {
		server.Status = true
		_, _ = ss.Update(id, server)
		return true, nil
	} else {
		server.Status = false
		_, _ = ss.Update(id, server)
		return false, nil
	}
}

func (ss *ServerService) Validate(id string) (bool, error) {
	server, err := ss.GetById(id)
	if err != nil {
		server.Validate = false
		_, _ = ss.Update(id, server)
		return false, status.Errorf(codes.NotFound, fmt.Sprintf("Could not find server with Id %s: %v", id, err))
	}

	conn, err := Connect(server.Ip+":"+strconv.FormatInt(server.Port, 10), server.Username, server.Password)
	if err != nil {
		server.Validate = false
		_, _ = ss.Update(id, server)
		return false, err
	}
	if conn != nil {
		server.Validate = true
		_, _ = ss.Update(id, server)
		return true, nil
	} else {
		server.Validate = false
		_, _ = ss.Update(id, server)
		return false, nil
	}
}

func (ss *ServerService) GetLog(id string, start string, end string, date string, month string) ([]*LogItem, []*ChangeLogItem, error) {
	serverRedis, err := ss.RedisService.redisClient.Get(ss.RedisService.redisClient.Context(), id+"_log").Result()
	var elasticServer ElasticsearchServer
	if (err != nil && (err.Error() == string(redis.Nil))) || serverRedis == "" {
		log.Printf("Key %s does not exist", id+"_log")
		elastic, err := ss.ElasticsearchService.Search(context.Background(), ss.ElasticsearchService.elasticClient, id)
		if err != nil {
			return nil, nil, err
		}
		redisVal, err := json.Marshal(elastic)
		if err == nil {
			ss.RedisService.redisClient.Set(ss.RedisService.redisClient.Context(), id+"_log", redisVal, 0)
			log.Println("Set key to redis successfully")
		}
		elasticServer = elastic
	} else if serverRedis != string(redis.Nil) {
		err = json.Unmarshal([]byte(serverRedis), &elasticServer)
		if err != nil {
			return nil, nil, err
		}
	} else {
		return nil, nil, err
	}
	// elasticServer, err := Search(ctx, b.esClient, id)
	// if err != nil {
	// 	return nil, err
	// }
	logs := strings.Split(elasticServer.Log, "\n")
	var startIndex int
	var endIndex int
	var allLog []*LogItem
	var responseLog []*LogItem
	var changeLogs []*ChangeLogItem
	for i := 0; i < len(logs)-1; i++ {
		var log LogItem
		log.Time = strings.Split(logs[i], " ")[0]
		log.Status = strings.Split(logs[i], " ")[1]
		allLog = append(allLog, &log)
	}

	re := regexp.MustCompile(`^\d{4}\-(0[1-9]|1[012])\-(0[1-9]|[12][0-9]|3[01])$`)
	reMonth := regexp.MustCompile(`^\d{4}\-(0[1-9]|1[012])$`)

	if month != "" && reMonth.MatchString(month) {
		for i := 0; i < len(allLog); i++ {
			if strings.Contains(FormatTime(allLog[i].Time), month) {
				responseLog = append(responseLog, allLog[i])
			}
		}
		return responseLog, changeLogs, nil
	} else if month != "" && !reMonth.MatchString(month) {
		return nil, nil, status.Errorf(codes.InvalidArgument, fmt.Sprintf("Invalid date: %s", month))
	} else {
		if date != "" && re.MatchString(date) {
			for i := 0; i < len(allLog); i++ {
				if strings.Contains(FormatTime(allLog[i].Time), date) {
					responseLog = append(responseLog, allLog[i])
				}
			}
			changeLogs := GetChangeLog(responseLog, changeLogs)
			return responseLog, changeLogs, nil
		} else if date != "" && !re.MatchString(date) {
			return nil, nil, status.Errorf(codes.InvalidArgument, fmt.Sprintf("Invalid date: %s", date))
		} else {
			if (!re.MatchString(start) && start != "") || (!re.MatchString(end) && end != "") {
				if !re.MatchString(start) && start != "" {
					return nil, nil, status.Errorf(codes.InvalidArgument, fmt.Sprintf("Invalid date: %s", start))
				}
				if !re.MatchString(end) && end != "" {
					return nil, nil, status.Errorf(codes.InvalidArgument, fmt.Sprintf("Invalid date: %s", end))
				}
			} else if start != "" && re.MatchString(start) {
				for i := 0; i < len(allLog); i++ {
					if strings.Contains(FormatTime(allLog[i].Time), start) {
						startIndex = i
						break
					}
				}
				if end != "" && re.MatchString(end) {
					if !CheckValidTimeRange(start, end) {
						return nil, nil, status.Errorf(codes.InvalidArgument, fmt.Sprintf("Invalid date: %s > %s", start, end))
					}
					for i := len(allLog) - 1; i >= 0; i-- {
						if strings.Contains(FormatTime(allLog[i].Time), end) {
							endIndex = i
							break
						}
					}
					for i := startIndex; i <= endIndex; i++ {
						responseLog = append(responseLog, allLog[i])

					}
					return responseLog, changeLogs, nil
				} else {
					for i := startIndex; i < len(allLog); i++ {
						responseLog = append(responseLog, allLog[i])
					}
					return responseLog, changeLogs, nil
				}
			} else if start == "" && end != "" && re.MatchString(end) {
				startIndex = 0
				for i := len(allLog) - 1; i >= 0; i-- {
					if strings.Contains(FormatTime(allLog[i].Time), end) {
						endIndex = i
						break
					} else {
						continue
					}
				}
				for i := 0; i <= endIndex; i++ {
					responseLog = append(responseLog, allLog[i])

				}
				return responseLog, changeLogs, nil
			}
			return allLog, changeLogs, nil
		}
	}
}
