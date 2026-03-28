package handlers

import (
	"bufio"
	"encoding/json"
	"fmt"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"hearth/internal/ai"
	"hearth/internal/ai/providers"
)

// AIHandler handles AI-related endpoints
type AIHandler struct {
	aiService *ai.AIService
}

// NewAIHandler creates a new AIHandler
func NewAIHandler(aiService *ai.AIService) *AIHandler {
	return &AIHandler{aiService: aiService}
}

// --- Error Response Helpers ---

// respondBadRequest sends a 400 Bad Request with the given message
func (h *AIHandler) respondBadRequest(c *fiber.Ctx, message string) error {
	return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
		"error":   "invalid_request",
		"message": message,
	})
}

// respondUnauthorized sends a 401 Unauthorized response
func (h *AIHandler) respondUnauthorized(c *fiber.Ctx, message string) error {
	return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
		"error":   "unauthorized",
		"message": message,
	})
}

// respondNotFound sends a 404 Not Found with the given message
func (h *AIHandler) respondNotFound(c *fiber.Ctx, message string) error {
	return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
		"error":   "not_found",
		"message": message,
	})
}

// respondInternalError sends a 500 Internal Server Error with the given message
func (h *AIHandler) respondInternalError(c *fiber.Ctx, message string) error {
	return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
		"error":   "internal_error",
		"message": message,
	})
}

// getUserID extracts and validates the user ID from context
func (h *AIHandler) getUserID(c *fiber.Ctx) (uuid.UUID, error) {
	userID, ok := c.Locals("userID").(uuid.UUID)
	if !ok {
		return uuid.UUID{}, fmt.Errorf("user ID not found in context")
	}
	return userID, nil
}

// --- Provider Management (Admin) ---

// GetProviders returns all configured AI providers
// @Summary Get all AI providers
// @Description Returns a list of all configured AI providers
// @Tags AI
// @Produce json
// @Success 200 {object} fiber.Map{providers=[]ai.AIProviderConfigResponse} "List of providers"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /ai/providers [get]
func (h *AIHandler) GetProviders(c *fiber.Ctx) error {
	configs, err := h.aiService.GetAllProviders(c.Context())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "internal_error",
			"message": "Failed to retrieve providers",
		})
	}

	responses := make([]ai.AIProviderConfigResponse, len(configs))
	for i, cfg := range configs {
		responses[i] = cfg.ToResponse()
	}

	return c.JSON(fiber.Map{
		"providers": responses,
	})
}

// GetProviderModels returns models for a specific provider
// @Summary Get provider models
// @Description Returns available models for a specific AI provider
// @Tags AI
// @Produce json
// @Param id path string true "Provider ID"
// @Success 200 {object} fiber.Map{models=[]string} "List of models"
// @Failure 400 {object} fiber.Map "Invalid provider ID"
// @Failure 404 {object} fiber.Map "Provider not found"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /ai/providers/{id}/models [get]
func (h *AIHandler) GetProviderModels(c *fiber.Ctx) error {
	idParam := c.Params("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "invalid_request",
			"message": "Invalid provider ID",
		})
	}

	// Get user ID for credential lookup
	var userID *uuid.UUID
	if uid, ok := c.Locals("userID").(uuid.UUID); ok {
		userID = &uid
	}

	// Get provider instance
	provider, err := h.aiService.GetProviderInstance(c.Context(), id, userID)
	if err != nil {
		if err == ai.ErrProviderNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error":   "not_found",
				"message": "Provider not found",
			})
		}
		if err == ai.ErrProviderNotEnabled {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error":   "provider_disabled",
				"message": "Provider is not enabled",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "internal_error",
			"message": "Failed to get provider",
		})
	}

	// List models from the provider
	models, err := provider.ListModels(c.Context())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "provider_error",
			"message": "Failed to list models from provider",
		})
	}

	return c.JSON(fiber.Map{
		"models": models,
	})
}

// GetProvider returns a specific AI provider
// @Summary Get AI provider by ID
// @Description Returns details of a specific AI provider
// @Tags AI
// @Produce json
// @Param id path string true "Provider ID"
// @Success 200 {object} ai.AIProviderConfigResponse "Provider details"
// @Failure 400 {object} fiber.Map "Invalid provider ID"
// @Failure 404 {object} fiber.Map "Provider not found"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /ai/providers/{id} [get]
func (h *AIHandler) GetProvider(c *fiber.Ctx) error {
	idParam := c.Params("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "invalid_request",
			"message": "Invalid provider ID",
		})
	}

	config, err := h.aiService.GetProvider(c.Context(), id)
	if err != nil {
		if err == ai.ErrProviderNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error":   "not_found",
				"message": "Provider not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "internal_error",
			"message": "Failed to retrieve provider",
		})
	}

	return c.JSON(config.ToResponse())
}

