package handlers

import (
	"bufio"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"hearth/internal/ai"
	"hearth/internal/models"
)

// AIChatHandler handles AI chat endpoints
type AIChatHandler struct {
	chatService *ai.ChatService
}

// NewAIChatHandler creates a new AIChatHandler
func NewAIChatHandler(chatService *ai.ChatService) *AIChatHandler {
	return &AIChatHandler{chatService: chatService}
}

// --- Conversations ---

// ListConversations lists user's AI conversations
// @Summary List AI conversations
// @Description Returns a paginated list of the user's AI conversations with optional filtering
// @Tags AI Chat
// @Produce json
// @Param include_archived query boolean false "Include archived conversations" default(false)
// @Param pinned_only query boolean false "Only return pinned conversations" default(false)
// @Param search query string false "Search query for conversation titles"
// @Param limit query integer false "Number of results to return" default(50)
// @Param offset query integer false "Offset for pagination" default(0)
// @Success 200 {object} fiber.Map "List of conversations"
// @Failure 401 {object} fiber.Map "Unauthorized"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /ai/conversations [get]
func (h *AIChatHandler) ListConversations(c *fiber.Ctx) error {
	userID, ok := c.Locals("userID").(uuid.UUID)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error":   "unauthorized",
			"message": "User ID not found in context",
		})
	}

	params := &models.ConversationListParams{
		UserID:          userID,
		IncludeArchived: c.Query("include_archived") == "true",
		OnlyPinned:      c.Query("pinned_only") == "true",
		Search:          c.Query("search"),
		Limit:           c.QueryInt("limit", 50),
		Offset:          c.QueryInt("offset", 0),
	}

	conversations, err := h.chatService.ListConversations(c.Context(), params)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "internal_error",
			"message": "Failed to list conversations",
		})
	}

	return c.JSON(fiber.Map{
		"conversations": conversations,
	})
}

// GetConversation gets a specific conversation with messages
// @Summary Get conversation details
// @Description Returns a specific conversation with its messages
// @Tags AI Chat
// @Produce json
// @Param id path string true "Conversation ID"
// @Param message_limit query integer false "Maximum number of messages to return" default(50)
// @Success 200 {object} models.Conversation "Conversation with messages"
// @Failure 400 {object} fiber.Map "Invalid conversation ID"
// @Failure 401 {object} fiber.Map "Unauthorized"
// @Failure 403 {object} fiber.Map "Forbidden - not authorized to access this conversation"
// @Failure 404 {object} fiber.Map "Conversation not found"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /ai/conversations/{id} [get]
func (h *AIChatHandler) GetConversation(c *fiber.Ctx) error {
	userID, ok := c.Locals("userID").(uuid.UUID)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error":   "unauthorized",
			"message": "User ID not found in context",
		})
	}

	conversationID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "invalid_request",
			"message": "Invalid conversation ID",
		})
	}

	limit := c.QueryInt("message_limit", 50)

	conv, err := h.chatService.GetConversationWithMessages(c.Context(), userID, conversationID, limit)
	if err != nil {
		if err == ai.ErrConversationNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error":   "not_found",
				"message": "Conversation not found",
			})
		}
		if err == ai.ErrNotAuthorized {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error":   "forbidden",
				"message": "Not authorized to access this conversation",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "internal_error",
			"message": "Failed to get conversation",
		})
	}

	return c.JSON(conv)
}

// CreateConversation creates a new AI conversation
// @Summary Create new conversation
// @Description Creates a new AI conversation with optional initial message
// @Tags AI Chat
// @Accept json
// @Produce json
// @Param body body models.CreateConversationRequest true "Conversation creation request"
// @Success 201 {object} models.Conversation "Conversation created successfully"
// @Failure 400 {object} fiber.Map "Invalid request body"
// @Failure 401 {object} fiber.Map "Unauthorized"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /ai/conversations [post]
func (h *AIChatHandler) CreateConversation(c *fiber.Ctx) error {
	userID, ok := c.Locals("userID").(uuid.UUID)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error":   "unauthorized",
			"message": "User ID not found in context",
		})
	}

	var req models.CreateConversationRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "invalid_request",
			"message": "Invalid request body",
		})
	}

	conv, err := h.chatService.CreateConversation(c.Context(), userID, &req)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "internal_error",
			"message": err.Error(),
		})
	}

	return c.Status(fiber.StatusCreated).JSON(conv)
}

