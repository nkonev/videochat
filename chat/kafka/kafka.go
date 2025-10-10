package kafka

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"nkonev.name/chat/app"
	"nkonev.name/chat/config"
	"nkonev.name/chat/logger"
	"nkonev.name/chat/utils"
	"os"
	"time"

	"github.com/Jeffail/gabs/v2"
	"github.com/twmb/franz-go/pkg/kadm"
	"github.com/twmb/franz-go/pkg/kerr"
	"github.com/twmb/franz-go/pkg/kgo"
	"github.com/twmb/franz-go/pkg/kversion"
	"go.uber.org/fx"
)

const kafkaConfigRetentionMs = "retention.ms"
const noOffset = -1
const maxOffsetZero = 0

func ConfigureKafkaAdmin(
	lgr *logger.LoggerWrapper,
	cfg *config.AppConfig,
	lc fx.Lifecycle,
) (*kadm.Client, error) {
	adm, err := kgo.NewClient(
		kgo.SeedBrokers(cfg.Kafka.BootstrapServers...),
		kgo.MinVersions(kversion.V4_1_0()),
	)
	if err != nil {
		return nil, fmt.Errorf("unable to create admin client: %w", err)
	}
	admCl := kadm.NewClient(adm)
	lc.Append(fx.Hook{
		OnStop: func(ctx context.Context) error {
			lgr.Info("Stopping kafka admin")
			admCl.Close()
			adm.Close()
			return nil
		},
	})
	return admCl, nil
}

func RunCreateTopicChat(
	lgr *logger.LoggerWrapper,
	cfg *config.AppConfig,
	admCl *kadm.Client,
) error {
	retention := cfg.Kafka.TopicChat.Retention
	topicName := cfg.Kafka.TopicChat.Topic
	lgr.Info("Creating topic", "topic", topicName)

	configs := map[string]*string{
		// https://kafka.apache.org/documentation/#topicconfigs_retention.ms
		kafkaConfigRetentionMs: &retention,
	}
	_, err := admCl.CreateTopic(context.Background(), cfg.Kafka.TopicChat.NumPartitions, cfg.Kafka.TopicChat.ReplicationFactor, configs, topicName)

	if errors.Is(err, kerr.TopicAlreadyExists) {
		lgr.Info("Topic is already exists", "topic", topicName)
	} else if err != nil {
		return err
	} else {
		lgr.Info("Topic was successfully created", "topic", topicName)
	}

	return nil
}

func RunCreateTopicUser(
	lgr *logger.LoggerWrapper,
	cfg *config.AppConfig,
	admCl *kadm.Client,
) error {
	retention := cfg.Kafka.TopicUser.Retention
	topicName := cfg.Kafka.TopicUser.Topic
	lgr.Info("Creating topic", "topic", topicName)

	configs := map[string]*string{
		// https://kafka.apache.org/documentation/#topicconfigs_retention.ms
		kafkaConfigRetentionMs: &retention,
	}
	_, err := admCl.CreateTopic(context.Background(), cfg.Kafka.TopicUser.NumPartitions, cfg.Kafka.TopicUser.ReplicationFactor, configs, topicName)

	if errors.Is(err, kerr.TopicAlreadyExists) {
		lgr.Info("Topic is already exists", "topic", topicName)
	} else if err != nil {
		return err
	} else {
		lgr.Info("Topic was successfully created", "topic", topicName)
	}

	return nil
}

func RunDeleteTopicChat(
	lgr *logger.LoggerWrapper,
	cfg *config.AppConfig,
	admCl *kadm.Client,
) error {
	lgr.Warn("Removing topic", "topic", cfg.Kafka.TopicChat.Topic)

	_, err := admCl.DeleteTopic(context.Background(), cfg.Kafka.TopicChat.Topic)
	if err != nil {
		if errors.Is(err, kerr.UnknownTopicOrPartition) {
			lgr.Warn("Topic does not exists", "topic", cfg.Kafka.TopicChat.Topic)
		} else {
			return err
		}
	}
	lgr.Warn("Topic was removed", "topic", cfg.Kafka.TopicChat.Topic)
	return nil
}

func RunDeleteTopicUser(
	lgr *logger.LoggerWrapper,
	cfg *config.AppConfig,
	admCl *kadm.Client,
) error {
	lgr.Warn("Removing topic", "topic", cfg.Kafka.TopicUser.Topic)

	_, err := admCl.DeleteTopic(context.Background(), cfg.Kafka.TopicUser.Topic)
	if err != nil {
		if errors.Is(err, kerr.UnknownTopicOrPartition) {
			lgr.Warn("Topic does not exists", "topic", cfg.Kafka.TopicUser.Topic)
		} else {
			return err
		}
	}
	lgr.Warn("Topic was removed", "topic", cfg.Kafka.TopicUser.Topic)
	return nil
}