// CreateProviderRequest is the request body for creating a provider
type CreateProviderRequest struct {
	ProviderType string  `json:"provider_type"`
	Name         string  `json:"name"`
	DisplayName  string  `json:"display_name"`
	BaseURL      *string `json:"base_url,omitempty"`
	IsEnabled    bool    `json:"is_enabled"`
	IsDefault    bool    `json:"is_default"`
	Priority     int     `json:"priority"`

	// Credentials
	APIKey         string `json:"api_key,omitempty"`
	SecretKey      string `json:"secret_key,omitempty"`
	Region         string `json:"region,omitempty"`
	ProjectID      string `json:"project_id,omitempty"`
	ServiceAccount string `json:"service_account,omitempty"`
}

// CreateProvider creates a new AI provider configuration
// @Summary Create AI provider
// @Description Creates a new AI provider configuration
// @Tags AI
// @Accept json
// @Produce json
// @Param body body CreateProviderRequest true "Provider configuration"
// @Success 201 {object} ai.AIProviderConfigResponse "Provider created"
// @Failure 400 {object} fiber.Map "Invalid request or provider type"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /ai/providers [post]
func (h *AIHandler) CreateProvider(c *fiber.Ctx) error {
	var req CreateProviderRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "invalid_request",
			"message": "Invalid request body",
		})
	}

	// Validate provider type
	providerType := ai.ProviderType(req.ProviderType)
	if !providerType.Valid() {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "invalid_request",
			"message": "Invalid provider type",
		})
	}

	config := &ai.AIProviderConfig{
		ProviderType: providerType,
		Name:         req.Name,
		DisplayName:  req.DisplayName,
		BaseURL:      req.BaseURL,
		IsEnabled:    req.IsEnabled,
		IsDefault:    req.IsDefault,
		Priority:     req.Priority,
	}

	var credentials *providers.ProviderConfig
	if req.APIKey != "" || req.SecretKey != "" || req.ServiceAccount != "" {
		credentials = &providers.ProviderConfig{
			APIKey:         req.APIKey,
			SecretKey:      req.SecretKey,
			Region:         req.Region,
			ProjectID:      req.ProjectID,
			ServiceAccount: req.ServiceAccount,
		}
	}

	if err := h.aiService.CreateProvider(c.Context(), config, credentials); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "internal_error",
			"message": err.Error(),
		})
	}

	return c.Status(fiber.StatusCreated).JSON(config.ToResponse())
}

// UpdateProvider updates an AI provider configuration
// @Summary Update AI provider
// @Description Updates an existing AI provider configuration
// @Tags AI
// @Accept json
// @Produce json
// @Param id path string true "Provider ID"
// @Param body body CreateProviderRequest true "Provider update data"
// @Success 200 {object} ai.AIProviderConfigResponse "Provider updated"
// @Failure 400 {object} fiber.Map "Invalid request or provider ID"
// @Failure 404 {object} fiber.Map "Provider not found"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /ai/providers/{id} [patch]
func (h *AIHandler) UpdateProvider(c *fiber.Ctx) error {
	idParam := c.Params("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "invalid_request",
			"message": "Invalid provider ID",
		})
	}

	var req CreateProviderRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "invalid_request",
			"message": "Invalid request body",
		})
	}

	// Get existing config
	existing, err := h.aiService.GetProvider(c.Context(), id)
	if err != nil {
		if err == ai.ErrProviderNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error":   "not_found",
				"message": "Provider not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "internal_error",
			"message": "Failed to retrieve provider",
		})
	}

	// Update fields
	if req.Name != "" {
		existing.Name = req.Name
	}
	if req.DisplayName != "" {
		existing.DisplayName = req.DisplayName
	}
	if req.BaseURL != nil {
		existing.BaseURL = req.BaseURL
	}
	existing.IsEnabled = req.IsEnabled
	existing.IsDefault = req.IsDefault
	if req.Priority > 0 {
		existing.Priority = req.Priority
	}

	var credentials *providers.ProviderConfig
	if req.APIKey != "" || req.SecretKey != "" || req.ServiceAccount != "" {
		credentials = &providers.ProviderConfig{
			APIKey:         req.APIKey,
			SecretKey:      req.SecretKey,
			Region:         req.Region,
			ProjectID:      req.ProjectID,
			ServiceAccount: req.ServiceAccount,
		}
	}

	if err := h.aiService.UpdateProvider(c.Context(), existing, credentials); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "internal_error",
			"message": err.Error(),
		})
	}

	return c.JSON(existing.ToResponse())
}