// UpdateConversation updates a conversation
// @Summary Update conversation
// @Description Updates conversation metadata such as title, model, or pinned status
// @Tags AI Chat
// @Accept json
// @Produce json
// @Param id path string true "Conversation ID"
// @Param body body models.UpdateConversationRequest true "Conversation update data"
// @Success 200 {object} models.Conversation "Conversation updated successfully"
// @Failure 400 {object} fiber.Map "Invalid request or conversation ID"
// @Failure 401 {object} fiber.Map "Unauthorized"
// @Failure 403 {object} fiber.Map "Forbidden - not authorized to update this conversation"
// @Failure 404 {object} fiber.Map "Conversation not found"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /ai/conversations/{id} [patch]
func (h *AIChatHandler) UpdateConversation(c *fiber.Ctx) error {
	userID, ok := c.Locals("userID").(uuid.UUID)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error":   "unauthorized",
			"message": "User ID not found in context",
		})
	}

	conversationID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "invalid_request",
			"message": "Invalid conversation ID",
		})
	}

	var req models.UpdateConversationRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "invalid_request",
			"message": "Invalid request body",
		})
	}

	conv, err := h.chatService.UpdateConversation(c.Context(), userID, conversationID, &req)
	if err != nil {
		if err == ai.ErrConversationNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error":   "not_found",
				"message": "Conversation not found",
			})
		}
		if err == ai.ErrNotAuthorized {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error":   "forbidden",
				"message": "Not authorized to update this conversation",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "internal_error",
			"message": err.Error(),
		})
	}

	return c.JSON(conv)
}

// DeleteConversation deletes a conversation
// @Summary Delete conversation
// @Description Permanently deletes a conversation and all its messages
// @Tags AI Chat
// @Param id path string true "Conversation ID"
// @Success 204 "Conversation deleted successfully"
// @Failure 400 {object} fiber.Map "Invalid conversation ID"
// @Failure 401 {object} fiber.Map "Unauthorized"
// @Failure 403 {object} fiber.Map "Forbidden - not authorized to delete this conversation"
// @Failure 404 {object} fiber.Map "Conversation not found"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /ai/conversations/{id} [delete]
func (h *AIChatHandler) DeleteConversation(c *fiber.Ctx) error {
	userID, ok := c.Locals("userID").(uuid.UUID)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error":   "unauthorized",
			"message": "User ID not found in context",
		})
	}

	conversationID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "invalid_request",
			"message": "Invalid conversation ID",
		})
	}

	if err := h.chatService.DeleteConversation(c.Context(), userID, conversationID); err != nil {
		if err == ai.ErrConversationNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error":   "not_found",
				"message": "Conversation not found",
			})
		}
		if err == ai.ErrNotAuthorized {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error":   "forbidden",
				"message": "Not authorized to delete this conversation",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "internal_error",
			"message": err.Error(),
		})
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// --- Messages ---

// GetMessages gets messages for a conversation
// @Summary Get conversation messages
// @Description Returns paginated messages for a specific conversation
// @Tags AI Chat
// @Produce json
// @Param id path string true "Conversation ID"
// @Param limit query integer false "Number of messages to return" default(50)
// @Param offset query integer false "Offset for pagination" default(0)
// @Success 200 {object} fiber.Map "List of messages"
// @Failure 400 {object} fiber.Map "Invalid conversation ID"
// @Failure 401 {object} fiber.Map "Unauthorized"
// @Failure 404 {object} fiber.Map "Conversation not found or not authorized"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /ai/conversations/{id}/messages [get]
func (h *AIChatHandler) GetMessages(c *fiber.Ctx) error {
	userID, ok := c.Locals("userID").(uuid.UUID)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error":   "unauthorized",
			"message": "User ID not found in context",
		})
	}

	conversationID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "invalid_request",
			"message": "Invalid conversation ID",
		})
	}

	limit := c.QueryInt("limit", 50)
	offset := c.QueryInt("offset", 0)

	messages, err := h.chatService.GetConversationMessages(c.Context(), userID, conversationID, limit, offset)
	if err != nil {
		if err == ai.ErrConversationNotFound || err == ai.ErrNotAuthorized {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error":   "not_found",
				"message": "Conversation not found or not authorized",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "internal_error",
			"message": "Failed to get messages",
		})
	}

	return c.JSON(fiber.Map{
		"messages": messages,
	})
}

