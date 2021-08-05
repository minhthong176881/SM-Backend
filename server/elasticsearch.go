package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
	elastic "github.com/olivere/elastic/v7"
)

const (
	indexName = "servers"
	docType   = "log"
)

type ElasticsearchServer struct {
	Id       string `json:"id"`
	ServerId string `json:"serverId"`
	Log      string `json:"log"`
}

// Elastic search
func GetESClient() (*elastic.Client, error) {
	err := godotenv.Load(".env")
	if err != nil {
		log.Fatalf("Error loading .env file")
	}
	client, err := elastic.NewClient(elastic.SetURL(os.Getenv("ELASTICSEARCH_HOST")),
		elastic.SetSniff(false),
		elastic.SetHealthcheck(false))
	fmt.Println("ES initialized...")
	return client, err
}

func Insert(ctx context.Context, esClient *elastic.Client, server ElasticsearchServer) error {
	dataJSON, _ := json.Marshal(server)
	js := string(dataJSON)
	_, err := esClient.Index().Index(indexName).BodyJson(js).Do(ctx)

	if err != nil {
		return err
	}
	fmt.Println("[Elastic][InsertProduct]Insertion Successful")
	esClient.Flush().Index(indexName).Do(ctx)
	b := New()
	flushElasticsearch(b.redisClient)
	return nil
}

func Update(ctx context.Context, esClient *elastic.Client, id string, log string) error {
	check, err := Search(ctx, esClient, id)
	if err != nil {
		return err
	}
	if check.ServerId == "" {
		Insert(ctx, esClient, ElasticsearchServer{ServerId: id, Log: log})
		return nil
	} else {
		_, err := esClient.Update().Index(indexName).Id(check.Id).Doc(map[string]interface{}{"log": log}).Do(ctx)
		if err != nil {
			fmt.Println("[Elastic][UpdateProduct]Error: ", err)
			return err
		}
		b := New()
		flushElasticsearch(b.redisClient)
		return nil
	}
}

func Delete(ctx context.Context, esClient *elastic.Client, id string) error {
	check, err := Search(ctx, esClient, id)
	if err != nil {
		return err
	}
	_, err = esClient.Delete().Index(indexName).Id(check.Id).Do(ctx)
	if err != nil {
		fmt.Println("[Elastic][DeleteProduct]Error: ", err)
		return err
	}
	b := New()
	flushElasticsearch(b.redisClient)
	return nil
}

func Search(ctx context.Context, esClient *elastic.Client, id string) (ElasticsearchServer, error) {
	searchSource := elastic.NewSearchSource()
	searchSource.Query(elastic.NewMatchQuery("serverId", id))

	queryStr, err1 := searchSource.Source()
	_, err2 := json.Marshal(queryStr)

	if err1 != nil || err2 != nil {
		fmt.Println("[esClient][GetResponse]err during query marshal = ", err1, err2)
	}
	// fmt.Println("[esClient]Final ESQuery = \n", string(queryJs))
	exist, err := esClient.IndexExists(indexName).Do(ctx)
	if err != nil || !exist {
		fmt.Println("[esClient]Index not found = ", err)
		err = initIndex(ctx, esClient, indexName)
		if err != nil {
			fmt.Println("[esClient]Init index error = ", err)
			return ElasticsearchServer{}, err
		}
		fmt.Println("[esClient]Index initialized.")
		return ElasticsearchServer{}, nil
	} else {
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