// DeleteProvider deletes an AI provider configuration
// @Summary Delete AI provider
// @Description Deletes an AI provider configuration
// @Tags AI
// @Param id path string true "Provider ID"
// @Success 204 "Provider deleted"
// @Failure 400 {object} fiber.Map "Invalid provider ID"
// @Failure 404 {object} fiber.Map "Provider not found"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /ai/providers/{id} [delete]
func (h *AIHandler) DeleteProvider(c *fiber.Ctx) error {
	idParam := c.Params("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "invalid_request",
			"message": "Invalid provider ID",
		})
	}

	if err := h.aiService.DeleteProvider(c.Context(), id); err != nil {
		if err == ai.ErrProviderNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error":   "not_found",
				"message": "Provider not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "internal_error",
			"message": "Failed to delete provider",
		})
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// --- User Credentials ---

// GetUserCredentials returns the current user's AI credentials
// @Summary Get user AI credentials
// @Description Returns the current user's AI provider credentials
// @Tags AI
// @Produce json
// @Success 200 {object} fiber.Map{credentials=[]*ai.UserAICredentialResponse} "List of credentials"
// @Failure 401 {object} fiber.Map "Unauthorized"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /ai/providers/{id}/credentials [get]
func (h *AIHandler) GetUserCredentials(c *fiber.Ctx) error {
	userID, err := h.getUserID(c)
	if err != nil {
		return h.respondUnauthorized(c, err.Error())
	}

	creds, err := h.aiService.GetUserCredentials(c.Context(), userID)
	if err != nil {
		return h.respondInternalError(c, "Failed to retrieve credentials")
	}

	return c.JSON(fiber.Map{
		"credentials": creds,
	})
}

// SetUserCredentialsRequest is the request body for setting user credentials
type SetUserCredentialsRequest struct {
	ProviderID     string `json:"provider_id"`
	APIKey         string `json:"api_key,omitempty"`
	SecretKey      string `json:"secret_key,omitempty"`
	Region         string `json:"region,omitempty"`
	ProjectID      string `json:"project_id,omitempty"`
	ServiceAccount string `json:"service_account,omitempty"`
}

// SetUserCredentials sets user-specific AI credentials
// @Summary Set user AI credentials
// @Description Sets or updates user-specific AI provider credentials
// @Tags AI
// @Accept json
// @Produce json
// @Param body body SetUserCredentialsRequest true "Credentials data"
// @Success 200 {object} fiber.Map "Credentials saved successfully"
// @Failure 400 {object} fiber.Map "Invalid request or provider ID"
// @Failure 401 {object} fiber.Map "Unauthorized"
// @Failure 404 {object} fiber.Map "Provider not found"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /ai/providers/{id}/credentials [post]
func (h *AIHandler) SetUserCredentials(c *fiber.Ctx) error {
	userID, err := h.getUserID(c)
	if err != nil {
		return h.respondUnauthorized(c, err.Error())
	}

	var req SetUserCredentialsRequest
	if err := c.BodyParser(&req); err != nil {
		return h.respondBadRequest(c, "Invalid request body")
	}

	providerID, err := uuid.Parse(req.ProviderID)
	if err != nil {
		return h.respondBadRequest(c, "Invalid provider ID")
	}

	credentials := &providers.ProviderConfig{
		APIKey:         req.APIKey,
		SecretKey:      req.SecretKey,
		Region:         req.Region,
		ProjectID:      req.ProjectID,
		ServiceAccount: req.ServiceAccount,
	}

	if err := h.aiService.SetUserCredentials(c.Context(), userID, providerID, credentials); err != nil {
		if err == ai.ErrProviderNotFound {
			return h.respondNotFound(c, "Provider not found")
		}
		return h.respondInternalError(c, err.Error())
	}

	return c.JSON(fiber.Map{
		"message": "Credentials saved successfully",
	})
}

