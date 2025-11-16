package imgprocessor

import (
	"github.com/stretchr/testify/assert"
	"image"
	"image/color"
	"image/gif"
	"image/jpeg"
	"image/png"
	"imageProcessor/internal/config"
	"imageProcessor/internal/domain"
	"os"
	"path/filepath"
	"testing"
)

func createTempImageByFormat(t *testing.T, dir, name, format string) string {
	t.Helper()
	path := filepath.Join(dir, name)

	img := image.NewRGBA(image.Rect(0, 0, 120, 120))
	for y := 0; y < 120; y++ {
		for x := 0; x < 120; x++ {
			img.Set(x, y, color.RGBA{0, 255, 0, 255})
		}
	}

	f, err := os.Create(path)
	assert.NoError(t, err)
	defer func() {
		_ = f.Close()
	}()

	switch format {
	case "png":
		err = png.Encode(f, img)
	case "jpg", "jpeg":
		err = jpeg.Encode(f, img, &jpeg.Options{Quality: 90})
	case "gif":
		gifImg := &gif.GIF{
			Image: []*image.Paletted{
				image.NewPaletted(img.Bounds(), []color.Color{
					color.RGBA{0, 255, 0, 255},
				}),
			},
			Delay: []int{0},
		}
		err = gif.EncodeAll(f, gifImg)
	default:
		t.Fatalf("unsupported test format: %s", format)
	}
	assert.NoError(t, err)

	return path
}

func TestAddWatermark(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 200, 100))
	out, err := addWatermark(img, "TEST")
	assert.NoError(t, err)
	assert.NotNil(t, out)
}

func TestSaveImage(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "test.png")
	img := image.NewRGBA(image.Rect(0, 0, 50, 50))
	err := saveImage(img, path, "png")
	assert.NoError(t, err)

	_, err = os.Stat(path)
	assert.NoError(t, err)
}

func TestProcess_FullFlow_MultipleFormats(t *testing.T) {
	tmpDir := t.TempDir()
	inputDir := filepath.Join(tmpDir, "input")
	outputDir := filepath.Join(tmpDir, "output")
	_ = os.MkdirAll(inputDir, 0755)
	_ = os.MkdirAll(outputDir, 0755)

	cfg := &config.AppConfig{
		StoragePathConfig: config.StoragePathConfig{
			InputDir:  inputDir + string(os.PathSeparator),
			OutputDir: outputDir + string(os.PathSeparator),
		},
		ImageFormats: config.ImageFormats{
			SupportedFormats: map[string]bool{
				"png":  true,
				"jpg":  true,
				"jpeg": true,
				"gif":  true,
			},
		},
	}

	tests := []struct {
		name   string
		format string
	}{
		{"PNG", "png"},
		{"JPG", "jpg"},
		{"JPEG", "jpeg"},
		{"GIF", "gif"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			filename := "test_image." + tt.format
			inputFilePath := createTempImageByFormat(t, inputDir, filename, tt.format)

			assert.FileExists(t, inputFilePath)

			img := &domain.Image{
				Name:      filename,
				Format:    tt.format,
				Watermark: "WM",
				Resize:    &domain.Resize{Width: 60, Height: 60},
				Mini:      true,
			}

			err := Process(cfg, img)
			assert.NoError(t, err)

			outputPath := filepath.Join(outputDir, filename)
			_, err = os.Stat(outputPath)
			assert.NoError(t, err)
		})
	}
}
