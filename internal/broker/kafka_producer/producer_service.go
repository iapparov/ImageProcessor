package kafkaproducer

import (
	"context"
	"encoding/json"
	"github.com/segmentio/kafka-go"
	wbkafka "github.com/wb-go/wbf/kafka"
	wbretry "github.com/wb-go/wbf/retry"
	wbzlog "github.com/wb-go/wbf/zlog"
	"imageProcessor/internal/config"
	"imageProcessor/internal/domain"
)

type KafkaProducerService struct {
	producer *wbkafka.Producer
	cfg      *config.AppConfig
}

func NewKafkaProducer(cfg *config.AppConfig) *KafkaProducerService {
	wbzlog.Logger.Info().Msgf("Kafka brokers: %v, topic: %s", cfg.KafkaConfig.Brokers, cfg.KafkaConfig.Topic)
	conn, _ := kafka.Dial("tcp", "localhost:9092")
	defer func() {
		err := conn.Close()
		if err != nil {
			wbzlog.Logger.Error().Err(err).Msg("failed to close kafka connection")
		}
	}()

	err := conn.CreateTopics(kafka.TopicConfig{
		Topic:             cfg.KafkaConfig.Topic,
		NumPartitions:     1,
		ReplicationFactor: 1,
	})
	if err != nil {
		wbzlog.Logger.Error().Err(err).Msg("failed to create kafka topic")
	}
	return &KafkaProducerService{
		producer: wbkafka.NewProducer(cfg.KafkaConfig.Brokers, cfg.KafkaConfig.Topic),
		cfg:      cfg,
	}
}

func (k *KafkaProducerService) Close() error {
	return k.producer.Close()
}

func (k *KafkaProducerService) CreateMessage(img *domain.Image) error {
	msg, err := json.Marshal(img)
	if err != nil {
		wbzlog.Logger.Error().Err(err).Msg("invalid message for kafka producer")
		return err
	}
	ctx := context.Background()
	err = k.producer.SendWithRetry(ctx,
		wbretry.Strategy{Attempts: k.cfg.RetrysConfig.Attempts, Delay: k.cfg.RetrysConfig.Delay, Backoff: k.cfg.RetrysConfig.Backoffs},
		[]byte(img.ID.String()),
		msg)
	if err != nil {
		wbzlog.Logger.Error().Err(err).Msg("bad send request kafka producer")
		return err
	}
	return nil
}