// DeleteUserCredential deletes user credentials for a provider
// @Summary Delete user AI credentials
// @Description Deletes the current user's credentials for a specific provider
// @Tags AI
// @Param providerId path string true "Provider ID"
// @Success 204 "Credentials deleted"
// @Failure 400 {object} fiber.Map "Invalid provider ID"
// @Failure 401 {object} fiber.Map "Unauthorized"
// @Failure 404 {object} fiber.Map "Credentials not found"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /ai/providers/{id}/credentials [delete]
func (h *AIHandler) DeleteUserCredential(c *fiber.Ctx) error {
	userID, ok := c.Locals("userID").(uuid.UUID)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error":   "unauthorized",
			"message": "User ID not found in context",
		})
	}

	providerIDParam := c.Params("providerId")
	providerID, err := uuid.Parse(providerIDParam)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "invalid_request",
			"message": "Invalid provider ID",
		})
	}

	if err := h.aiService.DeleteUserCredential(c.Context(), userID, providerID); err != nil {
		if err == ai.ErrCredentialsNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error":   "not_found",
				"message": "Credentials not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "internal_error",
			"message": "Failed to delete credentials",
		})
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// --- Model Routing ---

// GetModelRoutings returns all model routing configurations
// @Summary Get model routings
// @Description Returns all model routing configurations
// @Tags AI
// @Produce json
// @Success 200 {object} fiber.Map{routing=[]*ai.ModelRouting} "List of routing configurations"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /ai/routing [get]
func (h *AIHandler) GetModelRoutings(c *fiber.Ctx) error {
	routings, err := h.aiService.GetAllModelRoutings(c.Context())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "internal_error",
			"message": "Failed to retrieve routing configurations",
		})
	}

	return c.JSON(fiber.Map{
		"routing": routings,
	})
}

// SetModelRoutingRequest is the request body for setting model routing
type SetModelRoutingRequest struct {
	ServerID   *string `json:"server_id,omitempty"`
	UserID     *string `json:"user_id,omitempty"`
	Feature    string  `json:"feature"`
	ProviderID string  `json:"provider_id"`
	ModelID    string  `json:"model_id"`
	Priority   int     `json:"priority"`
	IsEnabled  bool    `json:"is_enabled"`
}

// SetModelRouting sets model routing for a feature
// @Summary Set model routing
// @Description Sets or updates model routing configuration for a feature
// @Tags AI
// @Accept json
// @Produce json
// @Param body body SetModelRoutingRequest true "Routing configuration"
// @Success 201 {object} ai.ModelRouting "Routing created"
// @Failure 400 {object} fiber.Map "Invalid request or feature type"
// @Failure 404 {object} fiber.Map "Provider not found"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /ai/routing [post]
func (h *AIHandler) SetModelRouting(c *fiber.Ctx) error {
	var req SetModelRoutingRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "invalid_request",
			"message": "Invalid request body",
		})
	}

	// Validate feature type
	feature := ai.FeatureType(req.Feature)
	if !feature.Valid() {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "invalid_request",
			"message": "Invalid feature type",
		})
	}

	providerID, err := uuid.Parse(req.ProviderID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "invalid_request",
			"message": "Invalid provider ID",
		})
	}

	routing := &ai.ModelRouting{
		Feature:    feature,
		ProviderID: providerID,
		ModelID:    req.ModelID,
		Priority:   req.Priority,
		IsEnabled:  req.IsEnabled,
	}

	// Parse optional server ID
	if req.ServerID != nil {
		serverID, err := uuid.Parse(*req.ServerID)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error":   "invalid_request",
				"message": "Invalid server ID",
			})
		}
		routing.ServerID = &serverID
	}

	// Parse optional user ID
	if req.UserID != nil {
		userID, err := uuid.Parse(*req.UserID)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error":   "invalid_request",
				"message": "Invalid user ID",
			})
		}
		routing.UserID = &userID
	}

	if err := h.aiService.SetModelRouting(c.Context(), routing); err != nil {
		if err == ai.ErrProviderNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error":   "not_found",
				"message": "Provider not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "internal_error",
			"message": err.Error(),
		})
	}

	return c.Status(fiber.StatusCreated).JSON(routing)
}

