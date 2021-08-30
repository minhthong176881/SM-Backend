package serverLogService

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"regexp"
	"strings"

	"github.com/go-redis/redis/v8"
	"github.com/joho/godotenv"
	"github.com/minhthong176881/Server_Management/utils"
	serverService "github.com/minhthong176881/Server_Management/service/serverService"
	elastic "github.com/olivere/elastic/v7"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	indexName = "servers"
	docType   = "log"
)

type ElasticsearchServerService struct {
	ElasticClient *elastic.Client
}

type ElasticsearchServer struct {
	Id       string `json:"id"`
	ServerId string `json:"serverId"`
	Log      string `json:"log"`
}

// Elastic search
func NewElasticsearchServerService() *ElasticsearchServerService {
	err := godotenv.Load(".env")
	if err != nil {
		log.Fatalf("Error loading .env file")
	}
	client, err := elastic.NewClient(elastic.SetURL(os.Getenv("ELASTICSEARCH_HOST")),
		elastic.SetSniff(false),
		elastic.SetHealthcheck(false))
	if err != nil {
		log.Fatal("Error initialize Elasticsearch: ", err)
	}
	fmt.Println("ES initialized...")
	exist, err := client.IndexExists(indexName).Do(context.Background())
	if err != nil || !exist {
		fmt.Println("[esClient]Index not found = ", err)
		err = initIndex(context.Background(), client, indexName)
		if err != nil {
			fmt.Println("[esClient]Init index error = ", err)
			log.Fatal(err)
		}
		fmt.Println("[esClient]Index initialized.")
	}
	return &ElasticsearchServerService{ElasticClient: client}
}

func (esServer *ElasticsearchServerService) Insert(ctx context.Context, esClient *elastic.Client, server ElasticsearchServer) error {
	dataJSON, _ := json.Marshal(server)
	js := string(dataJSON)
	_, err := esClient.Index().Index(indexName).BodyJson(js).Do(ctx)

	if err != nil {
		return err
	}
	fmt.Println("[Elastic][InsertProduct]Insertion Successful")
	esClient.Flush().Index(indexName).Do(ctx)
	return nil
}

func (esServer *ElasticsearchServerService) Update(ctx context.Context, esClient *elastic.Client, id string, log string) error {
	check, err := esServer.Search(ctx, esClient, id)
	if err != nil {
		return err
	}
	if check.ServerId == "" {
		esServer.Insert(ctx, esClient, ElasticsearchServer{ServerId: id, Log: log})
		return nil
	} else {
		_, err := esClient.Update().Index(indexName).Id(check.Id).Doc(map[string]interface{}{"log": log}).Do(ctx)
		if err != nil {
			fmt.Println("[Elastic][UpdateProduct]Error: ", err)
			return err
		}
		esClient.Flush().Index(indexName).Do(ctx)
		return nil
	}
}

func (esServer *ElasticsearchServerService) Delete(ctx context.Context, esClient *elastic.Client, id string) error {
	check, err := esServer.Search(ctx, esClient, id)
	if err != nil {
		return err
	}
	_, err = esClient.Delete().Index(indexName).Id(check.Id).Do(ctx)
	if err != nil {
		fmt.Println("[Elastic][DeleteProduct]Error: ", err)
		return err
	}
	esClient.Flush().Index(indexName).Do(ctx)
	return nil
}

func (esServer *ElasticsearchServerService) Search(ctx context.Context, esClient *elastic.Client, id string) (ElasticsearchServer, error) {
	searchSource := elastic.NewSearchSource()
	searchSource.Query(elastic.NewMatchQuery("serverId", id))

	queryStr, err1 := searchSource.Source()
	_, err2 := json.Marshal(queryStr)

	if err1 != nil || err2 != nil {
		fmt.Println("[esClient][GetResponse]err during query marshal = ", err1, err2)
	}
	// fmt.Println("[esClient]Final ESQuery = \n", string(queryJs))
	searchService := esClient.Search().Index(indexName).SearchSource(searchSource)
	searchResult, err := searchService.Do(ctx)
	if err != nil {
		fmt.Println("[ProductsES][GetPIDs]Error = ", err)
		return ElasticsearchServer{}, err
	}

	var result = convertSearchResultToServers(searchResult)
	if len(result) > 0 {
		return result[0], nil
	}
	return ElasticsearchServer{}, nil

}

func (esServer *ElasticsearchServerService) GetLog(id string, start string, end string, date string, month string) ([]*LogItem, []*ChangeLogItem, error) {
	redisClient := serverService.NewClient()
	var elasticServer ElasticsearchServer
	serverRedis, err := redisClient.Get(redisClient.Context(), id+"_log").Result()
	if err != nil && err.Error() != string(redis.Nil) {
		return nil, nil, err
	}
	if err == redis.Nil {
		elastic, err := esServer.Search(context.Background(), esServer.ElasticClient, id)
		if err != nil {
			return nil, nil, err
		}
		redisVal, err := json.Marshal(elastic)
		if err == nil {
			redisClient.Set(redisClient.Context(), id+"_log", redisVal, 0)
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
			if strings.Contains(utils.FormatTime(allLog[i].Time), month) {
				responseLog = append(responseLog, allLog[i])
			}
		}
		return responseLog, changeLogs, nil
	} else if month != "" && !reMonth.MatchString(month) {
		return nil, nil, status.Errorf(codes.InvalidArgument, fmt.Sprintf("Invalid date: %s", month))
	} else {
		if date != "" && re.MatchString(date) {
			for i := 0; i < len(allLog); i++ {
				if strings.Contains(utils.FormatTime(allLog[i].Time), date) {
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
					if strings.Contains(utils.FormatTime(allLog[i].Time), start) {
						startIndex = i
						break
					}
				}
				if end != "" && re.MatchString(end) {
					if !utils.CheckValidTimeRange(start, end) {
						return nil, nil, status.Errorf(codes.InvalidArgument, fmt.Sprintf("Invalid date: %s > %s", start, end))
					}
					for i := len(allLog) - 1; i >= 0; i-- {
						if strings.Contains(utils.FormatTime(allLog[i].Time), end) {
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
					if strings.Contains(utils.FormatTime(allLog[i].Time), end) {
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

func convertSearchResultToServers(searchResult *elastic.SearchResult) []ElasticsearchServer {
	var result []ElasticsearchServer
	for _, hit := range searchResult.Hits.Hits {
		var serverObj ElasticsearchServer
		err := json.Unmarshal(hit.Source, &serverObj)
		serverObj.Id = hit.Id
		if err != nil {
			fmt.Println("[Getting Servers][Unmarshal] Err = ", err)
			continue
		}
		result = append(result, serverObj)
	}
	return result
}

func initIndex(ctx context.Context, esClient *elastic.Client, index string) error {
	mappings := `
	{
		"settings":{
			"number_of_shards":2,
			"number_of_replicas":1
		},
		"mappings":{
			"properties":{
				"field serverId":{
					"type":"text"
				},
				"field log":{
					"type":"text"
				}
			}
		}
	}`
	_, err := esClient.CreateIndex(index).Body(mappings).Do(ctx)
	if err != nil {
		return err
	} else {
		return nil
	}
}

