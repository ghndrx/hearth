package services

import (
	"bytes"
	"context"
	"io"
	"mime/multipart"
	"net/textproto"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"hearth/internal/storage"
)

// mockStorageBackend implements storage.StorageBackend for testing
type mockStorageBackend struct {
	files map[string][]byte
}

func newMockStorageBackend() *mockStorageBackend {
	return &mockStorageBackend{
		files: make(map[string][]byte),
	}
}

func (m *mockStorageBackend) Upload(ctx context.Context, path string, file io.Reader, contentType string, size int64) (string, error) {
	data, _ := io.ReadAll(file)
	m.files[path] = data
	return "http://test.com/" + path, nil
}

func (m *mockStorageBackend) Download(ctx context.Context, path string) (io.ReadCloser, error) {
	if data, ok := m.files[path]; ok {
		return io.NopCloser(bytes.NewReader(data)), nil
	}
	return nil, io.EOF
}

func (m *mockStorageBackend) Delete(ctx context.Context, path string) error {
	delete(m.files, path)
	return nil
}

func (m *mockStorageBackend) GetURL(path string) string {
	return "http://test.com/" + path
}

func (m *mockStorageBackend) GetSignedURL(ctx context.Context, path string, expiry time.Duration) (string, error) {
	return "http://test.com/signed/" + path, nil
}

// createTestFileHeader creates a mock multipart.FileHeader for testing
func createTestFileHeader(filename, contentType string, content []byte) *multipart.FileHeader {
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	
	h := make(textproto.MIMEHeader)
	h.Set("Content-Disposition", `form-data; name="file"; filename="`+filename+`"`)
	h.Set("Content-Type", contentType)
	
	part, _ := writer.CreatePart(h)
	part.Write(content)
	writer.Close()
	
	reader := multipart.NewReader(&buf, writer.Boundary())
	form, _ := reader.ReadForm(32 << 20)
	
	if files := form.File["file"]; len(files) > 0 {
		return files[0]
	}
	
	// Fallback: create a simple FileHeader
	return &multipart.FileHeader{
		Filename: filename,
		Header:   textproto.MIMEHeader{"Content-Type": []string{contentType}},
		Size:     int64(len(content)),
	}
}

func TestAttachmentService_NewAttachmentService(t *testing.T) {
	t.Run("without storage", func(t *testing.T) {
		svc := NewAttachmentService(nil)
		assert.NotNil(t, svc)
		assert.NotNil(t, svc.attachments)
		assert.Nil(t, svc.storage)
	})

	t.Run("with storage", func(t *testing.T) {
		backend := newMockStorageBackend()
		storageSvc := storage.NewService(backend, 10, nil)
		svc := NewAttachmentService(storageSvc)
		assert.NotNil(t, svc)
		assert.NotNil(t, svc.storage)
	})
}

func TestAttachmentService_InMemory(t *testing.T) {
	svc := NewAttachmentService(nil)
	ctx := context.Background()

	assert.NotNil(t, svc)
	assert.NotNil(t, svc.attachments)

	// Test Get for non-existent attachment
	_, err := svc.Get(ctx, uuid.New())
	assert.Equal(t, ErrAttachmentNotFound, err)
}

func TestAttachmentService_Get(t *testing.T) {
	svc := NewAttachmentService(nil)
	ctx := context.Background()

	t.Run("not found", func(t *testing.T) {
		_, err := svc.Get(ctx, uuid.New())
		assert.Equal(t, ErrAttachmentNotFound, err)
	})

	t.Run("found", func(t *testing.T) {
		// Manually add an attachment
		id := uuid.New()
		svc.attachments[id] = &Attachment{
			ID:          id,
			Filename:    "test.txt",
			ContentType: "text/plain",
			Size:        100,
		}

		a, err := svc.Get(ctx, id)
		require.NoError(t, err)
		assert.Equal(t, id, a.ID)
		assert.Equal(t, "test.txt", a.Filename)
	})
}

