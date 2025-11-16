package kafkaconsumer

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/segmentio/kafka-go"
	wbkafka "github.com/wb-go/wbf/kafka"
	wbretry "github.com/wb-go/wbf/retry"
	wbzlog "github.com/wb-go/wbf/zlog"
	"imageProcessor/internal/app"
	"imageProcessor/internal/config"
	"imageProcessor/internal/domain"
	"imageProcessor/internal/imgprocessor"
	"sync"
)

type KafkaConsumerService struct {
	consumer *wbkafka.Consumer
}

func NewConsumer(cfg *config.AppConfig) *KafkaConsumerService {
	return &KafkaConsumerService{
		consumer: wbkafka.NewConsumer(cfg.KafkaConfig.Brokers, cfg.KafkaConfig.Topic, cfg.KafkaConfig.Group_id),
	}
}

func (c *KafkaConsumerService) Close() error {
	return c.consumer.Close()
}

func Consuming(ctx context.Context, cfg *config.AppConfig, imageService *app.ImageService) {
	var wg sync.WaitGroup
	out := make(chan kafka.Message)
	consumer := NewConsumer(cfg)
	defer func() {
		err := consumer.Close()
		if err != nil {
			wbzlog.Logger.Error().Err(err).Msg("failed to close kafka consumer")
		}
	}()
	go consumer.consumer.StartConsuming(ctx, out, wbretry.Strategy{Attempts: cfg.RetrysConfig.Attempts, Delay: cfg.RetrysConfig.Delay, Backoff: cfg.RetrysConfig.Backoffs})
	for i := 0; i < cfg.KafkaConfig.Consumer_worker_count; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					wbzlog.Logger.Info().Msg(fmt.Sprintf("Worker %d stopping...", workerID))
					return
				case msg, ok := <-out:
					if !ok {
						wbzlog.Logger.Info().Msg("Consumer channel closed, worker stopping")
						return
					}
					err := imageService.SetProcessing(string(msg.Key))
					if err != nil {
						wbzlog.Logger.Error().Err(err).Msg("failed to update image status to processing")
						continue
					}
					var task domain.Image
					if err := json.Unmarshal(msg.Value, &task); err != nil {
						wbzlog.Logger.Error().Err(err).Msg("invalid task in kafka consumer")
						continue
					}
					err = imgprocessor.Process(cfg, &task)
					if err != nil {
						wbzlog.Logger.Error().Err(err).Msg("image processing error")
						continue
					}
					err = imageService.SetProcessed(string(msg.Key))
					if err != nil {
						wbzlog.Logger.Error().Err(err).Msg("failed to update image status to processed")
						continue
					}
					err = consumer.consumer.Commit(ctx, msg)
					if err != nil {
						wbzlog.Logger.Error().Err(err).Msg("failed to commit message")
					}
				}
			}
		}(i + 1)
	}
	wg.Wait()
}
