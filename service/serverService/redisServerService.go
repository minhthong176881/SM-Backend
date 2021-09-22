package serverService

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
	Data           string   `json:"data"`
	DependencyKeys []string `json:"dependencyKeys"`
}

func NewRedisServerService(baseService ServerService) *RedisServerService {
	redisClient := NewClient()
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

func NewClient() *redis.Client {
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

func (inst *RedisServerService) GetAll(query Query) ([]*Server, int64, error) {
	var data *GetAllResponse
	option := "skip=" + strconv.Itoa(int((query.PageIndex-1)*query.PageOffset)) + "&offset=" + strconv.Itoa(int(query.PageOffset))
	key := "query=" + query.Query + "&" + option
	err := Get(inst.redisClient, key, &data, []string{"dependency-servers"})
	if err != nil {
		return nil, 0, err
	}
	if data != nil {
		return data.Servers, data.Total, nil
	}
	servers, total, err := inst.baseService.GetAll(query)
	if err != nil {
		return nil, 0, err
	}
	if servers != nil {
		res := &GetAllResponse{
			Servers: servers,
			Total:   total,
		}
		err = Set(inst.redisClient, key, res, []string{"dependency-servers"})
		if err != nil {
			return nil, 0, err
		}
	}
	return servers, total, nil
}

func (inst *RedisServerService) GetById(id string) (*Server, error) {
	var res *Server
	dependencyKey := "dependency-server-" + id
	err := Get(inst.redisClient, id, &res, []string{dependencyKey})
	if err != nil {
		return nil, err
	}
	if res != nil {
		return res, nil
	}
	server, err := inst.baseService.GetById(id)
	if err != nil {
		return nil, err
	}
	if server != nil {
		err = Set(inst.redisClient, id, server, []string{dependencyKey})
		if err != nil {
			return nil, err
		}
	}
	return server, nil
}

func (inst *RedisServerService) Insert(server *Server) (*Server, error) {
	Update(inst.redisClient, "", []string{"dependency-servers"})
	return inst.baseService.Insert(server)
}

func (inst *RedisServerService) Update(id string, server *Server) (*Server, error) {
	current, err := inst.GetById(id)
	if err != nil {
		return nil, err
	}
	if !reflect.DeepEqual(current, server) {
		key := "dependency-server-" + server.ID.Hex()
		Update(inst.redisClient, key, []string{"dependency-servers"})
		server.UpdatedAt = strconv.FormatInt(time.Now().Unix(), 10)
	}
	return inst.baseService.Update(id, server)
}

func (inst *RedisServerService) Delete(id string) error {
	Delete(inst.redisClient, "dependency-server-"+id, []string{"dependency-servers"})
	return inst.baseService.Delete(id)
}

func Get(client *redis.Client, cacheKey string, result interface{}, dependencyKeys []string) error {
	var dependencyResult []string
	var redisCache *RedisCache
	cache, err := client.Get(client.Context(), cacheKey).Result()
	if err != nil && (err.Error() != string(redis.Nil)) {
		return err
	}
	if cache == "" {
		return nil
	}
	for i := 0; i < len(dependencyKeys); i++ {
		res, err := client.Get(client.Context(), dependencyKeys[i]).Result()
		if err != nil && (err.Error() != string(redis.Nil)) {
			return err
		}
		if res != "" {
			dependencyResult = append(dependencyResult, dependencyKeys[i]+":"+res)
		}
	}

	if len(dependencyResult) > 0 {
		err := json.Unmarshal([]byte(cache), &redisCache)
		if err != nil {
			client.Del(client.Context(), cacheKey)
			return err
		}
		if reflect.DeepEqual(redisCache.DependencyKeys, dependencyResult) {
			err = json.Unmarshal([]byte(redisCache.Data), &result)
			if err != nil {
				return err
			}
			return nil
		} else {
			client.Del(client.Context(), cacheKey)
			return nil
		}
	}
	return nil
}

func Set(client *redis.Client, cacheKey string, cacheVal interface{}, dependency []string) error {
	val, err := json.Marshal(cacheVal)
	if err != nil {
		return err
	}
	var dependencyKey []string

	currentTime := time.Now().Unix()
	timeStampString := strconv.FormatInt(currentTime, 10)
	for i := 0; i < len(dependency); i++ {
		dependencyVal, err := client.Get(client.Context(), dependency[i]).Result()
		if err != nil && (err.Error() != string(redis.Nil)) {
			return err
		}
		if dependencyVal == "" {
			client.Set(client.Context(), dependency[i], timeStampString, 0)
			dependencyKey = append(dependencyKey, dependency[i]+":"+timeStampString)
		} else {
			dependencyKey = append(dependencyKey, dependency[i]+":"+dependencyVal)
		}
	}

	redisCache := RedisCache{
		Data:           string(val),
		DependencyKeys: dependencyKey,
	}
	redisVal, err := json.Marshal(redisCache)
	if err != nil {
		return err
	}
	client.Set(client.Context(), cacheKey, redisVal, 0)
	return nil
}

func Update(client *redis.Client, cacheKey string, dependencyKey []string) {
	currentTime := time.Now().Unix()
	timeStampString := strconv.FormatInt(currentTime, 10)
	if cacheKey != "" {
		client.Set(client.Context(), cacheKey, timeStampString, 0)
	}
	if len(dependencyKey) > 0 {
		for i := 0; i < len(dependencyKey); i++ {
			client.Set(client.Context(), dependencyKey[i], timeStampString, 0)
		}
	}
}

func Delete(client *redis.Client, cacheKey string, dependencyKey []string) {
	Update(client, "", []string{"dependency-servers"})
	client.Del(client.Context(), cacheKey)
}
