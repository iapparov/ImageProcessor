// @title           imageProcessor API
// @version         1.0
// @description     API для обработки изображений
// @BasePath        /

package main

import (
	wbzlog "github.com/wb-go/wbf/zlog"
	"go.uber.org/fx"
	"imageProcessor/internal/app"
	"imageProcessor/internal/broker/kafka_producer"
	"imageProcessor/internal/config"
	"imageProcessor/internal/di"
	"imageProcessor/internal/storage/db"
	"imageProcessor/internal/web"
)

func main() {
	wbzlog.Init()
	app := fx.New(
		fx.Provide(
			config.NewAppConfig,
			db.NewPostgres,

			func(db *db.Postgres) app.StorageProvider {
				return db
			},

			kafkaproducer.NewKafkaProducer,
			func(kafka *kafkaproducer.KafkaProducerService) app.BrokerProvider {
				return kafka
			},

			app.NewImageService,

			func(service *app.ImageService) web.ImageProcessorProvider {
				return service
			},
			web.NewCommentHandler,
		),
		fx.Invoke(
			di.StartHTTPServer,
			di.StartKafkaProducer,
			di.StartKafkaConsumer,
			di.ClosePostgresOnStop,
		),
	)

	app.Run()
}
