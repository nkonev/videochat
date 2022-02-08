package db

import (
	"context"
	"errors"
	"github.com/olivere/elastic/v7"
	"github.com/spf13/viper"
	. "nkonev.name/chat/logger"
	"nkonev.name/chat/utils"
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

func RunElasticMigrations(ops *ChatIndexOperations) error {
	ctx := context.Background()
	const mapping = `
{
	"settings":{
		"number_of_shards": 1,
		"number_of_replicas": 0
	},
	"mappings":{
		"properties":{
			"id":{
				"type":"long"
			},
			"name":{
				"type":"text"
			}
		}
	}
}`
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
