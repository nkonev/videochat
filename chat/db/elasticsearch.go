package db

import (
	"github.com/olivere/elastic/v7"
	"github.com/spf13/viper"
)

func NewElasticsearch() (*elastic.Client, error) {
	addrs := viper.GetStringSlice("elastic.addresses")
	return elastic.NewClient(
		elastic.SetURL(addrs[:]...))
}