// SendMessage sends a message to the conversation and gets AI response
// @Summary Send message to AI
// @Description Sends a message to the conversation and returns the AI's response (non-streaming)
// @Tags AI Chat
// @Accept json
// @Produce json
// @Param id path string true "Conversation ID"
// @Param body body models.SendChatMessageRequest true "Message request"
// @Success 201 {object} models.ChatMessage "AI response message created"
// @Failure 400 {object} fiber.Map "Invalid request or empty message content"
// @Failure 401 {object} fiber.Map "Unauthorized"
// @Failure 403 {object} fiber.Map "Forbidden - not authorized to send messages"
// @Failure 404 {object} fiber.Map "Conversation not found"
// @Failure 500 {object} fiber.Map "AI processing error"
// @Router /ai/conversations/{id}/messages [post]
func (h *AIChatHandler) SendMessage(c *fiber.Ctx) error {
	userID, ok := c.Locals("userID").(uuid.UUID)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error":   "unauthorized",
			"message": "User ID not found in context",
		})
	}

	conversationID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "invalid_request",
			"message": "Invalid conversation ID",
		})
	}

	var req models.SendChatMessageRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "invalid_request",
			"message": "Invalid request body",
		})
	}

	if strings.TrimSpace(req.Content) == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "invalid_request",
			"message": "Message content is required",
		})
	}

	// Handle streaming response
	if req.Stream {
		return h.handleStreamingMessage(c, userID, conversationID, &req)
	}

	// Non-streaming response
	msg, err := h.chatService.SendMessage(c.Context(), userID, conversationID, &req)
	if err != nil {
		if err == ai.ErrConversationNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error":   "not_found",
				"message": "Conversation not found",
			})
		}
		if err == ai.ErrNotAuthorized {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error":   "forbidden",
				"message": "Not authorized to send messages to this conversation",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "ai_error",
			"message": err.Error(),
		})
	}

	return c.Status(fiber.StatusCreated).JSON(msg)
}

// handleStreamingMessage handles SSE streaming for chat messages
func (h *AIChatHandler) handleStreamingMessage(c *fiber.Ctx, userID, conversationID uuid.UUID, req *models.SendChatMessageRequest) error {
	c.Set("Content-Type", "text/event-stream")
	c.Set("Cache-Control", "no-cache")
	c.Set("Connection", "keep-alive")
	c.Set("Transfer-Encoding", "chunked")
	c.Set("X-Accel-Buffering", "no") // Disable nginx buffering

	c.Context().SetBodyStreamWriter(func(w *bufio.Writer) {
		// Send initial connection event
		fmt.Fprintf(w, "event: connected\ndata: {\"status\":\"connected\"}\n\n")
		w.Flush()

		callback := func(chunk *models.StreamChunk) error {
			data, err := json.Marshal(chunk)
			if err != nil {
				return err
			}
			fmt.Fprintf(w, "data: %s\n\n", string(data))
			return w.Flush()
		}

		msg, err := h.chatService.SendMessageStream(c.Context(), userID, conversationID, req, callback)
		if err != nil {
			errData, _ := json.Marshal(fiber.Map{
				"error":   "stream_error",
				"message": err.Error(),
			})
			fmt.Fprintf(w, "event: error\ndata: %s\n\n", string(errData))
			w.Flush()
			return
		}

		// Send final message with complete data
		finalData, _ := json.Marshal(fiber.Map{
			"message": msg,
			"done":    true,
		})
		fmt.Fprintf(w, "event: done\ndata: %s\n\n", string(finalData))
		w.Flush()
	})

	return nil
}