// DeleteModelRouting deletes a model routing configuration
// @Summary Delete model routing
// @Description Deletes a model routing configuration
// @Tags AI
// @Param id path string true "Routing ID"
// @Success 204 "Routing deleted"
// @Failure 400 {object} fiber.Map "Invalid routing ID"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /ai/routing/{id} [delete]
func (h *AIHandler) DeleteModelRouting(c *fiber.Ctx) error {
	idParam := c.Params("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "invalid_request",
			"message": "Invalid routing ID",
		})
	}

	if err := h.aiService.DeleteModelRouting(c.Context(), id); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "internal_error",
			"message": "Failed to delete routing",
		})
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// --- Models & Info ---

// GetAvailableModels returns all available models from enabled providers
// @Summary Get available models
// @Description Returns all available AI models from enabled providers
// @Tags AI
// @Produce json
// @Success 200 {object} fiber.Map{models=[]ai.AvailableModel} "List of available models"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /ai/models [get]
func (h *AIHandler) GetAvailableModels(c *fiber.Ctx) error {
	var userID *uuid.UUID
	if uid, ok := c.Locals("userID").(uuid.UUID); ok {
		userID = &uid
	}

	models, err := h.aiService.ListAvailableModels(c.Context(), userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "internal_error",
			"message": "Failed to retrieve models",
		})
	}

	return c.JSON(fiber.Map{
		"models": models,
	})
}

// GetProviderTypes returns all supported provider types
// @Summary Get provider types
// @Description Returns all supported AI provider types
// @Tags AI
// @Produce json
// @Success 200 {object} fiber.Map{provider_types=[]map[string]interface{}} "List of provider types"
// @Router /ai/provider-types [get]
func (h *AIHandler) GetProviderTypes(c *fiber.Ctx) error {
	info := h.aiService.GetProviderInfo()
	return c.JSON(fiber.Map{
		"provider_types": info,
	})
}

// GetFeatureTypes returns all supported feature types
// @Summary Get feature types
// @Description Returns all supported AI feature types with default models
// @Tags AI
// @Produce json
// @Success 200 {object} fiber.Map{feature_types=[]fiber.Map} "List of feature types"
// @Router /ai/feature-types [get]
func (h *AIHandler) GetFeatureTypes(c *fiber.Ctx) error {
	features := ai.AllFeatureTypes()
	defaults := ai.DefaultModels()

	featureInfo := make([]fiber.Map, len(features))
	for i, f := range features {
		featureInfo[i] = fiber.Map{
			"type":          f,
			"default_model": defaults[f],
		}
	}

	return c.JSON(fiber.Map{
		"feature_types": featureInfo,
	})
}

// HealthCheck checks the health of all enabled AI providers
// @Summary Check AI provider health
// @Description Checks the health status of all enabled AI providers
// @Tags AI
// @Produce json
// @Success 200 {object} fiber.Map{providers=map[string]interface{}} "Health status of providers"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /ai/health [get]
func (h *AIHandler) HealthCheck(c *fiber.Ctx) error {
	status, err := h.aiService.HealthCheck(c.Context())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "internal_error",
			"message": "Failed to check provider health",
		})
	}

	return c.JSON(fiber.Map{
		"providers": status,
	})
}

// --- Chat API ---

// ChatRequest is the request body for chat completions
type ChatCompletionRequest struct {
	Model       string              `json:"model,omitempty"`
	Feature     string              `json:"feature,omitempty"` // Use routing if specified
	Messages    []providers.Message `json:"messages"`
	MaxTokens   int                 `json:"max_tokens,omitempty"`
	Temperature *float64            `json:"temperature,omitempty"`
	Stream      bool                `json:"stream,omitempty"`
	ServerID    *string             `json:"server_id,omitempty"`
}

