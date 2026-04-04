package handlers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"hearth/internal/models"
	"hearth/internal/services"
)

// ServerFolderHandler handles server folder HTTP requests
type ServerFolderHandler struct {
	folderService *services.ServerFolderService
	serverService *services.ServerService
}

func NewServerFolderHandler(
	folderService *services.ServerFolderService,
	serverService *services.ServerService,
) *ServerFolderHandler {
	return &ServerFolderHandler{
		folderService: folderService,
		serverService: serverService,
	}
}

// Create creates a new server folder
// @Summary Create a server folder
// @Description Creates a new folder for organizing servers
// @Tags Server Folders
// @Accept json
// @Produce json
// @Param body body models.CreateServerFolderRequest true "Folder creation data"
// @Success 201 {object} models.ServerFolder "Folder created successfully"
// @Failure 400 {object} fiber.Map "Invalid request body"
// @Failure 401 {object} fiber.Map "Unauthorized"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /api/v1/users/me/server-folders [post]
func (h *ServerFolderHandler) Create(c *fiber.Ctx) error {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return Unauthorized(c, "unauthorized")
	}

	var req models.CreateServerFolderRequest
	if err := c.BodyParser(&req); err != nil {
		return ParseError(c, err)
	}

	if req.Name == "" || len(req.Name) < 1 || len(req.Name) > 100 {
		return ValidationError(c, "name", "must be between 1 and 100 characters")
	}

	folder, err := h.folderService.CreateFolder(c.Context(), userID, &req)
	if err != nil {
		return HandleServiceError(c, err)
	}

	return c.Status(fiber.StatusCreated).JSON(folder)
}

// GetAll gets all server folders for the authenticated user
// @Summary Get all server folders
// @Description Gets all folders and unassigned servers for the current user
// @Tags Server Folders
// @Produce json
// @Success 200 {object} models.ServerFolderTree "Folders retrieved successfully"
// @Failure 401 {object} fiber.Map "Unauthorized"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /api/v1/users/me/server-folders [get]
func (h *ServerFolderHandler) GetAll(c *fiber.Ctx) error {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return Unauthorized(c, "unauthorized")
	}

	tree, err := h.folderService.GetUserFolders(c.Context(), userID)
	if err != nil {
		return HandleServiceError(c, err)
	}

	return c.JSON(tree)
}

// Get gets a server folder by ID
// @Summary Get a server folder
// @Description Gets a specific folder by ID
// @Tags Server Folders
// @Produce json
// @Param id path string true "Folder ID"
// @Success 200 {object} models.ServerFolder "Folder retrieved successfully"
// @Failure 400 {object} fiber.Map "Invalid folder ID"
// @Failure 401 {object} fiber.Map "Unauthorized"
// @Failure 404 {object} fiber.Map "Folder not found"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /api/v1/users/me/server-folders/{id} [get]
func (h *ServerFolderHandler) Get(c *fiber.Ctx) error {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return Unauthorized(c, "unauthorized")
	}

	folderID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return InvalidUUID(c, "folder id")
	}

	folder, err := h.folderService.GetFolder(c.Context(), userID, folderID)
	if err != nil {
		return HandleServiceError(c, err)
	}

	return c.JSON(folder)
}

// Update updates a server folder
// @Summary Update a server folder
// @Description Updates folder name, position, collapsed state, or parent
// @Tags Server Folders
// @Accept json
// @Produce json
// @Param id path string true "Folder ID"
// @Param body body models.UpdateServerFolderRequest true "Folder update data"
// @Success 200 {object} models.ServerFolder "Folder updated successfully"
// @Failure 400 {object} fiber.Map "Invalid request body or folder ID"
// @Failure 401 {object} fiber.Map "Unauthorized"
// @Failure 404 {object} fiber.Map "Folder not found"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /api/v1/users/me/server-folders/{id} [patch]
func (h *ServerFolderHandler) Update(c *fiber.Ctx) error {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return Unauthorized(c, "unauthorized")
	}

	folderID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return InvalidUUID(c, "folder id")
	}

	var req models.UpdateServerFolderRequest
	if err := c.BodyParser(&req); err != nil {
		return ParseError(c, err)
	}

	folder, err := h.folderService.UpdateFolder(c.Context(), userID, folderID, &req)
	if err != nil {
		return HandleServiceError(c, err)
	}

	return c.JSON(folder)
}