// RegenerateMessage regenerates an assistant message
// @Summary Regenerate AI message
// @Description Regenerates an assistant message with optional streaming support
// @Tags AI Chat
// @Accept json
// @Produce json
// @Param id path string true "Conversation ID"
// @Param messageId path string true "Message ID to regenerate"
// @Param stream query boolean false "Enable streaming response" default(false)
// @Success 200 {object} models.ChatMessage "Regenerated message (non-streaming)"
// @Failure 400 {object} fiber.Map "Invalid conversation or message ID"
// @Failure 401 {object} fiber.Map "Unauthorized"
// @Failure 404 {object} fiber.Map "Message not found"
// @Failure 500 {object} fiber.Map "AI regeneration error"
// @Router /ai/conversations/{id}/messages/{messageId}/regenerate [post]
func (h *AIChatHandler) RegenerateMessage(c *fiber.Ctx) error {
	userID, ok := c.Locals("userID").(uuid.UUID)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error":   "unauthorized",
			"message": "User ID not found in context",
		})
	}

	conversationID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "invalid_request",
			"message": "Invalid conversation ID",
		})
	}

	messageID, err := uuid.Parse(c.Params("messageId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "invalid_request",
			"message": "Invalid message ID",
		})
	}

	stream := c.Query("stream") == "true"

	if stream {
		c.Set("Content-Type", "text/event-stream")
		c.Set("Cache-Control", "no-cache")
		c.Set("Connection", "keep-alive")
		c.Set("X-Accel-Buffering", "no")

		c.Context().SetBodyStreamWriter(func(w *bufio.Writer) {
			callback := func(chunk *models.StreamChunk) error {
				data, _ := json.Marshal(chunk)
				fmt.Fprintf(w, "data: %s\n\n", string(data))
				return w.Flush()
			}

			msg, err := h.chatService.RegenerateMessage(c.Context(), userID, conversationID, messageID, true, callback)
			if err != nil {
				errData, _ := json.Marshal(fiber.Map{"error": err.Error()})
				fmt.Fprintf(w, "event: error\ndata: %s\n\n", string(errData))
				w.Flush()
				return
			}

			finalData, _ := json.Marshal(fiber.Map{"message": msg, "done": true})
			fmt.Fprintf(w, "event: done\ndata: %s\n\n", string(finalData))
			w.Flush()
		})
		return nil
	}

	msg, err := h.chatService.RegenerateMessage(c.Context(), userID, conversationID, messageID, false, nil)
	if err != nil {
		if err == ai.ErrMessageNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error":   "not_found",
				"message": "Message not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "ai_error",
			"message": err.Error(),
		})
	}

	return c.JSON(msg)
}

// DeleteMessage deletes a message
// @Summary Delete message
// @Description Deletes a specific message from a conversation
// @Tags AI Chat
// @Param id path string true "Conversation ID"
// @Param messageId path string true "Message ID to delete"
// @Success 204 "Message deleted successfully"
// @Failure 400 {object} fiber.Map "Invalid conversation or message ID"
// @Failure 401 {object} fiber.Map "Unauthorized"
// @Failure 404 {object} fiber.Map "Message not found"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /ai/conversations/{id}/messages/{messageId} [delete]
func (h *AIChatHandler) DeleteMessage(c *fiber.Ctx) error {
	userID, ok := c.Locals("userID").(uuid.UUID)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error":   "unauthorized",
			"message": "User ID not found in context",
		})
	}

	conversationID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "invalid_request",
			"message": "Invalid conversation ID",
		})
	}

	messageID, err := uuid.Parse(c.Params("messageId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "invalid_request",
			"message": "Invalid message ID",
		})
	}

	if err := h.chatService.DeleteMessage(c.Context(), userID, conversationID, messageID); err != nil {
		if err == ai.ErrMessageNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error":   "not_found",
				"message": "Message not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "internal_error",
			"message": err.Error(),
		})
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// --- Templates ---