// ChatCompletion performs a chat completion
// @Summary Generate chat completion
// @Description Generates a chat completion using the configured AI provider
// @Tags AI
// @Accept json
// @Produce json
// @Param body body ChatCompletionRequest true "Chat completion request"
// @Success 200 {object} providers.ChatResponse "Chat completion response"
// @Failure 400 {object} fiber.Map "Invalid request"
// @Failure 401 {object} fiber.Map "Unauthorized"
// @Failure 500 {object} fiber.Map "AI error"
// @Router /ai/completion [post]
func (h *AIHandler) ChatCompletion(c *fiber.Ctx) error {
	userID, ok := c.Locals("userID").(uuid.UUID)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error":   "unauthorized",
			"message": "User ID not found in context",
		})
	}

	var req ChatCompletionRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "invalid_request",
			"message": "Invalid request body",
		})
	}

	if len(req.Messages) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "invalid_request",
			"message": "Messages are required",
		})
	}

	// Determine feature type
	feature := ai.FeatureChat
	if req.Feature != "" {
		f := ai.FeatureType(req.Feature)
		if f.Valid() {
			feature = f
		}
	}

	// Parse optional server ID
	var serverID *uuid.UUID
	if req.ServerID != nil {
		sid, err := uuid.Parse(*req.ServerID)
		if err == nil {
			serverID = &sid
		}
	}

	chatReq := &providers.ChatRequest{
		Model:       req.Model,
		Messages:    req.Messages,
		MaxTokens:   req.MaxTokens,
		Temperature: req.Temperature,
		Stream:      req.Stream,
	}

	// Handle streaming request
	if req.Stream {
		return h.handleChatStream(c, feature, chatReq, serverID, &userID)
	}

	resp, err := h.aiService.Chat(c.Context(), feature, chatReq, serverID, &userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "ai_error",
			"message": err.Error(),
		})
	}

	return c.JSON(resp)
}

// handleChatStream handles SSE streaming for chat completions
func (h *AIHandler) handleChatStream(c *fiber.Ctx, feature ai.FeatureType, req *providers.ChatRequest, serverID, userID *uuid.UUID) error {
	c.Set("Content-Type", "text/event-stream")
	c.Set("Cache-Control", "no-cache")
	c.Set("Connection", "keep-alive")
	c.Set("Transfer-Encoding", "chunked")
	c.Set("X-Accel-Buffering", "no") // Disable nginx buffering

	c.Context().SetBodyStreamWriter(func(w *bufio.Writer) {
		// Send initial connection event
		fmt.Fprintf(w, "event: connected\ndata: {\"status\":\"connected\"}\n\n")
		w.Flush()

		// Callback for each streaming chunk
		callback := func(chunk *providers.ChatResponse) error {
			data, err := json.Marshal(chunk)
			if err != nil {
				return err
			}
			fmt.Fprintf(w, "data: %s\n\n", string(data))
			return w.Flush()
		}

		// Perform streaming chat
		err := h.aiService.ChatStream(c.Context(), feature, req, callback, serverID, userID)
		if err != nil {
			errData, _ := json.Marshal(fiber.Map{
				"error":   "stream_error",
				"message": err.Error(),
			})
			fmt.Fprintf(w, "event: error\ndata: %s\n\n", string(errData))
			w.Flush()
			return
		}

		// Send completion event
		fmt.Fprintf(w, "event: done\ndata: {\"done\":true}\n\n")
		w.Flush()
	})

	return nil
}

// EmbeddingRequest is the request body for generating embeddings
type EmbeddingRequestBody struct {
	Model    string   `json:"model,omitempty"`
	Input    []string `json:"input"`
	ServerID *string  `json:"server_id,omitempty"`
}

// GenerateEmbeddings generates embeddings for the given input
// @Summary Generate embeddings
// @Description Generates embeddings for the provided text input
// @Tags AI
// @Accept json
// @Produce json
// @Param body body EmbeddingRequestBody true "Embedding request"
// @Success 200 {object} providers.EmbeddingResponse "Embedding response"
// @Failure 400 {object} fiber.Map "Invalid request"
// @Failure 401 {object} fiber.Map "Unauthorized"
// @Failure 500 {object} fiber.Map "AI error"
// @Router /ai/embeddings [post]
func (h *AIHandler) GenerateEmbeddings(c *fiber.Ctx) error {
	userID, ok := c.Locals("userID").(uuid.UUID)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error":   "unauthorized",
			"message": "User ID not found in context",
		})
	}

	var req EmbeddingRequestBody
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "invalid_request",
			"message": "Invalid request body",
		})
	}

	if len(req.Input) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "invalid_request",
			"message": "Input is required",
		})
	}

	// Parse optional server ID
	var serverID *uuid.UUID
	if req.ServerID != nil {
		sid, err := uuid.Parse(*req.ServerID)
		if err == nil {
			serverID = &sid
		}
	}

	embedReq := &providers.EmbeddingRequest{
		Model: req.Model,
		Input: req.Input,
	}

	resp, err := h.aiService.Embed(c.Context(), embedReq, serverID, &userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "ai_error",
			"message": err.Error(),
		})
	}

	return c.JSON(resp)
}

// --- User AI Settings ---

