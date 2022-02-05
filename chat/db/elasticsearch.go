package db

import (
	"context"
	"errors"
	"github.com/olivere/elastic/v7"
	"github.com/spf13/viper"
	. "nkonev.name/chat/logger"
)

func NewElasticsearch() (*elastic.Client, error) {
	addrs := viper.GetStringSlice("elastic.addresses")
	return elastic.NewClient(
		elastic.SetURL(addrs[:]...))
}

func RunElasticMigrations(client *elastic.Client) error {
	ctx := context.Background()
	const chatIndex = "chat"
	const mapping = `
{
	"settings":{
		"number_of_shards": 1,
		"number_of_replicas": 0
	},
	"mappings":{
		"properties":{
			"ownerId":{
				"type":"long"
			},
			"title":{
				"type":"text"
			}
		}
	}
}`
	exists, err := client.IndexExists(chatIndex).Do(ctx)
	if err != nil {
		Logger.Errorf("Error during checking index: %v", err)
		return err
	}
	if !exists {
		createIndex, err := client.CreateIndex(chatIndex).BodyString(mapping).Do(ctx)
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
