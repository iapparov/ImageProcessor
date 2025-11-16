package app

import (
	"github.com/google/uuid"
	wbzlog "github.com/wb-go/wbf/zlog"
	"imageProcessor/internal/config"
	"imageProcessor/internal/domain"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
)

type ImageService struct {
	repo     StorageProvider
	producer BrokerProvider
	config   *config.AppConfig
}

type StorageProvider interface {
	SaveImage(img *domain.Image) error
	GetImage(id string) (*domain.Image, error)
	DeleteImage(id string) error
	SetProcessing(id string) error
	SetProcessed(id string) error
	UploadInProducer() ([]domain.Image, error)
}

type BrokerProvider interface {
	CreateMessage(*domain.Image) error
}

func NewImageService(repo StorageProvider, producer BrokerProvider, config *config.AppConfig) *ImageService {
	return &ImageService{
		repo:     repo,
		producer: producer,
		config:   config,
	}
}

func (s *ImageService) UploadImage(filename, watermark, resize string, mini bool, file multipart.File) (*domain.Image, error) {

	format := strings.Split(filepath.Ext(filename), ".")[1]

	img, err := domain.NewImage(strings.ToLower(format), watermark, resize, mini, s.config)
	if err != nil {
		wbzlog.Logger.Error().Err(err).Msg("Failed to create new image model")
		return nil, err
	}

	err = s.repo.SaveImage(img)
	if err != nil {
		wbzlog.Logger.Error().Err(err).Msg("Failed to save image metadata to storage")
		return nil, err
	}

	outDir := s.config.StoragePathConfig.InputDir

	if err := os.MkdirAll(outDir, 0755); err != nil {
		wbzlog.Logger.Error().Err(err).Msg("Failed to create input directory")
		return nil, err
	}

	out, err := os.Create(filepath.Join(outDir, img.Name))
	if err != nil {
		wbzlog.Logger.Error().Err(err).Msg("Failed to create image file in storage")
		return nil, err
	}
	defer func() {
		_ = out.Close()
	}()

	_, err = io.Copy(out, file)
	if err != nil {
		wbzlog.Logger.Error().Err(err).Msg("Failed to save uploaded file")
		return nil, err
	}

	err = s.producer.CreateMessage(img)
	if err != nil {
		wbzlog.Logger.Error().Err(err).Msg("Failed to send into kafka producer uploaded file")
		return nil, err
	}

	return img, nil
}

func (s *ImageService) GetImage(id string) (*domain.Image, error) {
	_, err := idParse(id)
	if err != nil {
		wbzlog.Logger.Error().Err(err).Msg("Failed to parse image ID")
		return nil, err
	}
	img, err := s.repo.GetImage(id)
	if err != nil {
		wbzlog.Logger.Error().Err(err).Msg("Failed to get image metadata from storage")
		return nil, err
	}
	return img, nil
}

func (s *ImageService) DeleteImage(id string) error {
	_, err := idParse(id)
	if err != nil {
		wbzlog.Logger.Error().Err(err).Msg("Failed to parse image ID")
		return err
	}
	err = s.repo.DeleteImage(id)
	if err != nil {
		wbzlog.Logger.Error().Err(err).Msg("Failed to delete image metadata from storage")
		return err
	}
	return nil
}

func (s *ImageService) SetProcessing(id string) error {
	_, err := idParse(id)
	if err != nil {
		wbzlog.Logger.Error().Err(err).Msg("Failed to set image status to processing")
		return err
	}
	return s.repo.SetProcessing(id)
}

func (s *ImageService) SetProcessed(id string) error {
	_, err := idParse(id)
	if err != nil {
		wbzlog.Logger.Error().Err(err).Msg("Failed to set image status to processed")
		return err
	}
	return s.repo.SetProcessed(id)
}

func idParse(id string) (*uuid.UUID, error) {
	uid, err := uuid.Parse(id)
	if err != nil {
		return &uuid.Nil, err
	}
	return &uid, nil
}

func (s *ImageService) UploadInProducer() {
	images, err := s.repo.UploadInProducer()
	if err != nil {
		wbzlog.Logger.Error().Err(err).Msg("Failed to get images from storage for producer")
		return
	}
	for _, img := range images {
		err := s.producer.CreateMessage(&img)
		if err != nil {
			wbzlog.Logger.Error().Err(err).Msg("Failed to send image to kafka producer")
		}
	}
}