func RunResetPartitionsChat(
	lgr *logger.LoggerWrapper,
	cfg *config.AppConfig,
	admCl *kadm.Client,
) error {
	lgr.Info("Start reset partitions", "consumer_group", cfg.Kafka.ConsumerGroupChat)

	_, err := admCl.DeleteGroup(context.Background(), cfg.Kafka.ConsumerGroupChat)

	if err != nil {
		if errors.Is(err, kerr.GroupIDNotFound) {
			lgr.Info("There is no consumer group", "consumer_group", cfg.Kafka.ConsumerGroupChat)
		} else {
			return err
		}
	}

	lgr.Info("Finished reset partitions")

	return nil
}

func RunResetPartitionsUser(
	lgr *logger.LoggerWrapper,
	cfg *config.AppConfig,
	admCl *kadm.Client,
) error {
	lgr.Info("Start reset partitions", "consumer_group", cfg.Kafka.ConsumerGroupUser)

	_, err := admCl.DeleteGroup(context.Background(), cfg.Kafka.ConsumerGroupUser)

	if err != nil {
		if errors.Is(err, kerr.GroupIDNotFound) {
			lgr.Info("There is no consumer group", "consumer_group", cfg.Kafka.ConsumerGroupUser)
		} else {
			return err
		}
	}

	lgr.Info("Finished reset partitions")

	return nil
}

type topicKind int16

const (
	topicKindUnspecified = iota
	topicKindChat
	topicKindUser
)

func (t topicKind) String() string {
	switch t {
	case topicKindUnspecified:
		return "unspecified"
	case topicKindChat:
		return "chat"
	case topicKindUser:
		return "user"
	}
	return "unknown"
}

func WaitForAllEventsProcessedChat(
	lgr *logger.LoggerWrapper,
	cfg *config.AppConfig,
	admCl *kadm.Client,
	lc fx.Lifecycle,
) error {
	return waitForAllEventsProcessed(lgr, cfg, admCl, lc, topicKindChat)
}

func WaitForAllEventsProcessedUser(
	lgr *logger.LoggerWrapper,
	cfg *config.AppConfig,
	admCl *kadm.Client,
	lc fx.Lifecycle,
) error {
	return waitForAllEventsProcessed(lgr, cfg, admCl, lc, topicKindUser)
}

// https://github.com/IBM/sarama/wiki/Frequently-Asked-Questions#how-do-i-consume-until-the-end-of-a-partition
func waitForAllEventsProcessed(
	lgr *logger.LoggerWrapper,
	cfg *config.AppConfig,
	admCl *kadm.Client,
	lc fx.Lifecycle,
	topicKind topicKind,
) error {
	stoppingCtx, cancelFunc := context.WithCancel(context.Background())

	lc.Append(fx.Hook{
		OnStop: func(ctx context.Context) error {
			lgr.Info("Stopping waiter")
			cancelFunc()
			return nil
		},
	})

	du := cfg.Cqrs.CheckAreEventsProcessedInterval

	for {
		lgr.Info("Checking for the current offsets will be equal to the latest ones for all partitions", "topic_kind", topicKind.String())
		isEnd, errE := isEndOnAllPartitions(lgr, cfg, admCl, topicKind)
		if errE != nil {
			lgr.Error("Error during checking isEndOnAllPartitions", logger.AttributeError, errE)
			return errE
		}
		if isEnd {
			lgr.Info("All the events was processed", "topic_kind", topicKind.String())
			cancelFunc()
		} else {
			lgr.Info("The current offsets still aren't equal to the latest ones")
		}

		if errors.Is(stoppingCtx.Err(), context.Canceled) {
			lgr.Info("Exiting from waiter", "topic_kind", topicKind.String())
			break
		} else {
			lgr.Info("Will wait before the next check iteration", "duration", du, "topic_kind", topicKind.String())
			time.Sleep(du)
		}
	}

	return nil
}

func getMaxOffsets(
	lgr *logger.LoggerWrapper,
	cfg *config.AppConfig,
	admCl *kadm.Client,
	topicKind topicKind,
) ([]int64, error) {
	var ktc config.KafkaTopicConfig
	switch topicKind {
	case topicKindChat:
		ktc = cfg.Kafka.TopicChat
	case topicKindUser:
		ktc = cfg.Kafka.TopicUser
	default:
		return nil, fmt.Errorf("Unknown topicKind: %v", topicKind)
	}

	maxOffsets := make([]int64, ktc.NumPartitions)

	lo, err := admCl.ListEndOffsets(context.Background(), ktc.Topic)
	if err != nil {
		return maxOffsets, err
	}

	for i := range ktc.NumPartitions {
		offset, ok := lo.Lookup(ktc.Topic, i)
		if ok {
			maxOffsets[i] = offset.Offset
		} else { // actually not need, for the case
			maxOffsets[i] = noOffset
		}
		lgr.Debug("Got max", "partition", i, "topic", ktc.Topic, "offset", maxOffsets[i])
	}
	return maxOffsets, nil
}

