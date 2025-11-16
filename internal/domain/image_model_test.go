package domain

import (
	"github.com/stretchr/testify/assert"
	"imageProcessor/internal/config"
	"testing"
)

func TestParseResize_Valid(t *testing.T) {
	width, height, err := parseResize("500x400")
	assert.NoError(t, err)
	assert.Equal(t, 500, width)
	assert.Equal(t, 400, height)
}

func TestParseResize_InvalidFormat(t *testing.T) {
	_, _, err := parseResize("500-400")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "resize must be in format")
}

func TestParseResize_NonInteger(t *testing.T) {
	_, _, err := parseResize("abcx400")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "resize width must be a positive integer")
}

func TestParseResize_ZeroOrNegative(t *testing.T) {
	_, _, err := parseResize("0x100")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "resize width must be a positive integer")

	_, _, err = parseResize("100x-1")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "resize height must be a positive integer")
}

func TestParamsValidation_WatermarkTooLong(t *testing.T) {
	err := paramsValidation("thisisaverylongwatermarktext", "500x500")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "watermark must be less than or equal to 20 characters")
}

func TestParamsValidation_InvalidResize(t *testing.T) {
	err := paramsValidation("WM", "500-500")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "resize must be in format")
}

func TestParamsValidation_Valid(t *testing.T) {
	err := paramsValidation("WM", "500x500")
	assert.NoError(t, err)

	err = paramsValidation("", "")
	assert.NoError(t, err)
}

func TestNewImage_Valid(t *testing.T) {
	cfg := &config.AppConfig{
		ImageFormats: config.ImageFormats{
			SupportedFormats: map[string]bool{"png": true, "jpg": true},
		},
	}
	img, err := NewImage("png", "WM", "500x500", true, cfg)
	assert.NoError(t, err)
	assert.NotNil(t, img)
	assert.Equal(t, "WM", img.Watermark)
	assert.Equal(t, true, img.Mini)
	assert.Equal(t, 500, img.Resize.Width)
	assert.Equal(t, 500, img.Resize.Height)
}

func TestNewImage_UnsupportedFormat(t *testing.T) {
	cfg := &config.AppConfig{
		ImageFormats: config.ImageFormats{
			SupportedFormats: map[string]bool{"png": true, "jpg": true},
		},
	}
	img, err := NewImage("gif", "WM", "500x500", false, cfg)
	assert.Error(t, err)
	assert.Nil(t, img)
	assert.Contains(t, err.Error(), "unsupported format")
}

func TestNewImage_InvalidWatermark(t *testing.T) {
	cfg := &config.AppConfig{
		ImageFormats: config.ImageFormats{
			SupportedFormats: map[string]bool{"png": true},
		},
	}
	img, err := NewImage("png", "thisisaverylongwatermarktext", "500x500", false, cfg)
	assert.Error(t, err)
	assert.Nil(t, img)
}

func TestNewImage_InvalidResize(t *testing.T) {
	cfg := &config.AppConfig{
		ImageFormats: config.ImageFormats{
			SupportedFormats: map[string]bool{"png": true},
		},
	}
	img, err := NewImage("png", "WM", "500-500", false, cfg)
	assert.Error(t, err)
	assert.Nil(t, img)
}
