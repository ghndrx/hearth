package handlers

import (
	"bytes"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"hearth/internal/services"
)

// setupAttachmentTestApp creates a test app with attachment routes
func setupAttachmentTestApp(attachmentSvc *services.AttachmentService) (*fiber.App, uuid.UUID) {
	app := fiber.New()
	userID := uuid.New()

	// Middleware to set userID
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("userID", userID)
		return c.Next()
	})

	handler := NewAttachmentHandler(attachmentSvc, nil)

	// Channel attachment routes
	app.Post("/channels/:id/attachments", handler.Upload)
	app.Get("/channels/:id/attachments", handler.GetChannelAttachments)

	// Attachment routes
	app.Get("/attachments/:id", handler.Get)
	app.Get("/attachments/:id/download", handler.Download)
	app.Get("/attachments/:id/signed-url", handler.GetSignedURL)
	app.Delete("/attachments/:id", handler.Delete)

	return app, userID
}

// createMultipartRequest creates a multipart form request with a file
func createMultipartRequest(url, filename, contentType string, content []byte) (*http.Request, error) {
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	part, err := writer.CreateFormFile("file", filename)
	if err != nil {
		return nil, err
	}
	part.Write(content)
	writer.Close()

	req := httptest.NewRequest(http.MethodPost, url, &buf)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	return req, nil
}

func TestAttachmentHandler_Upload(t *testing.T) {
	svc := services.NewAttachmentService(nil)
	app, _ := setupAttachmentTestApp(svc)

	t.Run("success", func(t *testing.T) {
		channelID := uuid.New()
		content := []byte("test file content")

		req, err := createMultipartRequest(
			"/channels/"+channelID.String()+"/attachments",
			"test.txt",
			"text/plain",
			content,
		)
		require.NoError(t, err)

		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusCreated, resp.StatusCode)

		var attachment services.Attachment
		json.NewDecoder(resp.Body).Decode(&attachment)
		assert.Equal(t, "test.txt", attachment.Filename)
		assert.Equal(t, channelID, attachment.ChannelID)
	})

	t.Run("invalid channel id", func(t *testing.T) {
		req, _ := createMultipartRequest(
			"/channels/invalid/attachments",
			"test.txt",
			"text/plain",
			[]byte("content"),
		)

		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)

		var result map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&result)
		assert.Equal(t, "invalid channel id", result["error"])
	})

	t.Run("no file provided", func(t *testing.T) {
		channelID := uuid.New()
		req := httptest.NewRequest(
			http.MethodPost,
			"/channels/"+channelID.String()+"/attachments",
			nil,
		)
		req.Header.Set("Content-Type", "multipart/form-data")

		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
	})

	t.Run("blocked file extension", func(t *testing.T) {
		channelID := uuid.New()
		req, _ := createMultipartRequest(
			"/channels/"+channelID.String()+"/attachments",
			"virus.exe",
			"application/x-msdownload",
			[]byte("malicious content"),
		)

		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)

		var result map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&result)
		assert.Equal(t, "file type not allowed", result["error"])
	})
}

func TestAttachmentHandler_Get(t *testing.T) {
	svc := services.NewAttachmentService(nil)
	app, userID := setupAttachmentTestApp(svc)

	t.Run("success", func(t *testing.T) {
		// Create an attachment
		channelID := uuid.New()
		id := uuid.New()
		
		// Manually add attachment
		svc.Upload_Test_Add(id, &services.Attachment{
			ID:          id,
			ChannelID:   channelID,
			UploaderID:  userID,
			Filename:    "test.txt",
			ContentType: "text/plain",
			Size:        100,
			URL:         "/attachments/" + id.String(),
		})

		req := httptest.NewRequest(http.MethodGet, "/attachments/"+id.String(), nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		var attachment services.Attachment
		json.NewDecoder(resp.Body).Decode(&attachment)
		assert.Equal(t, id, attachment.ID)
		assert.Equal(t, "test.txt", attachment.Filename)
	})

	t.Run("not found", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/attachments/"+uuid.New().String(), nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)
	})

	t.Run("invalid id", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/attachments/invalid", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
	})
}

func TestAttachmentHandler_Delete(t *testing.T) {
	svc := services.NewAttachmentService(nil)
	app, userID := setupAttachmentTestApp(svc)

	t.Run("success", func(t *testing.T) {
		channelID := uuid.New()
		id := uuid.New()
		
		svc.Upload_Test_Add(id, &services.Attachment{
			ID:          id,
			ChannelID:   channelID,
			UploaderID:  userID,
			Filename:    "test.txt",
			ContentType: "text/plain",
		})

		req := httptest.NewRequest(http.MethodDelete, "/attachments/"+id.String(), nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusNoContent, resp.StatusCode)
	})

	t.Run("not found", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, "/attachments/"+uuid.New().String(), nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)
	})

	t.Run("access denied", func(t *testing.T) {
		channelID := uuid.New()
		id := uuid.New()
		otherUserID := uuid.New()
		
		svc.Upload_Test_Add(id, &services.Attachment{
			ID:          id,
			ChannelID:   channelID,
			UploaderID:  otherUserID, // Different user
			Filename:    "test.txt",
			ContentType: "text/plain",
		})

		req := httptest.NewRequest(http.MethodDelete, "/attachments/"+id.String(), nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusForbidden, resp.StatusCode)
	})

	t.Run("invalid id", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, "/attachments/invalid", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
	})
}