func isEndOnAllPartitions(
	lgr *logger.LoggerWrapper,
	cfg *config.AppConfig,
	admCl *kadm.Client,
	topicKind topicKind,
) (bool, error) {
	maxOffsets, err := getMaxOffsets(lgr, cfg, admCl, topicKind)
	if err != nil {
		if errors.Is(err, kerr.NotLeaderForPartition) {
			return false, nil
		}
		return false, err
	}

	// check are all 0
	// 0 in max means "no messages"
	allZero := true
	for p := range maxOffsets {
		if maxOffsets[p] != maxOffsetZero {
			allZero = false
			break
		}
	}
	if allZero {
		return true, nil
	}

	var consumerGroup string
	var ktc config.KafkaTopicConfig
	switch topicKind {
	case topicKindChat:
		consumerGroup = cfg.Kafka.ConsumerGroupChat
		ktc = cfg.Kafka.TopicChat
	case topicKindUser:
		consumerGroup = cfg.Kafka.ConsumerGroupUser
		ktc = cfg.Kafka.TopicUser
	default:
		return false, fmt.Errorf("Unknown topicKind: %v", topicKind)
	}

	ofs, err := admCl.FetchOffsetsForTopics(context.Background(), consumerGroup, ktc.Topic)
	if err != nil {
		if errors.Is(err, kerr.UnknownTopicOrPartition) {
			return false, nil
		}
		if errors.Is(err, kerr.CoordinatorNotAvailable) {
			return false, nil
		}
		return false, fmt.Errorf("unable to fetch group offsets: %w", err)
	}

	givenOffsets := make([]int64, ktc.NumPartitions)
	for i := range ktc.NumPartitions {
		offs, ok := ofs.Lookup(ktc.Topic, i)
		if ok {
			givenOffsets[i] = offs.Offset.At
		} else { // actually not need, for the case
			givenOffsets[i] = noOffset
		}

		lgr.Debug("Got given", "partition", i, "offset", givenOffsets[i], "topic", offs.Offset.Topic)
	}

	var successful int32 = 0
	for i := range ktc.NumPartitions {
		if maxOffsets[i] == maxOffsetZero && givenOffsets[i] == noOffset {
			successful++
		} else {
			if maxOffsets[i] == givenOffsets[i] {
				successful++
			}
		}
	}

	return successful == ktc.NumPartitions, nil
}

const KeyKey = "key"
const ValueKey = "value"
const MetadataKey = "metadata"
const MetadataOffsetKey = "offset"
const MetadataPartitionKey = "partition"
const HeadersKey = "headers"