// ListTemplates lists available chat templates
// @Summary List chat templates
// @Description Returns available AI chat templates, optionally filtered by category
// @Tags AI Chat
// @Produce json
// @Param category query string false "Filter by template category"
// @Success 200 {object} fiber.Map "List of templates"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /ai/templates [get]
func (h *AIChatHandler) ListTemplates(c *fiber.Ctx) error {
	var userID *uuid.UUID
	if uid, ok := c.Locals("userID").(uuid.UUID); ok {
		userID = &uid
	}

	category := c.Query("category")

	templates, err := h.chatService.GetTemplates(c.Context(), userID, category)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "internal_error",
			"message": "Failed to list templates",
		})
	}

	return c.JSON(fiber.Map{
		"templates": templates,
	})
}

// GetTemplate gets a specific template
// @Summary Get template details
// @Description Returns details of a specific chat template
// @Tags AI Chat
// @Produce json
// @Param id path string true "Template ID"
// @Success 200 {object} models.AIChatTemplate "Template details"
// @Failure 400 {object} fiber.Map "Invalid template ID"
// @Failure 404 {object} fiber.Map "Template not found"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /ai/templates/{id} [get]
func (h *AIChatHandler) GetTemplate(c *fiber.Ctx) error {
	templateID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "invalid_request",
			"message": "Invalid template ID",
		})
	}

	template, err := h.chatService.GetTemplate(c.Context(), templateID)
	if err != nil {
		if err == ai.ErrTemplateNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error":   "not_found",
				"message": "Template not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "internal_error",
			"message": "Failed to get template",
		})
	}

	return c.JSON(template)
}

// CreateTemplate creates a new template
// @Summary Create chat template
// @Description Creates a new custom AI chat template
// @Tags AI Chat
// @Accept json
// @Produce json
// @Param body body models.AIChatTemplate true "Template data"
// @Success 201 {object} models.AIChatTemplate "Template created successfully"
// @Failure 400 {object} fiber.Map "Invalid request body"
// @Failure 401 {object} fiber.Map "Unauthorized"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /ai/templates [post]
func (h *AIChatHandler) CreateTemplate(c *fiber.Ctx) error {
	userID, ok := c.Locals("userID").(uuid.UUID)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error":   "unauthorized",
			"message": "User ID not found in context",
		})
	}

	var tmpl models.AIChatTemplate
	if err := c.BodyParser(&tmpl); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "invalid_request",
			"message": "Invalid request body",
		})
	}

	if err := h.chatService.CreateTemplate(c.Context(), userID, &tmpl); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "internal_error",
			"message": err.Error(),
		})
	}

	return c.Status(fiber.StatusCreated).JSON(tmpl)
}

// UpdateTemplate updates a template
// @Summary Update chat template
// @Description Updates an existing chat template
// @Tags AI Chat
// @Accept json
// @Produce json
// @Param id path string true "Template ID"
// @Param body body models.AIChatTemplate true "Updated template data"
// @Success 200 {object} models.AIChatTemplate "Template updated successfully"
// @Failure 400 {object} fiber.Map "Invalid request or template ID"
// @Failure 401 {object} fiber.Map "Unauthorized"
// @Failure 403 {object} fiber.Map "Forbidden - not authorized to update this template"
// @Failure 404 {object} fiber.Map "Template not found"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /ai/templates/{id} [patch]
func (h *AIChatHandler) UpdateTemplate(c *fiber.Ctx) error {
	userID, ok := c.Locals("userID").(uuid.UUID)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error":   "unauthorized",
			"message": "User ID not found in context",
		})
	}

	templateID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "invalid_request",
			"message": "Invalid template ID",
		})
	}

	var tmpl models.AIChatTemplate
	if err := c.BodyParser(&tmpl); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "invalid_request",
			"message": "Invalid request body",
		})
	}
	tmpl.ID = templateID

	if err := h.chatService.UpdateTemplate(c.Context(), userID, &tmpl); err != nil {
		if err == ai.ErrTemplateNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error":   "not_found",
				"message": "Template not found",
			})
		}
		if err == ai.ErrNotAuthorized {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error":   "forbidden",
				"message": "Not authorized to update this template",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "internal_error",
			"message": err.Error(),
		})
	}

	return c.JSON(tmpl)
}