// UserAISettingsResponse represents user's AI settings
type UserAISettingsResponse struct {
	DefaultProvider *uuid.UUID                     `json:"default_provider,omitempty"`
	DefaultModel    string                         `json:"default_model,omitempty"`
	Credentials     []*ai.UserAICredentialResponse `json:"credentials"`
	Routings        []*ai.ModelRouting             `json:"routings,omitempty"`
}

// GetUserSettings returns the user's AI settings
// @Summary Get user AI settings
// @Description Returns the current user's AI settings including credentials and routings
// @Tags AI
// @Produce json
// @Success 200 {object} UserAISettingsResponse "User AI settings"
// @Failure 401 {object} fiber.Map "Unauthorized"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /ai/settings [get]
func (h *AIHandler) GetUserSettings(c *fiber.Ctx) error {
	userID, ok := c.Locals("userID").(uuid.UUID)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error":   "unauthorized",
			"message": "User ID not found in context",
		})
	}

	// Get user credentials
	creds, err := h.aiService.GetUserCredentials(c.Context(), userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "internal_error",
			"message": "Failed to retrieve credentials",
		})
	}

	// Get user-specific routings
	routings, err := h.aiService.GetAllModelRoutings(c.Context())
	if err != nil {
		routings = nil // Non-fatal, continue without routings
	}

	// Filter to user's routings
	var userRoutings []*ai.ModelRouting
	for _, r := range routings {
		if r.UserID != nil && *r.UserID == userID {
			userRoutings = append(userRoutings, r)
		}
	}

	response := UserAISettingsResponse{
		Credentials: creds,
		Routings:    userRoutings,
	}

	return c.JSON(response)
}

// UpdateUserSettingsRequest is the request body for updating user settings
type UpdateUserSettingsRequest struct {
	DefaultProvider *string `json:"default_provider,omitempty"`
	DefaultModel    string  `json:"default_model,omitempty"`

	// Optional: Update credentials for a provider
	ProviderCredentials *SetUserCredentialsRequest `json:"provider_credentials,omitempty"`

	// Optional: Update feature routing
	FeatureRouting *SetModelRoutingRequest `json:"feature_routing,omitempty"`
}

// updateUserCredentials handles updating user credentials from the settings request
func (h *AIHandler) updateUserCredentials(c *fiber.Ctx, userID uuid.UUID, req *SetUserCredentialsRequest) error {
	providerID, err := uuid.Parse(req.ProviderID)
	if err != nil {
		return h.respondBadRequest(c, "Invalid provider ID")
	}

	credentials := &providers.ProviderConfig{
		APIKey:         req.APIKey,
		SecretKey:      req.SecretKey,
		Region:         req.Region,
		ProjectID:      req.ProjectID,
		ServiceAccount: req.ServiceAccount,
	}

	if err := h.aiService.SetUserCredentials(c.Context(), userID, providerID, credentials); err != nil {
		return h.respondInternalError(c, err.Error())
	}

	return nil
}

// updateFeatureRouting handles updating feature routing from the settings request
func (h *AIHandler) updateFeatureRouting(c *fiber.Ctx, userID uuid.UUID, req *SetModelRoutingRequest) error {
	feature := ai.FeatureType(req.Feature)
	if !feature.Valid() {
		return h.respondBadRequest(c, "Invalid feature type")
	}

	providerID, err := uuid.Parse(req.ProviderID)
	if err != nil {
		return h.respondBadRequest(c, "Invalid provider ID")
	}

	routing := &ai.ModelRouting{
		UserID:     &userID,
		Feature:    feature,
		ProviderID: providerID,
		ModelID:    req.ModelID,
		Priority:   req.Priority,
		IsEnabled:  req.IsEnabled,
	}

	if err := h.aiService.SetModelRouting(c.Context(), routing); err != nil {
		return h.respondInternalError(c, err.Error())
	}

	return nil
}

