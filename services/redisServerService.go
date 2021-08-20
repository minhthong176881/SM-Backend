package services

import (
	"encoding/json"
	"log"
	"os"
	"reflect"
	"strconv"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/joho/godotenv"
)

type RedisServerService struct {
	redisClient *redis.Client
	baseService ServerService
}

type RedisCache struct {
	Data           string `json:"data"`
	DependencyKeys string `json:"dependencyKeys"`
}

func NewRedisServerService(baseService ServerService) *RedisServerService {
	redisClient := newClient()
	result, err := ping(redisClient)
	if err != nil {
		log.Fatal(err)
	} else {
		log.Println(result)
	}
	return &RedisServerService{
		redisClient: redisClient,
		baseService: baseService,
	}
}

func newClient() *redis.Client {
	err := godotenv.Load(".env")
	if err != nil {
		log.Fatalf("Error loading .env file")
	}
	db, _ := strconv.Atoi(os.Getenv("REDIS_DATABASE"))
	redisClient := redis.NewClient(&redis.Options{
		Addr:     os.Getenv("REDIS_HOST"),
		Password: os.Getenv("REDIS_PASSWORD"),
		DB:       db,
	})
	return redisClient
}

func ping(inst *redis.Client) (string, error) {
	result, err := inst.Ping(inst.Context()).Result()

	if err != nil {
		return "", err
	} else {
		return result, err
	}
}

func (inst *RedisServerService) Register(user *User) (string, error) {
	return inst.baseService.Register(user)
}

func (inst *RedisServerService) Login(username string, password string) (bool, error) {
	return inst.baseService.Login(username, password)
}

func (inst *RedisServerService) GetAll(query Query) ([]*Server, int64, error) {
	var redisCache *RedisCache
	var data *GetAllResponse
	var dependencyExist bool
	option := "skip=" + strconv.Itoa(int((query.PageIndex-1)*query.PageOffset)) + "&offset=" + strconv.Itoa(int(query.PageOffset))
	key := "query=" + query.Query + option
	cachedData, err := inst.redisClient.Get(inst.redisClient.Context(), key).Result()
	if err != nil && (err.Error() != string(redis.Nil)) {
		return nil, 0, err
	}
	dependency, err := inst.redisClient.Get(inst.redisClient.Context(), "dependency-servers").Result()
	if err != nil && (err.Error() != string(redis.Nil)) {
		return nil, 0, err
	} 
	if cachedData != "" && dependency != "" {
		dependencyExist = true
		err := json.Unmarshal([]byte(cachedData), &redisCache)
		if err != nil {
			return nil, 0, err
		}
		if redisCache.DependencyKeys == "dependency-servers:" + dependency {
			err := json.Unmarshal([]byte(redisCache.Data), &data)
			if err != nil {
				return nil, 0, err
			}
			return data.Servers, data.Total, nil
		} else {
			inst.redisClient.Del(inst.redisClient.Context(), key)
		}
	}
	servers, total, err := inst.baseService.GetAll(query)
	if err != nil {
		return nil, 0, err
	}
	if servers != nil {
		var dependencyKey string
		if !dependencyExist {
			currentTime := time.Now().Unix()
			timeStampString := strconv.FormatInt(currentTime, 10)
			dependencyKey = "dependency-servers:" + timeStampString
			inst.redisClient.Set(inst.redisClient.Context(), "dependency-servers", timeStampString, 0)
		} else {
			dependencyKey = "dependency-servers:" + dependency
		}
		res := &GetAllResponse{
			Servers: servers,
			Total:   total,
		}
		val, err := json.Marshal(res)
		if err != nil {
			return nil, 0, err
		}
		redisCache := RedisCache {
			Data: string(val),
			DependencyKeys: dependencyKey,
		} 
		redisVal, err := json.Marshal(redisCache)
		if err != nil {
			return nil, 0, err
		}
		inst.redisClient.Set(inst.redisClient.Context(), key, redisVal, 0)
	}
	return servers, total, nil
}

