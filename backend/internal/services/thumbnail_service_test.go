package services

import (
	"bytes"
	"context"
	"errors"
	"image"
	"image/color"
	"image/gif"
	"image/jpeg"
	"image/png"
	"io"
	"mime/multipart"
	"net/textproto"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"hearth/internal/storage"
)

// thumbTestImage creates a valid encoded image of the given format and dimensions.
func thumbTestImage(t *testing.T, format string, width, height int) []byte {
	t.Helper()

	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, color.RGBA{
				R: uint8((x * 255) / max(width, 1)),
				G: uint8((y * 255) / max(height, 1)),
				B: 128, A: 255,
			})
		}
	}

	var buf bytes.Buffer
	switch format {
	case "jpeg":
		require.NoError(t, jpeg.Encode(&buf, img, &jpeg.Options{Quality: 90}))
	case "png":
		require.NoError(t, png.Encode(&buf, img))
	case "gif":
		require.NoError(t, gif.Encode(&buf, img, nil))
	default:
		t.Fatalf("unsupported test image format: %s", format)
	}
	return buf.Bytes()
}

// thumbFileHeader builds a multipart.FileHeader backed by the given data.
func thumbFileHeader(t *testing.T, filename, contentType string, data []byte) *multipart.FileHeader {
	t.Helper()

	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	h := make(textproto.MIMEHeader)
	h.Set("Content-Disposition", `form-data; name="file"; filename="`+filename+`"`)
	h.Set("Content-Type", contentType)

	part, err := writer.CreatePart(h)
	require.NoError(t, err)
	_, err = part.Write(data)
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	reader := multipart.NewReader(&buf, writer.Boundary())
	form, err := reader.ReadForm(32 << 20)
	require.NoError(t, err)

	files := form.File["file"]
	require.NotEmpty(t, files, "multipart form should contain file")
	return files[0]
}

// errReader is an io.Reader that always returns an error.
type errReader struct {
	err error
}

func (r *errReader) Read([]byte) (int, error) {
	return 0, r.err
}

// Ensure errReader satisfies io.Reader at compile time.
var _ io.Reader = (*errReader)(nil)

// --- NewThumbnailService ---

func TestNewThumbnailService(t *testing.T) {
	t.Run("nil storage", func(t *testing.T) {
		svc := NewThumbnailService(nil)
		require.NotNil(t, svc)
		assert.Nil(t, svc.storage)
	})

	t.Run("with storage", func(t *testing.T) {
		backend := newMockStorageBackend()
		storageSvc := storage.NewService(backend, 10, nil)
		svc := NewThumbnailService(storageSvc)
		require.NotNil(t, svc)
		assert.NotNil(t, svc.storage)
	})
}

// --- GenerateThumbnails: successful image formats ---

func TestGenerateThumbnails_ImageFormats(t *testing.T) {
	svc := NewThumbnailService(nil)
	uploaderID := uuid.New()
	sizes := []ThumbnailSize{ThumbnailSmall}

	tests := []struct {
		name        string
		filename    string
		contentType string
		format      string
		wantURLExt  string
	}{
		{"JPEG", "photo.jpg", "image/jpeg", "jpeg", ".jpg"},
		{"PNG", "icon.png", "image/png", "png", ".png"},
		{"GIF produces PNG thumbnail", "anim.gif", "image/gif", "gif", ".png"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			imgData := thumbTestImage(t, tt.format, 800, 600)
			fh := thumbFileHeader(t, tt.filename, tt.contentType, imgData)

			thumbs, err := svc.GenerateThumbnails(context.Background(), fh, uploaderID, sizes)
			require.NoError(t, err)
			require.Len(t, thumbs, 1)

			assert.Equal(t, "small", thumbs[0].Size)
			assert.Contains(t, thumbs[0].URL, tt.wantURLExt)
			assert.Greater(t, thumbs[0].Width, 0)
			assert.Greater(t, thumbs[0].Height, 0)
			assert.LessOrEqual(t, thumbs[0].Width, ThumbnailSmall.Width)
			assert.LessOrEqual(t, thumbs[0].Height, ThumbnailSmall.Height)
		})
	}
}

// --- GenerateThumbnails: multiple sizes ---

