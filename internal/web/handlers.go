package web

import (
	wbgin "github.com/wb-go/wbf/ginext"
	"imageProcessor/internal/config"
	"imageProcessor/internal/domain"
	"mime/multipart"
	"net/http"
)

// ImageReqUpload представляет параметры запроса на загрузку изображения
type ImageReqUpload struct {
	Resize    string `form:"resize" example:"500x500" description:"Размер изображения в формате WIDTHxHEIGHT"`
	Mini      string `form:"mini" example:"1" description:"Создать миниатюру, 1 = да, 0 = нет"`
	Watermark string `form:"watermark" example:"Мой Водяной Знак" description:"Текст водяного знака"`
}

// ImageResponse представляет ответ с информацией об изображении
type ImageResponse struct {
	ID     string `json:"ID" example:"123e4567-e89b-12d3-a456-426614174000" description:"Уникальный идентификатор изображения"`
	Name   string `json:"Name" example:"example.png" description:"Имя файла"`
	Status string `json:"Status" example:"Processing" description:"Статус обработки изображения"`
	URL    string `json:"URL" example:"/data_img/processed/example.png" description:"URL обработанного изображения"`
}

type ImageHandler struct {
	imageProcessor ImageProcessorProvider
	cfg            *config.AppConfig
}

type ImageProcessorProvider interface {
	UploadImage(filename, watermark, resize string, mini bool, file multipart.File) (*domain.Image, error)
	GetImage(id string) (*domain.Image, error)
	DeleteImage(id string) error
}

func NewCommentHandler(imageProcessor ImageProcessorProvider, cfg *config.AppConfig) *ImageHandler {
	return &ImageHandler{
		imageProcessor: imageProcessor,
		cfg:            cfg,
	}
}

// ErrorResponse представляет стандартную ошибку API
type ErrorResponse struct {
	Error string `json:"error" example:"invalid input data"`
}

// UploadImage godoc
// @Summary Загрузка изображения
// @Description Загружает изображение и ставит его на обработку (ресайз, миниатюра, водяной знак)
// @Tags Images
// @Accept multipart/form-data
// @Produce json
// @Param file formData file true "Image file"
// @Param watermark formData string false "Watermark text"
// @Param resize formData string false "Resize in format WIDTHxHEIGHT, e.g., 500x500"
// @Param mini formData string false "Generate thumbnail, 1 = true, 0 = false"
// @Success 200 {object} ImageResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/upload [post]
func (h *ImageHandler) UploadImage(ctx *wbgin.Context) {

	var req ImageReqUpload
	if err := ctx.ShouldBind(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, wbgin.H{"error": err.Error()})
		return
	}
	file, err := ctx.FormFile("file")
	if err != nil {
		ctx.JSON(http.StatusBadRequest, wbgin.H{"error": err.Error()})
		return
	}
	var m bool
	if req.Mini == "1" {
		m = true
	}

	f, err := file.Open()
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, wbgin.H{"error": err.Error()})
		return
	}
	defer func() {
		err := f.Close()
		if err != nil {
			ctx.JSON(http.StatusInternalServerError, wbgin.H{"error": err.Error()})
			return
		}
	}()

	img, err := h.imageProcessor.UploadImage(file.Filename, req.Watermark, req.Resize, m, f)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, wbgin.H{"error": err.Error()})
		return
	}

	resp := ImageResponse{
		ID:     img.ID.String(),
		Name:   img.Name,
		Status: string(img.Status),
		URL:    h.cfg.StoragePathConfig.OutputDir + img.Name,
	}
	ctx.JSON(http.StatusOK, resp)
}

// GetImage godoc
// @Summary Получение изображения
// @Description Возвращает обработанное изображение, если оно готово, иначе — статус обработки
// @Tags Images
// @Produce json
// @Param id path string true "Image ID"
// @Success 200 {file} file "Processed image file"
// @Success 202 {object} ImageResponse "Processing status"
// @Failure 500 {object} ErrorResponse
// @Router /api/image/{id} [get]
func (h *ImageHandler) GetImage(ctx *wbgin.Context) {
	id := ctx.Param("id")

	img, err := h.imageProcessor.GetImage(id)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, wbgin.H{"error": err.Error()})
		return
	}
	if img.Status == domain.Processed {
		ctx.File(h.cfg.StoragePathConfig.OutputDir + img.Name)
		return
	}
	resp := ImageResponse{
		ID:     img.ID.String(),
		Name:   img.Name,
		Status: string(img.Status),
		URL:    h.cfg.StoragePathConfig.OutputDir + img.Name,
	}
	ctx.JSON(http.StatusAccepted, resp)
}

// DeleteImage godoc
// @Summary Удаление изображения
// @Description Удаляет изображение из хранилища (помечает, как удаленное)
// @Tags Images
// @Param id path string true "Image ID"
// @Success 204 {string} string "No Content"
// @Failure 500 {object} ErrorResponse
// @Router /api/image/{id} [delete]
func (h *ImageHandler) DeleteImage(ctx *wbgin.Context) {
	id := ctx.Param("id")
	err := h.imageProcessor.DeleteImage(id)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, wbgin.H{"error": err.Error()})
		return
	}
	ctx.Status(http.StatusNoContent)
	ctx.Writer.WriteHeaderNow()
}