func (inst *RedisServerService) GetById(id string) (*Server, error) {
	var data *RedisCache
	var res *Server
	key := "dependency-server-" + id
	var dependencyExist bool
	cacheData, err := inst.redisClient.Get(inst.redisClient.Context(), id).Result()
	if err != nil && (err.Error() != string(redis.Nil)) {
		return nil, err
	}
	dependency, err := inst.redisClient.Get(inst.redisClient.Context(), key).Result()
	if err != nil && (err.Error() != string(redis.Nil)) {
		return nil, err
	}
	if cacheData != "" && dependency != "" {
		dependencyExist = true
		err := json.Unmarshal([]byte(cacheData), &data)
		if err != nil {
			return nil, err
		}
		if data.DependencyKeys == (key + ":" + dependency) {
			err = json.Unmarshal([]byte(data.Data), &res)
			if err != nil {
				return nil, err
			}
			return res, nil
		} else {
			inst.redisClient.Del(inst.redisClient.Context(), id)
		}
	}
	server, err := inst.baseService.GetById(id)
	if err != nil {
		return nil, err
	}
	if server != nil {
		val, err := json.Marshal(server)
		if err != nil {
			return nil, err
		}
		var dependencyKey string
		if !dependencyExist {
			currentTime := time.Now().Unix()
			timeStampString := strconv.FormatInt(currentTime, 10)
			dependencyKey = key + ":" + timeStampString
			inst.redisClient.Set(inst.redisClient.Context(), key, timeStampString, 0)
		} else {
			dependencyKey = key + ":" + dependency
		}
		redisCache := RedisCache{
			Data:           string(val),
			DependencyKeys: dependencyKey,
		}
		redisVal, err := json.Marshal(redisCache)
		if err != nil {
			return nil, err
		}
		inst.redisClient.Set(inst.redisClient.Context(), id, redisVal, 0)
	}
	return server, nil
}

func (inst *RedisServerService) Insert(server *Server) (*Server, error) {
	// inst.redisClient.FlushAll(inst.redisClient.Context())
	currentTime := time.Now().Unix()
	timeStampString := strconv.FormatInt(currentTime, 10)
	inst.redisClient.Set(inst.redisClient.Context(), "dependency-servers", timeStampString, 0)
	return inst.baseService.Insert(server)
}

func (inst *RedisServerService) Update(id string, server *Server) (*Server, error) {
	current, err := inst.GetById(id)
	if err != nil {
		return nil, err
	}
	currentTime := time.Now().Unix()
	timeStampString := strconv.FormatInt(currentTime, 10)
	if !reflect.DeepEqual(current, server) {
		// inst.flushServer(inst.redisClient, server.ID.Hex())
		key := "dependency-server-" + server.ID.Hex()
		inst.redisClient.Set(inst.redisClient.Context(), key, timeStampString, 0)
		inst.redisClient.Set(inst.redisClient.Context(), "dependency-servers", timeStampString, 0)
	}
	return inst.baseService.Update(id, server)
}

func (inst *RedisServerService) Delete(id string) error {
	// inst.redisClient.FlushAll(inst.redisClient.Context())
	currentTime := time.Now().Unix()
	timeStampString := strconv.FormatInt(currentTime, 10)
	inst.redisClient.Set(inst.redisClient.Context(), "dependency-keys", timeStampString, 0)
	key := "dependency-server-" + id
	inst.redisClient.Del(inst.redisClient.Context(), key)
	return inst.baseService.Delete(id)
}

func (inst *RedisServerService) Export() (string, error) {
	return inst.baseService.Export()
}

func (inst *RedisServerService) Check(id string) (bool, error) {
	return inst.baseService.Check(id)
}

func (inst *RedisServerService) Validate(id string) (bool, error) {
	return inst.baseService.Validate(id)
}

func (inst *RedisServerService) GetLog(id string, start string, end string, date string, month string) ([]*LogItem, []*ChangeLogItem, error) {
	return inst.baseService.GetLog(id, start, end, date, month)
}

func (inst *RedisServerService) flushElasticsearch(client *redis.Client) {
	iter := client.Scan(client.Context(), 0, "_log", 0).Iterator()
	for iter.Next(client.Context()) {
		client.Del(client.Context(), iter.Val())
	}
}

// func (inst *RedisServerService) flushServer(client *redis.Client, serverId string) {
// 	iter := client.Scan(client.Context(), 0, "*", 0).Iterator()
// 	for iter.Next(client.Context()) {
// 		response, _ := client.Get(client.Context(), iter.Val()).Result()
// 		if strings.Contains(response, serverId) {
// 			client.Del(client.Context(), iter.Val())
// 		}
// 	}
// 	if err := iter.Err(); err != nil {
// 		fmt.Println(err)
// 	}
// }

