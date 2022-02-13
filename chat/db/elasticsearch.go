package db

import (
	"context"
	"errors"
	"github.com/olivere/elastic/v7"
	"github.com/spf13/viper"
	. "nkonev.name/chat/logger"
	"nkonev.name/chat/utils"
	"reflect"
)

type ChatIndexOperations struct {
	esClient *elastic.Client
}

func NewChatIndexOperations() (*ChatIndexOperations, error) {
	addrs := viper.GetStringSlice("elastic.addresses")
	client, err := elastic.NewClient(elastic.SetURL(addrs[:]...))
	if err != nil {
		return nil, err
	}
	return &ChatIndexOperations{esClient: client}, nil
}

const chatIndex = "chat"
const chatNameFiled = "name"

func RunElasticMigrations(ops *ChatIndexOperations) error {
	ctx := context.Background()
	const mapping = `
{
  "settings": {
    "number_of_shards": 1,
    "number_of_replicas": 0,
    "analysis": {
      "filter": {
        "ru_stop": {
          "type": "stop",
          "stopwords": "_russian_"
        },
        "ru_stemmer": {
          "type": "stemmer",
          "language": "russian"
        },
        "en_stop": {
          "type": "stop",
          "stopwords": "_english_"
        },
        "en_stemmer": {
          "type": "stemmer",
          "language": "english"
        }
      },
      "analyzer": {
        "my_analyzer": {
          "tokenizer": "standard",
          "filter": [
            "lowercase",
            "ru_stop",
            "ru_stemmer",
            "en_stop",
            "en_stemmer"
          ]
        },
        "rebuilt_standard": {
          "tokenizer": "standard",
          "filter": [
            "lowercase"
          ]
        }
      }
    }
  },
  "mappings": {
    "properties": {
      "id": {
        "type": "long"
      },
      "name": {
        "type": "text",
        "fields": {
          "std": {
            "type": "text",
            "analyzer": "rebuilt_standard",
            "term_vector": "with_positions_offsets_payloads"
          }
        },
		"fielddata": true,
        "index": true,
        "search_analyzer": "my_analyzer",
        "analyzer": "my_analyzer",
        "term_vector": "with_positions_offsets_payloads"
      }
    }
  }
}
`
	shouldReplace := viper.GetBool("elastic.replace")
	exists, err := ops.esClient.IndexExists(chatIndex).Do(ctx)
	if err != nil {
		Logger.Errorf("Error during checking index: %v", err)
		return err
	}
	if exists && shouldReplace {
		did, err := ops.esClient.DeleteIndex(chatIndex).Do(ctx)
		if err != nil {
			Logger.Errorf("Error during deleting index: %v", err)
			return err
		}
		if !did.Acknowledged {
			err := errors.New("not Acknowledged")
			Logger.Errorf("Error during deleting index: %v", err)
			return err
		}
		Logger.Infof("Index has been deleted")
	}
	if !exists || shouldReplace {
		createIndex, err := ops.esClient.CreateIndex(chatIndex).BodyString(mapping).Do(ctx)
		if err != nil {
			Logger.Errorf("Error during creating index: %v", err)
			return err
		}
		if !createIndex.Acknowledged {
			err := errors.New("not Acknowledged")
			Logger.Errorf("Error during creating index: %v", err)
			return err
		}
	}
	return nil
}

type ElasticChatDto struct {
	Id             int64   `json:"id"`
	ParticipantIds []int64 `json:"participantIds"`
	Name           string  `json:"name"`
}

func (ops *ChatIndexOperations) SaveChat(dto *ElasticChatDto) error {
	if dto == nil {
		return errors.New("Saving dto cannot be null")
	}
	ctx := context.Background()
	_, err := ops.esClient.Index().
		Index(chatIndex).
		Id(utils.Int64ToString(dto.Id)).
		BodyJson(dto).
		Do(ctx)
	return err
}

func (ops *ChatIndexOperations) UpdateChat(dto *ElasticChatDto) error {
	ctx := context.Background()
	_, err := ops.esClient.Update().Index(chatIndex).Id(utils.Int64ToString(dto.Id)).
		Doc(dto).Do(ctx)
	return err
}

func (ops *ChatIndexOperations) DeleteChat(id int64) error {
	ctx := context.Background()
	_, err := ops.esClient.Delete().Index(chatIndex).Id(utils.Int64ToString(id)).Do(ctx)
	return err
}

func (ops *ChatIndexOperations) SearchChat(searchString string, size, offset int) ([]*ElasticChatDto, error) {
	ctx := context.Background()

	aQuery := elastic.NewWildcardQuery(chatNameFiled, "*"+searchString+"*")
	searchResult, err := ops.esClient.Search().
		Index(chatIndex).          // search in index "twitter"
		Query(aQuery).             // specify the query
		Sort(chatNameFiled, true). // sort by "user" field, ascending
		From(offset).Size(size).   // take documents 0-9
		Pretty(true).              // pretty print request and response JSON
		Do(ctx)                    // execute
	if err != nil {
		// Handle error
		return nil, err
	}

	// searchResult is of type SearchResult and returns hits, suggestions,
	// and all kinds of other information from Elasticsearch.
	Logger.Debugf("Query took %d milliseconds\n", searchResult.TookInMillis)

	var result = make([]*ElasticChatDto, 0)

	// Each is a convenience function that iterates over hits in a search result.
	// It makes sure you don't need to check for nil values in the response.
	var ttyp *ElasticChatDto
	for _, item := range searchResult.Each(reflect.TypeOf(ttyp)) {
		if elasticChatDto, ok := item.(*ElasticChatDto); ok {
			result = append(result, elasticChatDto)
		} else {
			Logger.Errorf("Not able to deserialize ElasticChatDto")
		}
	}
	// TotalHits is another convenience function that works even when something goes wrong.
	Logger.Debugf("Found a total of %d chats\n", searchResult.TotalHits())

	return result, nil
}