func Export(
	lgr *logger.LoggerWrapper,
	cfg *config.AppConfig,
	admCl *kadm.Client,
) error {
	lgr.Info("Start export function")

	maxOffsets, err := getMaxOffsets(lgr, cfg, admCl, topicKindChat)
	if err != nil {
		return err
	}

	finishedPartitions := make([]bool, cfg.Kafka.TopicChat.NumPartitions)

	reqStartOffs := map[int32]kgo.Offset{}
	for i := range cfg.Kafka.TopicChat.NumPartitions {
		partitionMaxOffset := maxOffsets[i]
		if partitionMaxOffset == noOffset || partitionMaxOffset == maxOffsetZero { // actually "partitionMaxOffset == maxOffsetZero" is enough
			lgr.Info("Skipping partition because absence of messages", "partition", i)

			// here we skip empty partitions in order not to hang below
			finishedPartitions[i] = true
			continue
		}

		reqStartOffs[i] = kgo.NewOffset().AtStart()
	}

	cl, err := kgo.NewClient(
		kgo.SeedBrokers(cfg.Kafka.BootstrapServers...),
		kgo.ConsumePartitions(map[string]map[int32]kgo.Offset{
			cfg.Kafka.TopicChat.Topic: reqStartOffs,
		}),
		kgo.MinVersions(kversion.V4_1_0()),
	)
	if err != nil {
		return err
	}
	defer cl.Close()

	var writer io.Writer
	var f *os.File
	if cfg.Cqrs.Export.File == app.PseudoFileStdout {
		writer = os.Stdout
	} else {
		f, err = os.Create(cfg.Cqrs.Export.File)
		if err != nil {
			return err
		}
		writer = f
	}
	if f != nil {
		defer f.Close()
	}

	ctx := context.Background()

	hasUnfinishedPartition := func() bool {
		for _, p := range finishedPartitions {
			if !p {
				return true
			}
		}
		return false
	}

	for hasUnfinishedPartition() {
		fetches := cl.PollFetches(ctx)

		if fetches.IsClientClosed() {
			break
		}
		fetches.EachError(func(t string, p int32, err error) {
			lgr.ErrorContext(ctx, fmt.Sprintf("fetch err topic %s partition %d: %v", t, p, err))
			return
		})

		var perr error
		fetches.EachPartition(func(partition kgo.FetchTopicPartition) {
			partition.EachRecord(func(record *kgo.Record) {
				partitionMaxOffset := maxOffsets[record.Partition]

				kafkaMessage := record

				jsonObj := gabs.New()
				_, err = jsonObj.SetP(kafkaMessage.Offset, MetadataKey+"."+MetadataOffsetKey)
				if err != nil {
					perr = err
					return
				}
				_, err = jsonObj.SetP(kafkaMessage.Partition, MetadataKey+"."+MetadataPartitionKey)
				if err != nil {
					perr = err
					return
				}

				parsedKey := string(kafkaMessage.Key)
				parsedValue, err := gabs.ParseJSON(kafkaMessage.Value)
				if err != nil {
					perr = err
					return
				}

				for _, h := range kafkaMessage.Headers {
					parsedHeaderKey := string(h.Key)
					parsedHeaderValue := string(h.Value)

					_, err = jsonObj.Set(parsedHeaderValue, HeadersKey, parsedHeaderKey)
					if err != nil {
						perr = err
						return
					}
				}

				_, err = jsonObj.Set(parsedKey, KeyKey)
				if err != nil {
					perr = err
					return
				}

				_, err = jsonObj.Set(parsedValue, ValueKey)
				if err != nil {
					perr = err
					return
				}

				_, err = fmt.Fprintln(writer, jsonObj.String())
				if err != nil {
					perr = err
					return
				}

				if kafkaMessage.Offset >= partitionMaxOffset-1 {
					lgr.Info("Reached max offset, closing partitionConsumer", "partition", kafkaMessage.Partition)

					finishedPartitions[record.Partition] = true
					return
				}
			})
		})

		if perr != nil {
			return perr
		}
	}

	lgr.Info("Export function was successfully finished")

	return nil
}

func Import(
	lgr *logger.LoggerWrapper,
	cfg *config.AppConfig,
) error {
	lgr.Info("Start import function")

	cl, err := kgo.NewClient(
		kgo.SeedBrokers(cfg.Kafka.BootstrapServers...),
		kgo.MinVersions(kversion.V4_1_0()),
	)
	if err != nil {
		return err
	}
	defer cl.Close()

	var reader io.Reader
	var f *os.File
	if cfg.Cqrs.Import.File == app.PseudoFileStdin {
		reader = os.Stdin
	} else {
		f, err = os.Open(cfg.Cqrs.Import.File)
		if err != nil {
			return err
		}
		reader = f
	}
	if f != nil {
		defer f.Close()
	}

	scanner := bufio.NewScanner(reader)
	i := 0

	ctx := context.Background()

	for scanner.Scan() {
		i++
		str := scanner.Text()
		jsonObj, err := gabs.ParseJSON([]byte(str))
		if err != nil {
			return fmt.Errorf("Error on reading line %v: %w", i, err)
		}

		kd := jsonObj.S(KeyKey).Data()
		aKey, okk := kd.(string)
		if !okk {
			return fmt.Errorf("Error on parsing key on reading line %v from %v", i, kd)
		}

		aValue := jsonObj.S(ValueKey).Bytes()
		aPartition := jsonObj.S(MetadataKey, MetadataPartitionKey).String()
		partition, err := utils.ParseInt64(aPartition)
		if err != nil {
			return fmt.Errorf("Error on parsing partition on reading line %v: %w", i, err)
		}

		headers := []kgo.RecordHeader{}

		for headerKey, headerValue := range jsonObj.S(HeadersKey).ChildrenMap() {
			hd := headerValue.Data()
			hds, okhv := hd.(string)
			if !okhv {
				return fmt.Errorf("Error on parsing header value on reading line %v from %v for key %v", i, hd, headerKey)
			}
			headers = append(headers, kgo.RecordHeader{
				Key:   headerKey,
				Value: []byte(hds),
			})
		}

		record := &kgo.Record{
			Topic:     cfg.Kafka.TopicChat.Topic,
			Key:       []byte(aKey),
			Headers:   headers,
			Value:     aValue,
			Partition: int32(partition),
		}

		prs := cl.ProduceSync(ctx, record)

		var serr error
		var aerr []error
		for i := range prs {
			if prs[i].Err != nil {
				aerr = append(aerr, prs[i].Err)
			}
		}
		serr = errors.Join(aerr...)
		if serr != nil {
			return fmt.Errorf("Error on sending message from line %v: %w", i, serr)
		}
	}

	lgr.Info("Import function was successfully finished")
	return nil
}
