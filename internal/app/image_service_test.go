package app

import (
	"errors"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"imageProcessor/internal/config"
	"imageProcessor/internal/domain"
	"io"
	"mime/multipart"
	"os"
	"testing"
)

type MockStorage struct {
	mock.Mock
}

func (m *MockStorage) SaveImage(img *domain.Image) error {
	args := m.Called(img)
	return args.Error(0)
}

func (m *MockStorage) GetImage(id string) (*domain.Image, error) {
	args := m.Called(id)
	return args.Get(0).(*domain.Image), args.Error(1)
}

func (m *MockStorage) DeleteImage(id string) error {
	args := m.Called(id)
	return args.Error(0)
}

func (m *MockStorage) SetProcessing(id string) error {
	args := m.Called(id)
	return args.Error(0)
}

func (m *MockStorage) SetProcessed(id string) error {
	args := m.Called(id)
	return args.Error(0)
}

func (m *MockStorage) UploadInProducer() ([]domain.Image, error) {
	args := m.Called()
	return args.Get(0).([]domain.Image), args.Error(1)
}

type MockBroker struct {
	mock.Mock
}

func (m *MockBroker) CreateMessage(img *domain.Image) error {
	args := m.Called(img)
	return args.Error(0)
}

func makeTempFile(t *testing.T, content string) multipart.File {
	tmpFile, err := os.CreateTemp("", "test-*.txt")
	if err != nil {
		t.Fatal(err)
	}
	_, _ = tmpFile.WriteString(content)
	_, _ = tmpFile.Seek(0, io.SeekStart)
	return tmpFile
}

func TestUploadImage(t *testing.T) {
	storage := new(MockStorage)
	broker := new(MockBroker)
	cfg := &config.AppConfig{
		ImageFormats: config.ImageFormats{
			SupportedFormats: map[string]bool{
				"png":  true,
				"jpg":  true,
				"jpeg": true,
				"gif":  true,
			},
		},
		StoragePathConfig: config.StoragePathConfig{
			InputDir:  "./tmp/input/",
			OutputDir: "./tmp/output/",
		},
	}
	defer func() {
		_ = os.RemoveAll("./tmp")
	}()

	service := NewImageService(storage, broker, cfg)

	file := makeTempFile(t, "test")
	defer func() { _ = file.Close() }()
	filename := "test.png"
	watermark := "WM"
	resize := "500x500"

	storage.On("SaveImage", mock.Anything).Return(nil)
	broker.On("CreateMessage", mock.Anything).Return(nil)

	result, err := service.UploadImage(filename, watermark, resize, true, file)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	storage.AssertCalled(t, "SaveImage", mock.Anything)
	broker.AssertCalled(t, "CreateMessage", mock.Anything)
}

func TestUploadImage_ErrorSave(t *testing.T) {
	storage := new(MockStorage)
	broker := new(MockBroker)
	cfg := &config.AppConfig{}
	service := NewImageService(storage, broker, cfg)

	file := makeTempFile(t, "test")
	defer func() { _ = file.Close() }()
	filename := "test.png"
	watermark := "WM"
	resize := "500x500"

	storage.On("SaveImage", mock.Anything).Return(errors.New("fail save"))

	result, err := service.UploadImage(filename, watermark, resize, true, file)
	assert.Error(t, err)
	assert.Nil(t, result)
}

func TestGetImage(t *testing.T) {
	storage := new(MockStorage)
	broker := new(MockBroker)
	cfg := &config.AppConfig{}
	service := NewImageService(storage, broker, cfg)

	id := uuid.New().String()
	img := &domain.Image{ID: uuid.New()}

	storage.On("GetImage", id).Return(img, nil)

	result, err := service.GetImage(id)
	assert.NoError(t, err)
	assert.Equal(t, img, result)
}

func TestGetImage_ParseError(t *testing.T) {
	storage := new(MockStorage)
	service := NewImageService(storage, nil, &config.AppConfig{})

	_, err := service.GetImage("invalid-uuid")
	assert.Error(t, err)
}

func TestDeleteImage(t *testing.T) {
	storage := new(MockStorage)
	service := NewImageService(storage, nil, &config.AppConfig{})
	id := uuid.New().String()

	storage.On("DeleteImage", id).Return(nil)
	err := service.DeleteImage(id)
	assert.NoError(t, err)
}

func TestSetProcessing(t *testing.T) {
	storage := new(MockStorage)
	service := NewImageService(storage, nil, &config.AppConfig{})
	id := uuid.New().String()

	storage.On("SetProcessing", id).Return(nil)
	err := service.SetProcessing(id)
	assert.NoError(t, err)
}

func TestSetProcessed(t *testing.T) {
	storage := new(MockStorage)
	service := NewImageService(storage, nil, &config.AppConfig{})
	id := uuid.New().String()

	storage.On("SetProcessed", id).Return(nil)
	err := service.SetProcessed(id)
	assert.NoError(t, err)
}

func TestUploadInProducer(t *testing.T) {
	storage := new(MockStorage)
	broker := new(MockBroker)
	service := NewImageService(storage, broker, &config.AppConfig{})

	img1 := domain.Image{ID: uuid.New()}
	img2 := domain.Image{ID: uuid.New()}

	storage.On("UploadInProducer").Return([]domain.Image{img1, img2}, nil)
	broker.On("CreateMessage", &img1).Return(nil)
	broker.On("CreateMessage", &img2).Return(nil)

	service.UploadInProducer()

	storage.AssertCalled(t, "UploadInProducer")
	broker.AssertNumberOfCalls(t, "CreateMessage", 2)
}

