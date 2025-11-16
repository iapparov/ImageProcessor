package imgprocessor

import (
	"github.com/disintegration/imaging"
	wbzlog "github.com/wb-go/wbf/zlog"
	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/math/fixed"
	"image"
	"image/color"
	"image/draw"
	"image/gif"
	"image/jpeg"
	"image/png"
	"imageProcessor/internal/config"
	"imageProcessor/internal/domain"
	"os"
	"strings"
)

func Process(cfg *config.AppConfig, img *domain.Image) error {

	inputPath := cfg.StoragePathConfig.InputDir + img.Name
	outputPath := cfg.StoragePathConfig.OutputDir + img.Name
	outpudDir := cfg.StoragePathConfig.OutputDir

	if err := os.MkdirAll(outpudDir, 0755); err != nil {
		wbzlog.Logger.Error().Err(err).Msg("Failed to create input directory")
		return err
	}

	src, err := imaging.Open(inputPath)
	if err != nil {
		wbzlog.Logger.Error().Err(err).Msg("Failed to open source image")
		return err
	}

	result := src

	if img.Watermark != "" {
		result, err = addWatermark(result, img.Watermark)
		if err != nil {
			wbzlog.Logger.Error().Err(err).Msg("Failed to add watermark")
			return err
		}
	}

	if (img.Resize.Height > 0 && img.Resize.Width >= 0) || (img.Resize.Width > 0 && img.Resize.Height >= 0) {
		result = imaging.Resize(result, img.Resize.Width, img.Resize.Height, imaging.Lanczos)
	}

	if img.Mini {
		result = imaging.Thumbnail(result, 300, 300, imaging.Lanczos)
	}

	err = saveImage(result, outputPath, img.Format)
	if err != nil {
		wbzlog.Logger.Error().Err(err).Msg("Failed to save processed image")
		return err
	}

	return nil
}

func addWatermark(img image.Image, text string) (image.Image, error) {
	bounds := img.Bounds()
	rgba := image.NewRGBA(bounds)

	// Копируем исходное изображение
	draw.Draw(rgba, bounds, img, bounds.Min, draw.Src)

	// Создаём цвет текста с прозрачностью 20%
	alpha := uint8(51) // 255 * 0.2 = 51
	col := image.NewUniform(color.RGBA{255, 255, 255, alpha})

	face := basicfont.Face7x13
	stepX := 150 // расстояние между водяными знаками по X
	stepY := 50  // расстояние по Y

	for y := 0; y < bounds.Dy(); y += stepY {
		for x := 0; x < bounds.Dx(); x += stepX {
			point := fixed.Point26_6{
				X: fixed.I(x),
				Y: fixed.I(y + face.Metrics().Ascent.Ceil()),
			}
			drawer := &font.Drawer{
				Dst:  rgba,
				Src:  col,
				Face: face,
				Dot:  point,
			}
			drawer.DrawString(text)
		}
	}

	return rgba, nil
}

func saveImage(img image.Image, path string, format string) error {
	out, err := os.Create(path)
	if err != nil {
		return err
	}
	defer func() {
		err := out.Close()
		if err != nil {
			wbzlog.Logger.Error().Err(err).Msg("Failed to close file")
		}
	}()

	format = strings.ToLower(strings.TrimPrefix(format, "."))

	switch format {
	case "jpg", "jpeg":
		return jpeg.Encode(out, img, &jpeg.Options{Quality: 90})
	case "png":
		return png.Encode(out, img)
	case "gif":
		return gif.Encode(out, img, nil)
	default:
		return imaging.Save(img, path)
	}
}