// UpdateUserSettings updates the user's AI settings
// @Summary Update user AI settings
// @Description Updates the current user's AI settings
// @Tags AI
// @Accept json
// @Produce json
// @Param body body UpdateUserSettingsRequest true "Settings update data"
// @Success 200 {object} fiber.Map "Settings updated successfully"
// @Failure 400 {object} fiber.Map "Invalid request"
// @Failure 401 {object} fiber.Map "Unauthorized"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /ai/settings [put]
func (h *AIHandler) UpdateUserSettings(c *fiber.Ctx) error {
	userID, err := h.getUserID(c)
	if err != nil {
		return h.respondUnauthorized(c, err.Error())
	}

	var req UpdateUserSettingsRequest
	if err := c.BodyParser(&req); err != nil {
		return h.respondBadRequest(c, "Invalid request body")
	}

	// Update credentials if provided
	if req.ProviderCredentials != nil {
		if err := h.updateUserCredentials(c, userID, req.ProviderCredentials); err != nil {
			return err
		}
	}

	// Update feature routing if provided
	if req.FeatureRouting != nil {
		if err := h.updateFeatureRouting(c, userID, req.FeatureRouting); err != nil {
			return err
		}
	}

	return c.JSON(fiber.Map{
		"message": "Settings updated successfully",
	})
}

// --- Admin AI Defaults ---

// AdminAIDefaultsRequest is the request body for setting admin defaults
type AdminAIDefaultsRequest struct {
	DefaultProvider   string                          `json:"default_provider"`
	AllowUserOverride bool                            `json:"allow_user_override"`
	FeatureDefaults   map[string]FeatureDefaultConfig `json:"feature_defaults,omitempty"`
}

// FeatureDefaultConfig is the config for a specific feature
type FeatureDefaultConfig struct {
	ProviderID string `json:"provider_id"`
	ModelID    string `json:"model_id"`
	Enabled    bool   `json:"enabled"`
}

// SetAdminDefaults sets server-wide AI defaults (admin only)
// @Summary Set admin AI defaults
// @Description Sets server-wide AI defaults (requires admin access)
// @Tags AI Admin
// @Accept json
// @Produce json
// @Param body body AdminAIDefaultsRequest true "Admin defaults configuration"
// @Success 200 {object} fiber.Map "Admin defaults updated"
// @Failure 400 {object} fiber.Map "Invalid request"
// @Failure 403 {object} fiber.Map "Admin access required"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /ai/admin/defaults [post]
func (h *AIHandler) SetAdminDefaults(c *fiber.Ctx) error {
	// Note: This should be protected by admin middleware
	// For now, we check if the user has admin role via locals
	isAdmin, _ := c.Locals("isAdmin").(bool)
	if !isAdmin {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error":   "forbidden",
			"message": "Admin access required",
		})
	}

	var req AdminAIDefaultsRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "invalid_request",
			"message": "Invalid request body",
		})
	}

	adminConfig := &ai.AdminAIConfig{
		DefaultProvider:   req.DefaultProvider,
		AllowUserOverride: req.AllowUserOverride,
		FeatureDefaults:   make(map[ai.FeatureType]ai.FeatureConfig),
	}

	// Convert feature defaults
	for featureStr, cfg := range req.FeatureDefaults {
		feature := ai.FeatureType(featureStr)
		if !feature.Valid() {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error":   "invalid_request",
				"message": "Invalid feature type: " + featureStr,
			})
		}

		providerID, err := uuid.Parse(cfg.ProviderID)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error":   "invalid_request",
				"message": "Invalid provider ID for feature: " + featureStr,
			})
		}

		// Validate provider type for the feature config
		providerConfig, err := h.aiService.GetProvider(c.Context(), providerID)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error":   "invalid_request",
				"message": "Provider not found for feature: " + featureStr,
			})
		}

		adminConfig.FeatureDefaults[feature] = ai.FeatureConfig{
			ProviderType: providerConfig.ProviderType,
			ProviderID:   providerID,
			ModelID:      cfg.ModelID,
			Enabled:      cfg.Enabled,
		}
	}

	h.aiService.SetAdminConfig(adminConfig)

	return c.JSON(fiber.Map{
		"message": "Admin defaults updated successfully",
		"config":  adminConfig,
	})
}

// GetAdminDefaults returns the current admin AI defaults
// @Summary Get admin AI defaults
// @Description Returns the current server-wide AI defaults (requires admin access)
// @Tags AI Admin
// @Produce json
// @Success 200 {object} ai.AdminAIConfig "Admin AI configuration"
// @Failure 403 {object} fiber.Map "Admin access required"
// @Router /ai/admin/defaults [get]
func (h *AIHandler) GetAdminDefaults(c *fiber.Ctx) error {
	// Note: This should be protected by admin middleware
	isAdmin, _ := c.Locals("isAdmin").(bool)
	if !isAdmin {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error":   "forbidden",
			"message": "Admin access required",
		})
	}

	config := h.aiService.GetAdminConfig()
	return c.JSON(config)
}
