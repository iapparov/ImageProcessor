package domain

import (
	"errors"
	"github.com/google/uuid"
	"imageProcessor/internal/config"
	"strconv"
	"strings"
	"time"
)

type StatusType string

const (
	Created    StatusType = "created"
	Processing StatusType = "processing"
	Processed  StatusType = "processed"
	Deleted    StatusType = "deleted"
)

type Image struct {
	ID        uuid.UUID  `json:"id"`
	CreatedAt time.Time  `json:"created_at"`
	Status    StatusType `json:"status"`
	Format    string     `json:"format"`
	Name      string     `json:"name"`
	Watermark string     `json:"watermark"`
	Resize    *Resize    `json:"resize"`
	Mini      bool       `json:"mini"`
}

type Resize struct {
	Width  int `json:"width"`
	Height int `json:"height"`
}

func NewImage(frmt, watermark, resize string, mini bool, cfg *config.AppConfig) (*Image, error) {

	if err := paramsValidation(watermark, resize); err != nil {
		return nil, err
	}
	if !cfg.ImageFormats.SupportedFormats[frmt] {
		return nil, errors.New("unsupported format:" + frmt)
	}

	var resizeStruct *Resize

	if resize != "" {
		w, h, err := parseResize(resize)
		if err != nil {
			return nil, err
		}
		resizeStruct = &Resize{Width: w, Height: h}
	} else {
		resizeStruct = &Resize{Width: 0, Height: 0}
	}

	img := &Image{
		ID:        uuid.New(),
		CreatedAt: time.Now(),
		Status:    Created,
		Format:    frmt,
		Name:      uuid.New().String() + "." + frmt,
		Watermark: watermark,
		Resize:    resizeStruct,
		Mini:      mini,
	}

	return img, nil
}

func paramsValidation(watermark, resize string) error {

	if watermark != "" && len(watermark) > 20 {
		return errors.New("watermark must be less than or equal to 20 characters")
	}

	if resize != "" {
		_, _, err := parseResize(resize)
		if err != nil {
			return err
		}
	}

	return nil
}

func parseResize(s string) (int, int, error) {
	parts := strings.Split(s, "x")
	if len(parts) != 2 {
		return 0, 0, errors.New("resize must be in format WIDTHxHEIGHT, e.g. 1024x768, u have:" + s)
	}

	width, err := strconv.Atoi(parts[0])
	if err != nil || width <= 0 {
		return 0, 0, errors.New("resize width must be a positive integer")
	}

	height, err := strconv.Atoi(parts[1])
	if err != nil || height <= 0 {
		return 0, 0, errors.New("resize height must be a positive integer")
	}

	return width, height, nil
}