func TestGenerateThumbnails_MultipleSizes(t *testing.T) {
	svc := NewThumbnailService(nil)
	uploaderID := uuid.New()
	sizes := []ThumbnailSize{ThumbnailSmall, ThumbnailMedium, ThumbnailLarge}

	imgData := thumbTestImage(t, "jpeg", 1024, 768)
	fh := thumbFileHeader(t, "big.jpg", "image/jpeg", imgData)

	thumbs, err := svc.GenerateThumbnails(context.Background(), fh, uploaderID, sizes)
	require.NoError(t, err)
	require.Len(t, thumbs, 3)

	expectedSizes := map[string]ThumbnailSize{
		"small":  ThumbnailSmall,
		"medium": ThumbnailMedium,
		"large":  ThumbnailLarge,
	}

	for _, thumb := range thumbs {
		expected, ok := expectedSizes[thumb.Size]
		require.True(t, ok, "unexpected size: %s", thumb.Size)
		assert.LessOrEqual(t, thumb.Width, expected.Width)
		assert.LessOrEqual(t, thumb.Height, expected.Height)
		assert.Greater(t, thumb.Width, 0)
		assert.Greater(t, thumb.Height, 0)
	}
}

// --- GenerateThumbnails: aspect ratio preservation ---

func TestGenerateThumbnails_AspectRatio(t *testing.T) {
	svc := NewThumbnailService(nil)
	uploaderID := uuid.New()

	tests := []struct {
		name string
		w, h int
	}{
		{"landscape", 800, 400},
		{"portrait", 400, 800},
		{"square", 600, 600},
		{"wide panoramic", 2000, 200},
		{"tall narrow", 200, 2000},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			imgData := thumbTestImage(t, "png", tt.w, tt.h)
			fh := thumbFileHeader(t, "test.png", "image/png", imgData)

			thumbs, err := svc.GenerateThumbnails(context.Background(), fh, uploaderID, []ThumbnailSize{ThumbnailSmall})
			require.NoError(t, err)
			require.Len(t, thumbs, 1)

			thumb := thumbs[0]
			assert.LessOrEqual(t, thumb.Width, ThumbnailSmall.Width)
			assert.LessOrEqual(t, thumb.Height, ThumbnailSmall.Height)
			assert.True(t,
				thumb.Width == ThumbnailSmall.Width || thumb.Height == ThumbnailSmall.Height,
				"one dimension should reach max for %s (got %dx%d)", tt.name, thumb.Width, thumb.Height)
		})
	}
}

// --- GenerateThumbnails: error cases ---

func TestGenerateThumbnails_UnsupportedContentType(t *testing.T) {
	svc := NewThumbnailService(nil)
	uploaderID := uuid.New()

	tests := []struct {
		name        string
		contentType string
	}{
		{"text/plain", "text/plain"},
		{"application/pdf", "application/pdf"},
		{"video/mp4", "video/mp4"},
		{"empty", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fh := thumbFileHeader(t, "file.bin", tt.contentType, []byte("not an image"))

			thumbs, err := svc.GenerateThumbnails(context.Background(), fh, uploaderID, []ThumbnailSize{ThumbnailSmall})
			assert.Error(t, err)
			assert.Nil(t, thumbs)
			assert.Contains(t, err.Error(), "not an image")
		})
	}
}

func TestGenerateThumbnails_CorruptedImage(t *testing.T) {
	svc := NewThumbnailService(nil)
	uploaderID := uuid.New()

	fh := thumbFileHeader(t, "corrupt.jpg", "image/jpeg", []byte("not valid jpeg data"))

	thumbs, err := svc.GenerateThumbnails(context.Background(), fh, uploaderID, []ThumbnailSize{ThumbnailSmall})
	assert.Error(t, err)
	assert.Nil(t, thumbs)
	assert.Contains(t, err.Error(), "failed to decode image")
}

func TestGenerateThumbnails_EmptyFile(t *testing.T) {
	svc := NewThumbnailService(nil)
	uploaderID := uuid.New()

	fh := thumbFileHeader(t, "empty.png", "image/png", []byte{})

	thumbs, err := svc.GenerateThumbnails(context.Background(), fh, uploaderID, []ThumbnailSize{ThumbnailSmall})
	assert.Error(t, err)
	assert.Nil(t, thumbs)
}

func TestGenerateThumbnails_EmptySizes(t *testing.T) {
	svc := NewThumbnailService(nil)
	uploaderID := uuid.New()

	imgData := thumbTestImage(t, "jpeg", 100, 100)
	fh := thumbFileHeader(t, "test.jpg", "image/jpeg", imgData)

	thumbs, err := svc.GenerateThumbnails(context.Background(), fh, uploaderID, nil)
	require.NoError(t, err)
	assert.Empty(t, thumbs)
}

func TestGenerateThumbnails_URLFormat(t *testing.T) {
	svc := NewThumbnailService(nil)
	uploaderID := uuid.New()

	imgData := thumbTestImage(t, "jpeg", 200, 200)
	fh := thumbFileHeader(t, "myimage.jpg", "image/jpeg", imgData)

	thumbs, err := svc.GenerateThumbnails(context.Background(), fh, uploaderID, []ThumbnailSize{ThumbnailSmall})
	require.NoError(t, err)
	require.Len(t, thumbs, 1)

	assert.Contains(t, thumbs[0].URL, "/api/v1/files/thumbnails/")
	assert.Contains(t, thumbs[0].URL, uploaderID.String()[:8])
	assert.Contains(t, thumbs[0].URL, "myimage_small")
}

