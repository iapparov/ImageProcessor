package web

import (
	"bytes"
	"encoding/json"
	"errors"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"imageProcessor/internal/config"
	"imageProcessor/internal/domain"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"
)

type MockImageService struct {
	mock.Mock
}

func (m *MockImageService) UploadImage(filename, watermark, resize string, mini bool, file multipart.File) (*domain.Image, error) {
	args := m.Called(filename, watermark, resize, mini, file)
	return args.Get(0).(*domain.Image), args.Error(1)
}

func (m *MockImageService) GetImage(id string) (*domain.Image, error) {
	args := m.Called(id)
	return args.Get(0).(*domain.Image), args.Error(1)
}

func (m *MockImageService) DeleteImage(id string) error {
	args := m.Called(id)
	return args.Error(0)
}

func TestUploadImage_Success(t *testing.T) {
	mockSvc := new(MockImageService)
	cfg := &config.AppConfig{
		StoragePathConfig: config.StoragePathConfig{OutputDir: "/tmp/"},
	}
	handler := NewCommentHandler(mockSvc, cfg)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, _ := writer.CreateFormFile("file", "test.png")
	_, _ = part.Write([]byte("data"))
	_ = writer.WriteField("watermark", "WM")
	_ = writer.WriteField("resize", "500x500")
	_ = writer.WriteField("mini", "1")
	_ = writer.Close()

	req := httptest.NewRequest("POST", "/api/upload", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	w := httptest.NewRecorder()

	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = req

	img := &domain.Image{
		Name:      "test.png",
		Status:    domain.Created,
		Format:    "png",
		Watermark: "WM",
		Resize:    &domain.Resize{Width: 500, Height: 500},
		Mini:      true,
	}

	mockSvc.
		On("UploadImage", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(img, nil)

	handler.UploadImage(ctx)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp ImageResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, img.Name, resp.Name)
}

func TestUploadImage_ErrorUpload(t *testing.T) {
	mockSvc := new(MockImageService)
	cfg := &config.AppConfig{}
	handler := NewCommentHandler(mockSvc, cfg)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, _ := writer.CreateFormFile("file", "test.png")
	_, _ = part.Write([]byte("data"))
	_ = writer.WriteField("watermark", "WM")
	_ = writer.WriteField("resize", "500x500")
	_ = writer.WriteField("mini", "1")
	_ = writer.Close()

	req := httptest.NewRequest("POST", "/api/upload", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	w := httptest.NewRecorder()

	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = req

	mockSvc.
		On("UploadImage", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return((*domain.Image)(nil), errors.New("fail"))

	handler.UploadImage(ctx)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestGetImage_Processing(t *testing.T) {
	mockSvc := new(MockImageService)
	cfg := &config.AppConfig{}
	handler := NewCommentHandler(mockSvc, cfg)

	img := &domain.Image{
		Name:   "test.png",
		Status: domain.Processing,
	}

	mockSvc.On("GetImage", "1").Return(img, nil)

	req := httptest.NewRequest("GET", "/api/image/1", nil)
	w := httptest.NewRecorder()

	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = req
	ctx.Params = gin.Params{gin.Param{Key: "id", Value: "1"}}

	handler.GetImage(ctx)
	assert.Equal(t, http.StatusAccepted, w.Code)
}

func TestDeleteImage_Success(t *testing.T) {
	mockSvc := new(MockImageService)
	handler := &ImageHandler{
		imageProcessor: mockSvc,
	}

	mockSvc.On("DeleteImage", "1").Return(nil)

	req := httptest.NewRequest("DELETE", "/api/image/1", nil)
	w := httptest.NewRecorder()

	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = req
	ctx.Params = gin.Params{
		{Key: "id", Value: "1"},
	}

	handler.DeleteImage(ctx)

	assert.Equal(t, http.StatusNoContent, w.Code)
	mockSvc.AssertCalled(t, "DeleteImage", "1")
}

func TestDeleteImage_Error(t *testing.T) {
	mockSvc := new(MockImageService)
	handler := &ImageHandler{
		imageProcessor: mockSvc,
	}

	mockSvc.On("DeleteImage", "1").Return(errors.New("fail"))

	req := httptest.NewRequest("DELETE", "/api/image/1", nil)
	w := httptest.NewRecorder()

	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = req
	ctx.Params = gin.Params{
		{Key: "id", Value: "1"},
	}

	handler.DeleteImage(ctx)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	mockSvc.AssertCalled(t, "DeleteImage", "1")
}

func TestUploadImage_Errors(t *testing.T) {
	mockSvc := new(MockImageService)
	handler := &ImageHandler{
		imageProcessor: mockSvc,
		cfg: &config.AppConfig{
			StoragePathConfig: config.StoragePathConfig{OutputDir: "/tmp/"},
		},
	}

	t.Run("bind error", func(t *testing.T) {
		w := httptest.NewRecorder()
		ctx, _ := gin.CreateTestContext(w)

		ctx.Request = httptest.NewRequest("POST", "/api/upload", nil)

		handler.UploadImage(ctx)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("form file error", func(t *testing.T) {
		w := httptest.NewRecorder()
		ctx, _ := gin.CreateTestContext(w)

		body := &bytes.Buffer{}
		ctx.Request = httptest.NewRequest("POST", "/api/upload", body)
		ctx.Request.Header.Set("Content-Type", "multipart/form-data; boundary=abc")

		handler.UploadImage(ctx)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("service error", func(t *testing.T) {
		w := httptest.NewRecorder()
		ctx, _ := gin.CreateTestContext(w)

		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)
		part, _ := writer.CreateFormFile("file", "test.png")
		_, _ = part.Write([]byte("data"))
		_ = writer.WriteField("watermark", "WM")
		_ = writer.WriteField("resize", "500x500")
		_ = writer.WriteField("mini", "1")
		_ = writer.Close()

		ctx.Request = httptest.NewRequest("POST", "/api/upload", body)
		ctx.Request.Header.Set("Content-Type", writer.FormDataContentType())

		mockSvc.On("UploadImage", mock.Anything, "WM", "500x500", true, mock.Anything).
			Return((*domain.Image)(nil), errors.New("fail"))

		handler.UploadImage(ctx)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}