// DeleteTemplate deletes a template
// @Summary Delete chat template
// @Description Deletes a chat template
// @Tags AI Chat
// @Param id path string true "Template ID"
// @Success 204 "Template deleted successfully"
// @Failure 400 {object} fiber.Map "Invalid template ID"
// @Failure 401 {object} fiber.Map "Unauthorized"
// @Failure 403 {object} fiber.Map "Forbidden - not authorized to delete this template"
// @Failure 404 {object} fiber.Map "Template not found"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /ai/templates/{id} [delete]
func (h *AIChatHandler) DeleteTemplate(c *fiber.Ctx) error {
	userID, ok := c.Locals("userID").(uuid.UUID)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error":   "unauthorized",
			"message": "User ID not found in context",
		})
	}

	templateID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "invalid_request",
			"message": "Invalid template ID",
		})
	}

	if err := h.chatService.DeleteTemplate(c.Context(), userID, templateID); err != nil {
		if err == ai.ErrTemplateNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error":   "not_found",
				"message": "Template not found",
			})
		}
		if err == ai.ErrNotAuthorized {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error":   "forbidden",
				"message": "Not authorized to delete this template",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "internal_error",
			"message": err.Error(),
		})
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// --- Shares ---

// ShareConversation creates a shareable link
// @Summary Share conversation
// @Description Creates a shareable link for a conversation
// @Tags AI Chat
// @Accept json
// @Produce json
// @Param id path string true "Conversation ID"
// @Param body body struct{IsPublic bool `json:"is_public"`; CanContinue bool `json:"can_continue"`; ExpiresIn string `json:"expires_in"`} true "Share settings"
// @Success 201 {object} models.ConversationShare "Share created successfully"
// @Failure 400 {object} fiber.Map "Invalid request or conversation ID"
// @Failure 401 {object} fiber.Map "Unauthorized"
// @Failure 404 {object} fiber.Map "Conversation not found"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /ai/conversations/{id}/share [post]
func (h *AIChatHandler) ShareConversation(c *fiber.Ctx) error {
	userID, ok := c.Locals("userID").(uuid.UUID)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error":   "unauthorized",
			"message": "User ID not found in context",
		})
	}

	conversationID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "invalid_request",
			"message": "Invalid conversation ID",
		})
	}

	var req struct {
		IsPublic    bool   `json:"is_public"`
		CanContinue bool   `json:"can_continue"`
		ExpiresIn   string `json:"expires_in"` // e.g., "24h", "7d"
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "invalid_request",
			"message": "Invalid request body",
		})
	}

	var expiresIn *time.Duration
	if req.ExpiresIn != "" {
		duration, err := time.ParseDuration(req.ExpiresIn)
		if err == nil {
			expiresIn = &duration
		}
	}

	share, err := h.chatService.ShareConversation(c.Context(), userID, conversationID, req.IsPublic, req.CanContinue, expiresIn)
	if err != nil {
		if err == ai.ErrConversationNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error":   "not_found",
				"message": "Conversation not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "internal_error",
			"message": err.Error(),
		})
	}

	return c.Status(fiber.StatusCreated).JSON(share)
}

// GetSharedConversation gets a shared conversation by code
// @Summary Get shared conversation
// @Description Retrieves a shared conversation using its share code
// @Tags AI Chat
// @Produce json
// @Param code path string true "Share code"
// @Success 200 {object} models.Conversation "Shared conversation details"
// @Failure 404 {object} fiber.Map "Share not found"
// @Failure 410 {object} fiber.Map "Share has expired"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /ai/shared/{code} [get]
func (h *AIChatHandler) GetSharedConversation(c *fiber.Ctx) error {
	shareCode := c.Params("code")

	conv, err := h.chatService.GetSharedConversation(c.Context(), shareCode)
	if err != nil {
		if err == ai.ErrShareNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error":   "not_found",
				"message": "Share not found",
			})
		}
		if err == ai.ErrShareExpired {
			return c.Status(fiber.StatusGone).JSON(fiber.Map{
				"error":   "expired",
				"message": "Share has expired",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "internal_error",
			"message": err.Error(),
		})
	}

	return c.JSON(conv)
}
