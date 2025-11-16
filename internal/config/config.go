package config

import (
	"fmt"
	wbfconfig "github.com/wb-go/wbf/config"
	wbzlog "github.com/wb-go/wbf/zlog"
	"os"
	"strings"
	"time"
)

type AppConfig struct {
	ServerConfig      ServerConfig      `mapstructure:"server"`
	LoggerConfig      loggerConfig      `mapstructure:"logger"`
	DBConfig          dbConfig          `mapstructure:"db_config"`
	RetrysConfig      RetrysConfig      `mapstructure:"retry_strategy"`
	GinConfig         ginConfig         `mapstructure:"gin"`
	KafkaConfig       kafkaConfig       `mapstructure:"kafka"`
	StoragePathConfig StoragePathConfig `mapstructure:"storage_path"`
	ImageFormats      ImageFormats      `mapstructure:",squash"`
}

type ImageFormats struct {
	Formats          []string        `mapstructure:"img_formats"`
	SupportedFormats map[string]bool `mapstructure:"-"`
}

type StoragePathConfig struct {
	InputDir  string `mapstructure:"input_dir" default:"./data_img/original/"`
	OutputDir string `mapstructure:"output_dir" default:"./data/img/processed/"`
}

type kafkaConfig struct {
	Brokers               []string `mapstructure:"brokers"`
	Group_id              string   `mapstructure:"group_id"`
	Topic                 string   `mapstructure:"topic"`
	Consumer_worker_count int      `mapstructure:"consumer_worker_count" default:"4"`
}

type RetrysConfig struct {
	Attempts int           `mapstructure:"attempts" default:"3"`
	Delay    time.Duration `mapstructure:"delay" default:"1s"`
	Backoffs float64       `mapstructure:"backoffs" default:"2"`
}

type ginConfig struct {
	Mode string `mapstructure:"mode" default:"debug"`
}

type ServerConfig struct {
	Host string `mapstructure:"host" default:"localhost"`
	Port int    `mapstructure:"port" default:"8080"`
}

type loggerConfig struct {
	Level string `mapstructure:"level" default:"info"`
}

type postgresConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	User     string `mapstructure:"user"`
	Password string `mapstructure:"password"`
	DBName   string `mapstructure:"db_name"`
	SSLMode  string `mapstructure:"ssl_mode" default:"disable"`
}

type dbConfig struct {
	Master          postgresConfig   `mapstructure:"postgres"`
	Slaves          []postgresConfig `mapstructure:"slaves"`
	MaxOpenConns    int              `mapstructure:"maxOpenConns"`
	MaxIdleConns    int              `mapstructure:"maxIdleConns"`
	ConnMaxLifetime time.Duration    `mapstructure:"connMaxLifetime"`
}

func NewAppConfig() (*AppConfig, error) {
	envFilePath := "./.env"
	appConfigFilePath := "./config/local.yaml"

	cfg := wbfconfig.New()

	// Загрузка .env файлов
	if err := cfg.LoadEnvFiles(envFilePath); err != nil {
		wbzlog.Logger.Fatal().Err(err).Msg("Failed to load env files")
		return nil, fmt.Errorf("failed to load env files: %w", err)
	}

	// Включение поддержки переменных окружения
	cfg.EnableEnv("")

	if err := cfg.LoadConfigFiles(appConfigFilePath); err != nil {
		wbzlog.Logger.Fatal().Err(err).Msg("Failed to load config files")
		return nil, fmt.Errorf("failed to load config files: %w", err)
	}

	var appCfg AppConfig
	if err := cfg.Unmarshal(&appCfg); err != nil {
		wbzlog.Logger.Fatal().Err(err).Msg("Failed to unmarshal config")
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	appCfg.DBConfig.Master.DBName = os.Getenv("POSTGRES_DB")
	appCfg.DBConfig.Master.User = os.Getenv("POSTGRES_USER")
	appCfg.DBConfig.Master.Password = os.Getenv("POSTGRES_PASSWORD")
	appCfg.ImageFormats.SupportedFormats = configFormats(appCfg.ImageFormats.Formats)
	return &appCfg, nil
}

func configFormats(formats []string) map[string]bool {
	imgFormats := make(map[string]bool, len(formats))
	for _, f := range formats {
		imgFormats[strings.ToLower(f)] = true
	}
	return imgFormats
}
