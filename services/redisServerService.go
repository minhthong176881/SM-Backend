package services

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"reflect"
	"strconv"
	"strings"

	"github.com/go-redis/redis/v8"
	"github.com/joho/godotenv"
)

type RedisServerService struct {
	redisClient *redis.Client
	baseService ServerService
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
	var data *GetAllResponse
	queryJson, _ := json.Marshal(query)
	option := "skip=" + strconv.Itoa(int((query.PageIndex-1)*query.PageOffset)) + "&offset=" + strconv.Itoa(int(query.PageOffset))
	key := string(queryJson) + option
	cachedData, err := inst.redisClient.Get(inst.redisClient.Context(), key).Result()
	if err != nil && (err.Error() != string(redis.Nil)) {
		return nil, 0, err
	}
	if cachedData != "" {
		err := json.Unmarshal([]byte(cachedData), &data)
		if err != nil {
			return nil, 0, err
		}
		return data.Servers, data.Total, nil
	}
	servers, total, err := inst.baseService.GetAll(query)
	if err != nil {
		return nil, 0, err
	}
	if servers != nil {
		res := &GetAllResponse {
			Servers: servers,
			Total: total,
		}
		redisVal, err := json.Marshal(res)
		if err != nil {
			return nil, 0, err
		}
		inst.redisClient.Set(inst.redisClient.Context(), key, redisVal, 0)
	}
	return servers, total, nil
}

func (inst *RedisServerService) GetById(id string) (*Server, error) {
	var data *Server
	cacheData, err := inst.redisClient.Get(inst.redisClient.Context(), id).Result()
	if err != nil && (err.Error() != string(redis.Nil)) {
		return nil, err
	}
	if cacheData != "" {
		err = json.Unmarshal([]byte(cacheData), &data)
		if err != nil {
			return nil, err
		}
		return data, nil
	}

	server, err := inst.baseService.GetById(id)
	if err != nil {
		return nil, err
	}
	if server != nil {
		redisVal, err := json.Marshal(server)
		if err != nil {
			return nil, err
		}
		inst.redisClient.Set(inst.redisClient.Context(), id, redisVal, 0)
	}
	return server, nil
}

func (inst *RedisServerService) Insert(server *Server) (*Server, error) {
	inst.redisClient.FlushAll(inst.redisClient.Context())
	return inst.baseService.Insert(server)
}

func (inst *RedisServerService) Update(id string, server *Server) (*Server, error) {
	current, err := inst.GetById(id)
	if err != nil {
		return nil, err
	}
	if !reflect.DeepEqual(current, server) {
		inst.flushServer(inst.redisClient, server.ID.Hex())
	}
	return inst.baseService.Update(id, server)
}

func (inst *RedisServerService) Delete(id string) (error) {
	inst.redisClient.FlushAll(inst.redisClient.Context())
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

func (inst *RedisServerService) flushServer(client *redis.Client, serverId string) {
	iter := client.Scan(client.Context(), 0, "*", 0).Iterator()
	for iter.Next(client.Context()) {
		response, _ := client.Get(client.Context(), iter.Val()).Result()
		if strings.Contains(response, serverId) {
			client.Del(client.Context(), iter.Val())
		}
	}
	if err := iter.Err(); err != nil {
		fmt.Println(err)
	}
}