func TestUploadImage_ErrorProducer(t *testing.T) {
	storage := new(MockStorage)
	broker := new(MockBroker)

	cfg := &config.AppConfig{
		ImageFormats: config.ImageFormats{
			SupportedFormats: map[string]bool{"png": true},
		},
		StoragePathConfig: config.StoragePathConfig{
			InputDir: "./tmp/input/",
		},
	}

	service := NewImageService(storage, broker, cfg)

	file := makeTempFile(t, "data")
	defer func() { _ = file.Close() }()

	storage.On("SaveImage", mock.Anything).Return(nil)
	broker.On("CreateMessage", mock.Anything).Return(errors.New("producer error"))

	result, err := service.UploadImage("test.png", "WM", "100x100", false, file)

	assert.Error(t, err)
	assert.Nil(t, result)
}

func TestGetImage_RepoError(t *testing.T) {
	storage := new(MockStorage)
	service := NewImageService(storage, nil, &config.AppConfig{})

	id := uuid.New().String()

	storage.On("GetImage", id).Return((*domain.Image)(nil), errors.New("get error"))

	result, err := service.GetImage(id)

	assert.Error(t, err)
	assert.Nil(t, result)
}

func TestDeleteImage_Error(t *testing.T) {
	storage := new(MockStorage)
	service := NewImageService(storage, nil, &config.AppConfig{})

	id := uuid.New().String()

	storage.On("DeleteImage", id).Return(errors.New("delete error"))

	err := service.DeleteImage(id)

	assert.Error(t, err)
}

func TestSetProcessing_Error(t *testing.T) {
	storage := new(MockStorage)
	service := NewImageService(storage, nil, &config.AppConfig{})

	id := uuid.New().String()

	storage.On("SetProcessing", id).Return(errors.New("set processing error"))

	err := service.SetProcessing(id)

	assert.Error(t, err)
}

func TestSetProcessed_Error(t *testing.T) {
	storage := new(MockStorage)
	service := NewImageService(storage, nil, &config.AppConfig{})

	id := uuid.New().String()

	storage.On("SetProcessed", id).Return(errors.New("set processed error"))

	err := service.SetProcessed(id)

	assert.Error(t, err)
}

func TestUploadInProducer_RepoError(t *testing.T) {
	storage := new(MockStorage)
	broker := new(MockBroker)
	service := NewImageService(storage, broker, &config.AppConfig{})

	storage.On("UploadInProducer").Return([]domain.Image(nil), errors.New("repo error"))
	service.UploadInProducer()

	storage.AssertCalled(t, "UploadInProducer")
	broker.AssertNotCalled(t, "CreateMessage")
}

func TestUploadInProducer_BrokerError(t *testing.T) {
	storage := new(MockStorage)
	broker := new(MockBroker)
	service := NewImageService(storage, broker, &config.AppConfig{})

	img := domain.Image{ID: uuid.New()}

	storage.On("UploadInProducer").Return([]domain.Image{img}, nil)
	broker.On("CreateMessage", &img).Return(errors.New("broker error"))

	service.UploadInProducer()

	broker.AssertCalled(t, "CreateMessage", &img)
}

func TestUploadImage_NewImageError(t *testing.T) {
	storage := new(MockStorage)
	broker := new(MockBroker)

	cfg := &config.AppConfig{
		ImageFormats: config.ImageFormats{
			SupportedFormats: map[string]bool{
				"png": true,
			},
		},
	}

	service := NewImageService(storage, broker, cfg)

	file := makeTempFile(t, "data")
	defer func() { _ = file.Close() }()

	result, err := service.UploadImage("file.jpg", "", "", false, file)

	assert.Error(t, err)
	assert.Nil(t, result)

	storage.AssertNotCalled(t, "SaveImage")
	broker.AssertNotCalled(t, "CreateMessage")
}

func TestUploadImage_ErrorMkdirAll(t *testing.T) {
	_ = os.RemoveAll("./tmp")
	defer func() {
		_ = os.RemoveAll("./tmp")
	}()

	err := os.WriteFile("./tmp", []byte("block dir"), 0644)
	if err != nil {
		t.Fatal(err)
	}

	storage := new(MockStorage)
	broker := new(MockBroker)

	cfg := &config.AppConfig{
		StoragePathConfig: config.StoragePathConfig{
			InputDir:  "./tmp/input/",
			OutputDir: "./tmp/out/",
		},
		ImageFormats: config.ImageFormats{
			SupportedFormats: map[string]bool{"png": true},
		},
	}

	service := NewImageService(storage, broker, cfg)

	file := makeTempFile(t, "data")
	defer func() { _ = file.Close() }()

	storage.On("SaveImage", mock.Anything).Return(nil)

	result, err := service.UploadImage("file.png", "", "", false, file)

	assert.Error(t, err)
	assert.Nil(t, result)

	broker.AssertNotCalled(t, "CreateMessage")
}

func TestUploadImage_SaveImageError(t *testing.T) {
	storage := new(MockStorage)
	broker := new(MockBroker)

	cfg := &config.AppConfig{
		ImageFormats: config.ImageFormats{
			SupportedFormats: map[string]bool{
				"png": true,
			},
		},
		StoragePathConfig: config.StoragePathConfig{
			InputDir:  "./tmp/input/",
			OutputDir: "./tmp/output/",
		},
	}

	service := NewImageService(storage, broker, cfg)

	file := makeTempFile(t, "test-content")
	defer func() { _ = file.Close() }()

	storage.On("SaveImage", mock.Anything).Return(errors.New("save failed"))

	result, err := service.UploadImage("file.png", "wm", "100x100", false, file)

	assert.Error(t, err)
	assert.Nil(t, result)

	storage.AssertCalled(t, "SaveImage", mock.Anything)

	broker.AssertNotCalled(t, "CreateMessage")
}