func TestAttachmentService_GetByChannel(t *testing.T) {
	svc := NewAttachmentService(nil)
	ctx := context.Background()

	channelID := uuid.New()
	
	t.Run("no attachments", func(t *testing.T) {
		attachments, err := svc.GetByChannel(ctx, channelID)
		assert.NoError(t, err)
		assert.Empty(t, attachments)
	})

	t.Run("with attachments", func(t *testing.T) {
		// Add some attachments
		id1 := uuid.New()
		id2 := uuid.New()
		otherChannelID := uuid.New()

		svc.attachments[id1] = &Attachment{
			ID:        id1,
			ChannelID: channelID,
			Filename:  "file1.txt",
		}
		svc.attachments[id2] = &Attachment{
			ID:        id2,
			ChannelID: channelID,
			Filename:  "file2.txt",
		}
		svc.attachments[uuid.New()] = &Attachment{
			ID:        uuid.New(),
			ChannelID: otherChannelID,
			Filename:  "other.txt",
		}

		attachments, err := svc.GetByChannel(ctx, channelID)
		assert.NoError(t, err)
		assert.Len(t, attachments, 2)
	})
}

func TestAttachmentService_GetByMessage(t *testing.T) {
	svc := NewAttachmentService(nil)
	ctx := context.Background()

	messageID := uuid.New()
	
	t.Run("no attachments", func(t *testing.T) {
		attachments, err := svc.GetByMessage(ctx, messageID)
		assert.NoError(t, err)
		assert.Empty(t, attachments)
	})

	t.Run("with attachments", func(t *testing.T) {
		id1 := uuid.New()
		svc.attachments[id1] = &Attachment{
			ID:        id1,
			MessageID: messageID,
			Filename:  "attachment.pdf",
		}

		attachments, err := svc.GetByMessage(ctx, messageID)
		assert.NoError(t, err)
		assert.Len(t, attachments, 1)
		assert.Equal(t, "attachment.pdf", attachments[0].Filename)
	})
}

func TestAttachmentService_Delete(t *testing.T) {
	svc := NewAttachmentService(nil)
	ctx := context.Background()

	t.Run("not found", func(t *testing.T) {
		err := svc.Delete(ctx, uuid.New(), uuid.New())
		assert.Equal(t, ErrAttachmentNotFound, err)
	})

	t.Run("access denied", func(t *testing.T) {
		id := uuid.New()
		ownerID := uuid.New()
		otherUserID := uuid.New()

		svc.attachments[id] = &Attachment{
			ID:         id,
			UploaderID: ownerID,
			Filename:   "test.txt",
		}

		err := svc.Delete(ctx, id, otherUserID)
		assert.Equal(t, ErrAttachmentAccessDenied, err)
	})

	t.Run("success", func(t *testing.T) {
		id := uuid.New()
		ownerID := uuid.New()

		svc.attachments[id] = &Attachment{
			ID:         id,
			UploaderID: ownerID,
			Filename:   "test.txt",
		}

		err := svc.Delete(ctx, id, ownerID)
		assert.NoError(t, err)

		// Verify it's deleted
		_, err = svc.Get(ctx, id)
		assert.Equal(t, ErrAttachmentNotFound, err)
	})
}

func TestAttachmentService_DeleteByMessage(t *testing.T) {
	svc := NewAttachmentService(nil)
	ctx := context.Background()

	messageID := uuid.New()
	id1 := uuid.New()
	id2 := uuid.New()

	svc.attachments[id1] = &Attachment{
		ID:        id1,
		MessageID: messageID,
		Filename:  "file1.txt",
	}
	svc.attachments[id2] = &Attachment{
		ID:        id2,
		MessageID: messageID,
		Filename:  "file2.txt",
	}

	err := svc.DeleteByMessage(ctx, messageID)
	assert.NoError(t, err)

	// Verify both are deleted
	_, err = svc.Get(ctx, id1)
	assert.Equal(t, ErrAttachmentNotFound, err)
	_, err = svc.Get(ctx, id2)
	assert.Equal(t, ErrAttachmentNotFound, err)
}