// Delete deletes a server folder
// @Summary Delete a server folder
// @Description Deletes a folder and moves servers to unassigned
// @Tags Server Folders
// @Produce json
// @Param id path string true "Folder ID"
// @Success 204 "Folder deleted successfully"
// @Failure 400 {object} fiber.Map "Invalid folder ID"
// @Failure 401 {object} fiber.Map "Unauthorized"
// @Failure 404 {object} fiber.Map "Folder not found"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /api/v1/users/me/server-folders/{id} [delete]
func (h *ServerFolderHandler) Delete(c *fiber.Ctx) error {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return Unauthorized(c, "unauthorized")
	}

	folderID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return InvalidUUID(c, "folder id")
	}

	if err := h.folderService.DeleteFolder(c.Context(), userID, folderID); err != nil {
		return HandleServiceError(c, err)
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// MoveServer moves a single server to a folder
// @Summary Move server to folder
// @Description Moves a server to a specified folder or unassigns it
// @Tags Server Folders
// @Accept json
// @Produce json
// @Param body body struct{ServerID string `json:"server_id"`; FolderID *string `json:"folder_id,omitempty"`} true "Move data"
// @Success 200 {object} fiber.Map "Server moved successfully"
// @Failure 400 {object} fiber.Map "Invalid request body"
// @Failure 401 {object} fiber.Map "Unauthorized"
// @Failure 404 {object} fiber.Map "Folder or server not found"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /api/v1/users/me/server-folders/move [post]
func (h *ServerFolderHandler) MoveServer(c *fiber.Ctx) error {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return Unauthorized(c, "unauthorized")
	}

	var req struct {
		ServerID string   `json:"server_id"`
		FolderID *string `json:"folder_id,omitempty"`
	}

	if err := c.BodyParser(&req); err != nil {
		return ParseError(c, err)
	}

	serverID, err := uuid.Parse(req.ServerID)
	if err != nil {
		return InvalidUUID(c, "server_id")
	}

	var folderID *uuid.UUID
	if req.FolderID != nil {
		fid, err := uuid.Parse(*req.FolderID)
		if err != nil {
			return InvalidUUID(c, "folder_id")
		}
		folderID = &fid
	}

	if err := h.folderService.MoveServerToFolder(c.Context(), userID, serverID, folderID); err != nil {
		return HandleServiceError(c, err)
	}

	return c.JSON(fiber.Map{"success": true})
}

// MoveServers moves multiple servers to a folder
// @Summary Move multiple servers to folder
// @Description Moves multiple servers to a specified folder or unassigns them
// @Tags Server Folders
// @Accept json
// @Produce json
// @Param body body models.MoveServersToFolderRequest true "Move data"
// @Success 200 {object} fiber.Map "Servers moved successfully"
// @Failure 400 {object} fiber.Map "Invalid request body"
// @Failure 401 {object} fiber.Map "Unauthorized"
// @Failure 404 {object} fiber.Map "Folder not found"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /api/v1/users/me/server-folders/move-batch [post]
func (h *ServerFolderHandler) MoveServers(c *fiber.Ctx) error {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return Unauthorized(c, "unauthorized")
	}

	var req models.MoveServersToFolderRequest
	if err := c.BodyParser(&req); err != nil {
		return ParseError(c, err)
	}

	if len(req.ServerIDs) == 0 {
		return ValidationError(c, "server_ids", "must contain at least one server")
	}

	if err := h.folderService.MoveServersToFolder(c.Context(), userID, &req); err != nil {
		return HandleServiceError(c, err)
	}

	return c.JSON(fiber.Map{"success": true})
}

// ReorderServers reorders servers within a folder
// @Summary Reorder servers
// @Description Reorders servers within a folder or at root level
// @Tags Server Folders
// @Accept json
// @Produce json
// @Param body body struct{FolderID *string `json:"folder_id,omitempty"`; ServerPositions []models.ServerPosition `json:"server_positions"`} true "Reorder data"
// @Success 200 {object} fiber.Map "Servers reordered successfully"
// @Failure 400 {object} fiber.Map "Invalid request body"
// @Failure 401 {object} fiber.Map "Unauthorized"
// @Failure 404 {object} fiber.Map "Folder not found"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /api/v1/users/me/server-folders/reorder [post]
func (h *ServerFolderHandler) ReorderServers(c *fiber.Ctx) error {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return Unauthorized(c, "unauthorized")
	}

	var req struct {
		FolderID       *string                  `json:"folder_id,omitempty"`
		ServerPositions []models.ServerPosition `json:"server_positions"`
	}

	if err := c.BodyParser(&req); err != nil {
		return ParseError(c, err)
	}

	if len(req.ServerPositions) == 0 {
		return ValidationError(c, "server_positions", "must contain at least one server")
	}

	var folderID *uuid.UUID
	if req.FolderID != nil {
		fid, err := uuid.Parse(*req.FolderID)
		if err != nil {
			return InvalidUUID(c, "folder_id")
		}
		folderID = &fid
	}

	reorderReq := &models.ReorderServersRequest{
		ServerPositions: req.ServerPositions,
	}

	if err := h.folderService.ReorderServers(c.Context(), userID, folderID, reorderReq); err != nil {
		return HandleServiceError(c, err)
	}

	return c.JSON(fiber.Map{"success": true})
}