// --- GenerateThumbnails: defer close error propagation (security fix a9a38c0) ---

func TestGenerateThumbnails_DeferCloseErrorPropagation(t *testing.T) {
	svc := NewThumbnailService(nil)
	uploaderID := uuid.New()

	t.Run("success path does not produce close error", func(t *testing.T) {
		imgData := thumbTestImage(t, "jpeg", 200, 200)
		fh := thumbFileHeader(t, "test.jpg", "image/jpeg", imgData)

		thumbs, err := svc.GenerateThumbnails(context.Background(), fh, uploaderID, []ThumbnailSize{ThumbnailSmall})
		require.NoError(t, err, "close error should not appear on successful path")
		require.Len(t, thumbs, 1)
	})

	t.Run("primary error preserved when retErr already set", func(t *testing.T) {
		corruptFh := thumbFileHeader(t, "bad.jpg", "image/jpeg", []byte("not valid"))
		_, err := svc.GenerateThumbnails(context.Background(), corruptFh, uploaderID, []ThumbnailSize{ThumbnailSmall})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to decode image",
			"original error preserved; close error does not overwrite retErr")
	})
}

// --- GenerateThumbnail (single thumbnail) ---

func TestGenerateThumbnail_Success(t *testing.T) {
	svc := NewThumbnailService(nil)
	uploaderID := uuid.New()

	tests := []struct {
		name    string
		format  string
		wantExt string
	}{
		{"JPEG encodes as jpg", "jpeg", ".jpg"},
		{"PNG encodes as png", "png", ".png"},
		{"GIF encodes as png", "gif", ".png"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			imgData := thumbTestImage(t, tt.format, 800, 600)

			info, err := svc.GenerateThumbnail(
				context.Background(),
				bytes.NewReader(imgData),
				"image/"+tt.format,
				uploaderID,
				"photo."+tt.format,
				256, 256,
			)
			require.NoError(t, err)
			require.NotNil(t, info)

			assert.Equal(t, "thumb", info.Size)
			assert.LessOrEqual(t, info.Width, 256)
			assert.LessOrEqual(t, info.Height, 256)
			assert.Contains(t, info.URL, tt.wantExt)
			assert.Contains(t, info.URL, "photo_thumb")
		})
	}
}

func TestGenerateThumbnail_NoResizeNeeded(t *testing.T) {
	svc := NewThumbnailService(nil)
	uploaderID := uuid.New()

	imgData := thumbTestImage(t, "jpeg", 100, 100)

	info, err := svc.GenerateThumbnail(
		context.Background(),
		bytes.NewReader(imgData),
		"image/jpeg",
		uploaderID,
		"small.jpg",
		256, 256,
	)
	require.NoError(t, err)
	assert.Nil(t, info, "returns nil when image fits within max dimensions")
}

func TestGenerateThumbnail_CorruptedData(t *testing.T) {
	svc := NewThumbnailService(nil)
	uploaderID := uuid.New()

	info, err := svc.GenerateThumbnail(
		context.Background(),
		bytes.NewReader([]byte("corrupt image data")),
		"image/jpeg",
		uploaderID,
		"bad.jpg",
		256, 256,
	)
	assert.Error(t, err)
	assert.Nil(t, info)
	assert.Contains(t, err.Error(), "failed to decode image")
}

func TestGenerateThumbnail_ReadError(t *testing.T) {
	svc := NewThumbnailService(nil)
	uploaderID := uuid.New()

	info, err := svc.GenerateThumbnail(
		context.Background(),
		&errReader{err: errors.New("disk read error")},
		"image/jpeg",
		uploaderID,
		"fail.jpg",
		256, 256,
	)
	assert.Error(t, err)
	assert.Nil(t, info)
	assert.Contains(t, err.Error(), "failed to read image")
}

// --- resizeImage ---

func TestResizeImage(t *testing.T) {
	tests := []struct {
		name             string
		srcW, srcH       int
		maxW, maxH       int
		expectW, expectH int
	}{
		{"landscape downscale", 800, 600, 128, 128, 128, 96},
		{"portrait downscale", 600, 800, 128, 128, 96, 128},
		{"square downscale", 500, 500, 128, 128, 128, 128},
		{"wide panoramic", 2000, 200, 256, 256, 256, 25},
		{"tall narrow", 200, 2000, 256, 256, 25, 256},
		{"upscale small", 64, 48, 128, 128, 128, 96},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			src := image.NewRGBA(image.Rect(0, 0, tt.srcW, tt.srcH))
			result := resizeImage(src, tt.maxW, tt.maxH)
			bounds := result.Bounds()
			assert.Equal(t, tt.expectW, bounds.Dx(), "width")
			assert.Equal(t, tt.expectH, bounds.Dy(), "height")
		})
	}
}