func TestAttachmentHandler_GetChannelAttachments(t *testing.T) {
	svc := services.NewAttachmentService(nil)
	app, userID := setupAttachmentTestApp(svc)

	t.Run("empty list", func(t *testing.T) {
		channelID := uuid.New()
		req := httptest.NewRequest(http.MethodGet, "/channels/"+channelID.String()+"/attachments", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		var attachments []services.Attachment
		json.NewDecoder(resp.Body).Decode(&attachments)
		assert.Empty(t, attachments)
	})

	t.Run("with attachments", func(t *testing.T) {
		channelID := uuid.New()
		id1 := uuid.New()
		id2 := uuid.New()

		svc.Upload_Test_Add(id1, &services.Attachment{
			ID:          id1,
			ChannelID:   channelID,
			UploaderID:  userID,
			Filename:    "file1.txt",
			ContentType: "text/plain",
		})
		svc.Upload_Test_Add(id2, &services.Attachment{
			ID:          id2,
			ChannelID:   channelID,
			UploaderID:  userID,
			Filename:    "file2.txt",
			ContentType: "text/plain",
		})

		req := httptest.NewRequest(http.MethodGet, "/channels/"+channelID.String()+"/attachments", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		var attachments []services.Attachment
		json.NewDecoder(resp.Body).Decode(&attachments)
		assert.Len(t, attachments, 2)
	})

	t.Run("invalid channel id", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/channels/invalid/attachments", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
	})
}

func TestAttachmentHandler_GetSignedURL(t *testing.T) {
	svc := services.NewAttachmentService(nil)
	app, userID := setupAttachmentTestApp(svc)

	t.Run("success", func(t *testing.T) {
		channelID := uuid.New()
		id := uuid.New()

		svc.Upload_Test_Add(id, &services.Attachment{
			ID:          id,
			ChannelID:   channelID,
			UploaderID:  userID,
			Filename:    "test.txt",
			ContentType: "text/plain",
			URL:         "/attachments/" + id.String(),
		})

		req := httptest.NewRequest(http.MethodGet, "/attachments/"+id.String()+"/signed-url", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		var result map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&result)
		assert.NotEmpty(t, result["url"])
		assert.NotEmpty(t, result["expires_at"])
	})

	t.Run("not found", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/attachments/"+uuid.New().String()+"/signed-url", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)
	})

	t.Run("invalid id", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/attachments/invalid/signed-url", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
	})
}

func TestAttachmentHandler_Download(t *testing.T) {
	svc := services.NewAttachmentService(nil)
	app, userID := setupAttachmentTestApp(svc)

	t.Run("not found", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/attachments/"+uuid.New().String()+"/download", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)
	})

	t.Run("invalid id", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/attachments/invalid/download", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
	})

	t.Run("storage not configured", func(t *testing.T) {
		channelID := uuid.New()
		id := uuid.New()

		svc.Upload_Test_Add(id, &services.Attachment{
			ID:          id,
			ChannelID:   channelID,
			UploaderID:  userID,
			Filename:    "test.txt",
			ContentType: "text/plain",
			Path:        "attachments/test.txt",
		})

		req := httptest.NewRequest(http.MethodGet, "/attachments/"+id.String()+"/download", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		// Should fail because no storage is configured
		assert.Equal(t, fiber.StatusInternalServerError, resp.StatusCode)
	})
}

func TestAttachmentHandler_Unauthorized(t *testing.T) {
	svc := services.NewAttachmentService(nil)
	handler := NewAttachmentHandler(svc, nil)

	// Create app WITHOUT userID middleware
	app := fiber.New()
	app.Post("/channels/:id/attachments", handler.Upload)
	app.Delete("/attachments/:id", handler.Delete)

	t.Run("upload unauthorized", func(t *testing.T) {
		channelID := uuid.New()
		req, _ := createMultipartRequest(
			"/channels/"+channelID.String()+"/attachments",
			"test.txt",
			"text/plain",
			[]byte("content"),
		)

		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusUnauthorized, resp.StatusCode)
	})

	t.Run("delete unauthorized", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, "/attachments/"+uuid.New().String(), nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusUnauthorized, resp.StatusCode)
	})
}

// Test helper to read response body
func readBody(r io.Reader) string {
	body, _ := io.ReadAll(r)
	return string(body)
}
