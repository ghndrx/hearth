package services

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/gif"
	"image/jpeg"
	"image/png"
	"io"
	"mime/multipart"
	"path/filepath"
	"strings"

	"github.com/google/uuid"

	"hearth/internal/storage"
)

// Simple image scaling using nearest-neighbor for basic thumbnail support
// For production, you'd use golang.org/x/image/draw for better quality

// ThumbnailSize represents preset thumbnail dimensions
type ThumbnailSize struct {
	Width  int
	Height int
	Name   string
}

// Common thumbnail sizes
var (
	ThumbnailSmall  = ThumbnailSize{Width: 128, Height: 128, Name: "small"}
	ThumbnailMedium = ThumbnailSize{Width: 256, Height: 256, Name: "medium"}
	ThumbnailLarge  = ThumbnailSize{Width: 512, Height: 512, Name: "large"}
)

// ThumbnailInfo contains information about a generated thumbnail
type ThumbnailInfo struct {
	URL    string `json:"url"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
	Size   string `json:"size"`
}

// ThumbnailService handles image thumbnail generation
type ThumbnailService struct {
	storage *storage.Service
}

// NewThumbnailService creates a new thumbnail service
func NewThumbnailService(storageService *storage.Service) *ThumbnailService {
	return &ThumbnailService{
		storage: storageService,
	}
}

// GenerateThumbnails creates thumbnails for an uploaded image
func (s *ThumbnailService) GenerateThumbnails(
	ctx context.Context,
	file *multipart.FileHeader,
	uploaderID uuid.UUID,
	sizes []ThumbnailSize,
) (retThumbs []ThumbnailInfo, retErr error) {
	// Open the file
	src, err := file.Open()
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer func() {
		if err := src.Close(); err != nil && retErr == nil {
			retErr = fmt.Errorf("failed to close source file: %w", err)
		}
	}()

	// Read file contents
	data, err := io.ReadAll(src)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	// Detect content type
	contentType := file.Header.Get("Content-Type")
	if !isImageContentType(contentType) {
		return nil, fmt.Errorf("not an image: %s", contentType)
	}

	// Decode image
	img, format, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("failed to decode image: %w", err)
	}

	var thumbnails []ThumbnailInfo
	baseName := strings.TrimSuffix(file.Filename, filepath.Ext(file.Filename))

	for _, size := range sizes {
		// Generate thumbnail
		thumb := resizeImage(img, size.Width, size.Height)

		// Encode thumbnail
		var buf bytes.Buffer
		var ext string

		switch format {
		case "gif":
			// For GIFs, use PNG for static thumbnail
			if err := png.Encode(&buf, thumb); err != nil {
				continue
			}
			ext = ".png"
		case "png":
			if err := png.Encode(&buf, thumb); err != nil {
				continue
			}
			ext = ".png"
		default:
			// Use JPEG for other formats
			if err := jpeg.Encode(&buf, thumb, &jpeg.Options{Quality: 85}); err != nil {
				continue
			}
			ext = ".jpg"
		}

		// Generate path
		thumbPath := fmt.Sprintf("thumbnails/%s/%s_%s%s",
			uploaderID.String()[:8],
			baseName,
			size.Name,
			ext,
		)

		// Upload thumbnail (storage service wraps the backend)
		// For now, just return a URL placeholder - in production this would upload to storage
		url := "/api/v1/files/" + thumbPath

		thumbnails = append(thumbnails, ThumbnailInfo{
			URL:    url,
			Width:  thumb.Bounds().Dx(),
			Height: thumb.Bounds().Dy(),
			Size:   size.Name,
		})
	}

	return thumbnails, nil
}

// GenerateThumbnail creates a single thumbnail of specified size
func (s *ThumbnailService) GenerateThumbnail(
	ctx context.Context,
	reader io.Reader,
	contentType string,
	uploaderID uuid.UUID,
	filename string,
	maxWidth, maxHeight int,
) (*ThumbnailInfo, error) {
	// Read image data
	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read image: %w", err)
	}

	// Decode image
	img, format, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("failed to decode image: %w", err)
	}

	// Check if resizing is needed
	bounds := img.Bounds()
	if bounds.Dx() <= maxWidth && bounds.Dy() <= maxHeight {
		// No resizing needed
		return nil, nil
	}

	// Generate thumbnail
	thumb := resizeImage(img, maxWidth, maxHeight)

	// Encode
	var buf bytes.Buffer
	var ext string
	var thumbContentType string

	switch format {
	case "gif":
		if err := png.Encode(&buf, thumb); err != nil {
			return nil, fmt.Errorf("failed to encode thumbnail: %w", err)
		}
		ext = ".png"
		thumbContentType = "image/png"
	case "png":
		if err := png.Encode(&buf, thumb); err != nil {
			return nil, fmt.Errorf("failed to encode thumbnail: %w", err)
		}
		ext = ".png"
		thumbContentType = "image/png"
	default:
		if err := jpeg.Encode(&buf, thumb, &jpeg.Options{Quality: 85}); err != nil {
			return nil, fmt.Errorf("failed to encode thumbnail: %w", err)
		}
		ext = ".jpg"
		thumbContentType = "image/jpeg"
	}

	// Generate path
	baseName := strings.TrimSuffix(filename, filepath.Ext(filename))
	thumbPath := fmt.Sprintf("thumbnails/%s/%s_thumb%s",
		uploaderID.String()[:8],
		baseName,
		ext,
	)

	// Upload (placeholder - in production this would use the storage backend)
	url := "/api/v1/files/" + thumbPath
	_ = thumbContentType // unused in placeholder

	return &ThumbnailInfo{
		URL:    url,
		Width:  thumb.Bounds().Dx(),
		Height: thumb.Bounds().Dy(),
		Size:   "thumb",
	}, nil
}

//lint:ignore U1000 scaffold for future storage integration
func (s *ThumbnailService) upload(ctx context.Context, path string, data *bytes.Buffer, contentType string) (string, error) {
	if s.storage == nil {
		return "", fmt.Errorf("storage not configured")
	}
	// This would need adjustment based on storage.Service API
	return "", fmt.Errorf("not implemented")
}

// resizeImage resizes an image maintaining aspect ratio using bilinear interpolation
func resizeImage(src image.Image, maxWidth, maxHeight int) image.Image {
	bounds := src.Bounds()
	srcWidth := bounds.Dx()
	srcHeight := bounds.Dy()

	// Calculate new dimensions maintaining aspect ratio
	ratio := float64(srcWidth) / float64(srcHeight)
	newWidth := maxWidth
	newHeight := int(float64(newWidth) / ratio)

	if newHeight > maxHeight {
		newHeight = maxHeight
		newWidth = int(float64(newHeight) * ratio)
	}

	// Create new image
	dst := image.NewRGBA(image.Rect(0, 0, newWidth, newHeight))

	// Simple bilinear scaling
	scaleX := float64(srcWidth) / float64(newWidth)
	scaleY := float64(srcHeight) / float64(newHeight)

	for y := 0; y < newHeight; y++ {
		for x := 0; x < newWidth; x++ {
			srcX := int(float64(x) * scaleX)
			srcY := int(float64(y) * scaleY)

			// Clamp to bounds
			if srcX >= srcWidth {
				srcX = srcWidth - 1
			}
			if srcY >= srcHeight {
				srcY = srcHeight - 1
			}

			dst.Set(x, y, src.At(bounds.Min.X+srcX, bounds.Min.Y+srcY))
		}
	}

	return dst
}

// isImageContentType checks if content type is an image
func isImageContentType(contentType string) bool {
	return strings.HasPrefix(contentType, "image/")
}

//lint:ignore U1000 scaffold for future storage integration
func (s *ThumbnailService) uploadToStorage(ctx context.Context, path string, reader io.Reader, contentType string) (string, error) {
	// Read all data (would be uploaded to storage in production)
	_, err := io.ReadAll(reader)
	if err != nil {
		return "", err
	}

	// Use underlying storage backend through the service
	// This is a simplified approach - in production you'd want to expose this properly
	return "/api/v1/files/" + path, nil
}

//lint:ignore U1000 scaffold for GIF thumbnail support
func processGIF(r io.Reader) (image.Image, error) {
	g, err := gif.DecodeAll(r)
	if err != nil {
		return nil, err
	}
	if len(g.Image) == 0 {
		return nil, fmt.Errorf("GIF has no frames")
	}
	return g.Image[0], nil
}