func TestResizeImage_PreservesColor(t *testing.T) {
	src := image.NewRGBA(image.Rect(0, 0, 100, 100))
	red := color.RGBA{R: 255, G: 0, B: 0, A: 255}
	for y := 0; y < 100; y++ {
		for x := 0; x < 100; x++ {
			src.Set(x, y, red)
		}
	}

	result := resizeImage(src, 50, 50)
	r, g, b, a := result.At(25, 25).RGBA()
	assert.Equal(t, uint32(0xffff), r)
	assert.Equal(t, uint32(0), g)
	assert.Equal(t, uint32(0), b)
	assert.Equal(t, uint32(0xffff), a)
}

func TestResizeImage_NonZeroOrigin(t *testing.T) {
	src := image.NewRGBA(image.Rect(10, 10, 110, 110))
	green := color.RGBA{R: 0, G: 255, B: 0, A: 255}
	for y := 10; y < 110; y++ {
		for x := 10; x < 110; x++ {
			src.Set(x, y, green)
		}
	}

	result := resizeImage(src, 50, 50)
	assert.Equal(t, 50, result.Bounds().Dx())
	assert.Equal(t, 50, result.Bounds().Dy())

	_, g, _, _ := result.At(25, 25).RGBA()
	assert.Equal(t, uint32(0xffff), g)
}

// --- isImageContentType ---

func TestIsImageContentType(t *testing.T) {
	tests := []struct {
		contentType string
		want        bool
	}{
		{"image/jpeg", true},
		{"image/png", true},
		{"image/gif", true},
		{"image/webp", true},
		{"image/svg+xml", true},
		{"text/plain", false},
		{"application/json", false},
		{"video/mp4", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.contentType, func(t *testing.T) {
			assert.Equal(t, tt.want, isImageContentType(tt.contentType))
		})
	}
}

// --- processGIF ---

func TestProcessGIF(t *testing.T) {
	t.Run("valid GIF", func(t *testing.T) {
		imgData := thumbTestImage(t, "gif", 100, 100)
		img, err := processGIF(bytes.NewReader(imgData))
		require.NoError(t, err)
		require.NotNil(t, img)
		assert.Greater(t, img.Bounds().Dx(), 0)
	})

	t.Run("invalid data", func(t *testing.T) {
		img, err := processGIF(bytes.NewReader([]byte("not a gif")))
		assert.Error(t, err)
		assert.Nil(t, img)
	})
}

// --- upload / uploadToStorage scaffolds ---

func TestUpload_NilStorage(t *testing.T) {
	svc := NewThumbnailService(nil)
	url, err := svc.upload(context.Background(), "test/path", bytes.NewBufferString("data"), "image/png")
	assert.Error(t, err)
	assert.Empty(t, url)
	assert.Contains(t, err.Error(), "storage not configured")
}

func TestUpload_WithStorage(t *testing.T) {
	backend := newMockStorageBackend()
	storageSvc := storage.NewService(backend, 10, nil)
	svc := NewThumbnailService(storageSvc)

	url, err := svc.upload(context.Background(), "test/path", bytes.NewBufferString("data"), "image/png")
	assert.Error(t, err)
	assert.Empty(t, url)
	assert.Contains(t, err.Error(), "not implemented")
}

func TestUploadToStorage_Success(t *testing.T) {
	svc := NewThumbnailService(nil)
	url, err := svc.uploadToStorage(context.Background(), "some/path.png", bytes.NewReader([]byte("data")), "image/png")
	require.NoError(t, err)
	assert.Equal(t, "/api/v1/files/some/path.png", url)
}

func TestUploadToStorage_ReadError(t *testing.T) {
	svc := NewThumbnailService(nil)
	url, err := svc.uploadToStorage(context.Background(), "path.png", &errReader{err: errors.New("io fail")}, "image/png")
	assert.Error(t, err)
	assert.Empty(t, url)
}

// --- ThumbnailSize constants ---

func TestThumbnailSizeConstants(t *testing.T) {
	assert.Equal(t, 128, ThumbnailSmall.Width)
	assert.Equal(t, 128, ThumbnailSmall.Height)
	assert.Equal(t, "small", ThumbnailSmall.Name)

	assert.Equal(t, 256, ThumbnailMedium.Width)
	assert.Equal(t, 256, ThumbnailMedium.Height)
	assert.Equal(t, "medium", ThumbnailMedium.Name)

	assert.Equal(t, 512, ThumbnailLarge.Width)
	assert.Equal(t, 512, ThumbnailLarge.Height)
	assert.Equal(t, "large", ThumbnailLarge.Name)
}
