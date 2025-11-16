package di

import (
	"context"
	"fmt"
	wbgin "github.com/wb-go/wbf/ginext"
	"go.uber.org/fx"
	"imageProcessor/internal/app"
	kafkaconsumer "imageProcessor/internal/broker/kafka_consumer"
	kafkaproducer "imageProcessor/internal/broker/kafka_producer"
	"imageProcessor/internal/config"
	"imageProcessor/internal/storage/db"
	"imageProcessor/internal/web"
	"log"
	"net/http"
)

func StartHTTPServer(lc fx.Lifecycle, imageHandler *web.ImageHandler, config *config.AppConfig) {
	router := wbgin.New(config.GinConfig.Mode)

	router.Use(wbgin.Logger(), wbgin.Recovery())
	router.Use(func(c *wbgin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, DELETE")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	})

	web.RegisterRoutes(router, imageHandler)

	addres := fmt.Sprintf("%s:%d", config.ServerConfig.Host, config.ServerConfig.Port)
	server := &http.Server{
		Addr:    addres,
		Handler: router.Engine,
	}

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			log.Printf("Server started")
			go func() {
				if err := server.ListenAndServe(); err != nil {
					log.Printf("ListenAndServe error: %v", err)
				}
			}()
			return nil
		},
		OnStop: func(ctx context.Context) error {
			log.Printf("Shutting down server...")
			return server.Close()
		},
	})
}

func StartKafkaProducer(lc fx.Lifecycle, k *kafkaproducer.KafkaProducerService, s *app.ImageService) {
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			log.Println("Start Kafka Producer...")
			go s.UploadInProducer()
			lc.Append(fx.Hook{
				OnStop: func(ctx context.Context) error {
					log.Println("Stopping Kafka producer")
					err := k.Close()
					if err != nil {
						log.Printf("Failed to close Kafka producer: %v", err)
					}
					return nil
				},
			})
			log.Println("Kafka Producer started successfully")
			return nil
		},
	})
}

func StartKafkaConsumer(lc fx.Lifecycle, cfg *config.AppConfig, s *app.ImageService) {
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			log.Println("Start Kafka Consumer...")

			consumerCtx, cancel := context.WithCancel(context.Background())
			go kafkaconsumer.Consuming(consumerCtx, cfg, s)

			lc.Append(fx.Hook{
				OnStop: func(ctx context.Context) error {
					log.Println("Stopping Kafka Consumer...")
					cancel()

					return nil
				},
			})

			log.Println("Kafka Consumer started successfully")
			return nil
		},
	})
}

func ClosePostgresOnStop(lc fx.Lifecycle, postgres *db.Postgres) {
	lc.Append(fx.Hook{
		OnStop: func(ctx context.Context) error {
			log.Println("Closing Postgres connections...")
			if err := postgres.Close(); err != nil {
				log.Printf("Failed to close Postgres: %v", err)
				return err
			}
			log.Println("Postgres closed successfully")
			return nil
		},
	})
}