func TestAttachmentService_GetSignedURL(t *testing.T) {
	svc := NewAttachmentService(nil)
	ctx := context.Background()

	t.Run("not found", func(t *testing.T) {
		_, err := svc.GetSignedURL(ctx, uuid.New(), time.Hour)
		assert.Equal(t, ErrAttachmentNotFound, err)
	})

	t.Run("success without storage", func(t *testing.T) {
		id := uuid.New()
		svc.attachments[id] = &Attachment{
			ID:       id,
			URL:      "/attachments/test.txt",
			Filename: "test.txt",
		}

		url, err := svc.GetSignedURL(ctx, id, time.Hour)
		assert.NoError(t, err)
		assert.Equal(t, "/attachments/test.txt", url)
	})
}

func TestAttachmentService_WithStorage(t *testing.T) {
	backend := newMockStorageBackend()
	storageSvc := storage.NewService(backend, 10, nil)
	svc := NewAttachmentService(storageSvc)
	ctx := context.Background()

	t.Run("get signed URL with storage", func(t *testing.T) {
		id := uuid.New()
		svc.attachments[id] = &Attachment{
			ID:       id,
			Path:     "attachments/test.txt",
			Filename: "test.txt",
		}

		url, err := svc.GetSignedURL(ctx, id, time.Hour)
		assert.NoError(t, err)
		assert.Contains(t, url, "signed")
	})
}

func TestValidateContentType(t *testing.T) {
	tests := []struct {
		contentType string
		expected    bool
	}{
		// Allowed types
		{"image/png", true},
		{"image/jpeg", true},
		{"image/gif", true},
		{"image/webp", true},
		{"application/pdf", true},
		{"text/plain", true},
		{"text/html", true},
		{"application/json", true},
		{"audio/mpeg", true},
		{"video/mp4", true},
		
		// Blocked types
		{"application/x-msdownload", false},
		{"application/x-msdos-program", false},
		{"application/x-executable", false},
		{"application/x-dosexec", false},
	}

	for _, tt := range tests {
		t.Run(tt.contentType, func(t *testing.T) {
			assert.Equal(t, tt.expected, ValidateContentType(tt.contentType))
		})
	}
}

func TestValidateFileExtension(t *testing.T) {
	tests := []struct {
		filename string
		expected bool
	}{
		// Allowed extensions
		{"document.pdf", true},
		{"image.png", true},
		{"image.jpg", true},
		{"image.jpeg", true},
		{"data.json", true},
		{"text.txt", true},
		{"archive.zip", true},
		{"video.mp4", true},
		{"audio.mp3", true},
		
		// Blocked extensions
		{"virus.exe", false},
		{"script.bat", false},
		{"command.cmd", false},
		{"legacy.com", false},
		{"malware.msi", false},
		{"screensaver.scr", false},
		{"pif.pif", false},
		{"vbscript.vbs", false},
		{"javascript.js", false},
		{"java.jar", false},
		{"powershell.ps1", false},
		{"shell.sh", false},
		{"bash.bash", false},
		
		// Edge cases
		{"VIRUS.EXE", false},     // uppercase
		{"file.Exe", false},      // mixed case
		{"no_extension", true},   // no extension
		{".hidden", true},        // hidden file (no real extension)
	}

	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			assert.Equal(t, tt.expected, ValidateFileExtension(tt.filename))
		})
	}
}

func TestAttachmentService_Download(t *testing.T) {
	ctx := context.Background()

	t.Run("without storage", func(t *testing.T) {
		svc := NewAttachmentService(nil)
		
		id := uuid.New()
		svc.attachments[id] = &Attachment{
			ID:       id,
			Filename: "test.txt",
			Path:     "attachments/test.txt",
		}

		_, _, err := svc.Download(ctx, id)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "storage not configured")
	})

	t.Run("attachment not found", func(t *testing.T) {
		svc := NewAttachmentService(nil)
		
		_, _, err := svc.Download(ctx, uuid.New())
		assert.Equal(t, ErrAttachmentNotFound, err)
	})
}

func TestAttachmentErrors(t *testing.T) {
	// Test error string representations
	assert.Equal(t, "attachment not found", ErrAttachmentNotFound.Error())
	assert.Equal(t, "file too large", ErrFileTooLarge.Error())
	assert.Equal(t, "file type not allowed", ErrFileTypeNotAllowed.Error())
	assert.Equal(t, "access denied", ErrAttachmentAccessDenied.Error())
}
