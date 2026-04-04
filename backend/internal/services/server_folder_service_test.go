package services

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"hearth/internal/models"
)

// MockServerFolderRepository is a mock implementation of ServerFolderRepository
type MockServerFolderRepository struct {
	mock.Mock
}

func (m *MockServerFolderRepository) Create(ctx context.Context, folder *models.ServerFolder) error {
	args := m.Called(ctx, folder)
	return args.Error(0)
}

func (m *MockServerFolderRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.ServerFolder, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.ServerFolder), args.Error(1)
}

func (m *MockServerFolderRepository) GetByUserID(ctx context.Context, userID uuid.UUID) ([]*models.ServerFolder, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.ServerFolder), args.Error(1)
}

func (m *MockServerFolderRepository) Update(ctx context.Context, folder *models.ServerFolder) error {
	args := m.Called(ctx, folder)
	return args.Error(0)
}

func (m *MockServerFolderRepository) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockServerFolderRepository) GetChildFolders(ctx context.Context, parentID uuid.UUID) ([]*models.ServerFolder, error) {
	args := m.Called(ctx, parentID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.ServerFolder), args.Error(1)
}

func (m *MockServerFolderRepository) GetMaxPositionAtLevel(ctx context.Context, userID uuid.UUID, depth int, parentID *uuid.UUID) (int, error) {
	args := m.Called(ctx, userID, depth, parentID)
	return args.Int(0), args.Error(1)
}

func (m *MockServerFolderRepository) AssignServerToFolder(ctx context.Context, userID, serverID uuid.UUID, folderID *uuid.UUID, position int) error {
	args := m.Called(ctx, userID, serverID, folderID, position)
	return args.Error(0)
}

func (m *MockServerFolderRepository) RemoveServerFromFolder(ctx context.Context, userID, serverID uuid.UUID) error {
	args := m.Called(ctx, userID, serverID)
	return args.Error(0)
}

func (m *MockServerFolderRepository) GetServerFolder(ctx context.Context, userID, serverID uuid.UUID) (*models.ServerInFolder, error) {
	args := m.Called(ctx, userID, serverID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.ServerInFolder), args.Error(1)
}

func (m *MockServerFolderRepository) GetServersInFolder(ctx context.Context, userID uuid.UUID, folderID *uuid.UUID) ([]*models.ServerInFolder, error) {
	args := m.Called(ctx, userID, folderID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.ServerInFolder), args.Error(1)
}

func (m *MockServerFolderRepository) GetAllUserServersWithFolders(ctx context.Context, userID uuid.UUID) ([]*models.ServerInFolder, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.ServerInFolder), args.Error(1)
}

func (m *MockServerFolderRepository) UpdateServerPositions(ctx context.Context, userID uuid.UUID, positions []models.ServerPosition) error {
	args := m.Called(ctx, userID, positions)
	return args.Error(0)
}

func (m *MockServerFolderRepository) GetChildFolderIDs(ctx context.Context, parentID uuid.UUID) ([]uuid.UUID, error) {
	args := m.Called(ctx, parentID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]uuid.UUID), args.Error(1)
}

// MockServerRepositoryForFolderTests is a mock for ServerRepository (limited interface for folder tests)
type MockServerRepositoryForFolderTests struct {
	mock.Mock
}

func (m *MockServerRepositoryForFolderTests) Create(ctx context.Context, server *models.Server) error {
	args := m.Called(ctx, server)
	return args.Error(0)
}

func (m *MockServerRepositoryForFolderTests) GetByID(ctx context.Context, id uuid.UUID) (*models.Server, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Server), args.Error(1)
}

func (m *MockServerRepositoryForFolderTests) Update(ctx context.Context, server *models.Server) error {
	args := m.Called(ctx, server)
	return args.Error(0)
}

func (m *MockServerRepositoryForFolderTests) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockServerRepositoryForFolderTests) TransferOwnership(ctx context.Context, serverID, newOwnerID uuid.UUID) error {
	args := m.Called(ctx, serverID, newOwnerID)
	return args.Error(0)
}

func (m *MockServerRepositoryForFolderTests) GetMembers(ctx context.Context, serverID uuid.UUID, limit, offset int) ([]*models.Member, error) {
	args := m.Called(ctx, serverID, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Member), args.Error(1)
}

func (m *MockServerRepositoryForFolderTests) GetMembersPaginated(ctx context.Context, serverID uuid.UUID, cursor *models.MemberCursor, limit int) (*models.PaginatedMembers, error) {
	args := m.Called(ctx, serverID, cursor, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.PaginatedMembers), args.Error(1)
}

func (m *MockServerRepositoryForFolderTests) GetMember(ctx context.Context, serverID, userID uuid.UUID) (*models.Member, error) {
	args := m.Called(ctx, serverID, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Member), args.Error(1)
}

func (m *MockServerRepositoryForFolderTests) GetMembersWithRole(ctx context.Context, serverID, roleID uuid.UUID) ([]*models.Member, error) {
	args := m.Called(ctx, serverID, roleID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Member), args.Error(1)
}

func (m *MockServerRepositoryForFolderTests) AddMember(ctx context.Context, member *models.Member) error {
	args := m.Called(ctx, member)
	return args.Error(0)
}

func (m *MockServerRepositoryForFolderTests) UpdateMember(ctx context.Context, member *models.Member) error {
	args := m.Called(ctx, member)
	return args.Error(0)
}

func (m *MockServerRepositoryForFolderTests) RemoveMember(ctx context.Context, serverID, userID uuid.UUID) error {
	args := m.Called(ctx, serverID, userID)
	return args.Error(0)
}

func (m *MockServerRepositoryForFolderTests) GetMemberCount(ctx context.Context, serverID uuid.UUID) (int, error) {
	args := m.Called(ctx, serverID)
	return args.Int(0), args.Error(1)
}

func (m *MockServerRepositoryForFolderTests) GetUserServers(ctx context.Context, userID uuid.UUID) ([]*models.Server, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Server), args.Error(1)
}

func (m *MockServerRepositoryForFolderTests) GetOwnedServersCount(ctx context.Context, userID uuid.UUID) (int, error) {
	args := m.Called(ctx, userID)
	return args.Int(0), args.Error(1)
}

func (m *MockServerRepositoryForFolderTests) GetBan(ctx context.Context, serverID, userID uuid.UUID) (*models.Ban, error) {
	args := m.Called(ctx, serverID, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Ban), args.Error(1)
}

func (m *MockServerRepositoryForFolderTests) AddBan(ctx context.Context, ban *models.Ban) error {
	args := m.Called(ctx, ban)
	return args.Error(0)
}

func (m *MockServerRepositoryForFolderTests) RemoveBan(ctx context.Context, serverID, userID uuid.UUID) error {
	args := m.Called(ctx, serverID, userID)
	return args.Error(0)
}

func (m *MockServerRepositoryForFolderTests) GetBans(ctx context.Context, serverID uuid.UUID) ([]*models.Ban, error) {
	args := m.Called(ctx, serverID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Ban), args.Error(1)
}

func (m *MockServerRepositoryForFolderTests) CreateInvite(ctx context.Context, invite *models.Invite) error {
	args := m.Called(ctx, invite)
	return args.Error(0)
}

func (m *MockServerRepositoryForFolderTests) GetInvite(ctx context.Context, code string) (*models.Invite, error) {
	args := m.Called(ctx, code)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Invite), args.Error(1)
}

func (m *MockServerRepositoryForFolderTests) GetInviteByVanityCode(ctx context.Context, vanityCode string) (*models.Invite, error) {
	args := m.Called(ctx, vanityCode)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Invite), args.Error(1)
}

func (m *MockServerRepositoryForFolderTests) GetInvites(ctx context.Context, serverID uuid.UUID) ([]*models.Invite, error) {
	args := m.Called(ctx, serverID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Invite), args.Error(1)
}

func (m *MockServerRepositoryForFolderTests) DeleteInvite(ctx context.Context, code string) error {
	args := m.Called(ctx, code)
	return args.Error(0)
}

func (m *MockServerRepositoryForFolderTests) IncrementInviteUses(ctx context.Context, code string) error {
	args := m.Called(ctx, code)
	return args.Error(0)
}

func (m *MockServerRepositoryForFolderTests) LogInviteUse(ctx context.Context, log *models.InviteUseLog) error {
	args := m.Called(ctx, log)
	return args.Error(0)
}

func (m *MockServerRepositoryForFolderTests) GetInviteUseLogs(ctx context.Context, inviteCode string) ([]models.InviteUseLog, error) {
	args := m.Called(ctx, inviteCode)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.InviteUseLog), args.Error(1)
}

func (m *MockServerRepositoryForFolderTests) GetServerInviteUseLogs(ctx context.Context, serverID uuid.UUID) ([]models.InviteUseLog, error) {
	args := m.Called(ctx, serverID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.InviteUseLog), args.Error(1)
}

func TestServerFolderService_CreateFolder(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()
	parentID := uuid.New()

	t.Run("create root folder successfully", func(t *testing.T) {
		repo := new(MockServerFolderRepository)
		eventBus := new(MockEventBus)

		repo.On("GetMaxPositionAtLevel", ctx, userID, 0, (*uuid.UUID)(nil)).Return(0, nil)
		repo.On("Create", ctx, mock.AnythingOfType("*models.ServerFolder")).Return(nil)
		eventBus.On("Publish", "server_folder_created", mock.AnythingOfType("map[string]interface {}"))

		svc := NewServerFolderService(repo, nil, eventBus)
		req := &models.CreateServerFolderRequest{
			Name: "My Folder",
		}

		folder, err := svc.CreateFolder(ctx, userID, req)

		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if folder == nil {
			t.Fatal("expected folder, got nil")
		}
		if folder.Name != req.Name {
			t.Errorf("expected folder name %s, got %s", req.Name, folder.Name)
		}
		if folder.Depth != 0 {
			t.Errorf("expected depth 0, got %d", folder.Depth)
		}

		repo.AssertExpectations(t)
		eventBus.AssertExpectations(t)
	})

	t.Run("create nested folder successfully", func(t *testing.T) {
		repo := new(MockServerFolderRepository)
		eventBus := new(MockEventBus)

		repo.On("GetByID", ctx, parentID).Return(&models.ServerFolder{
			ID:     parentID,
			UserID: userID,
			Depth:  0,
		}, nil)
		repo.On("GetMaxPositionAtLevel", ctx, userID, 1, &parentID).Return(0, nil)
		repo.On("Create", ctx, mock.AnythingOfType("*models.ServerFolder")).Return(nil)
		eventBus.On("Publish", "server_folder_created", mock.AnythingOfType("map[string]interface {}"))

		svc := NewServerFolderService(repo, nil, eventBus)
		req := &models.CreateServerFolderRequest{
			Name:     "Nested Folder",
			ParentID: &parentID,
		}

		folder, err := svc.CreateFolder(ctx, userID, req)

		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if folder == nil {
			t.Fatal("expected folder, got nil")
		}
		if folder.Name != req.Name {
			t.Errorf("expected folder name %s, got %s", req.Name, folder.Name)
		}
		if folder.Depth != 1 {
			t.Errorf("expected depth 1, got %d", folder.Depth)
		}

		repo.AssertExpectations(t)
		eventBus.AssertExpectations(t)
	})

	t.Run("parent folder not found", func(t *testing.T) {
		repo := new(MockServerFolderRepository)

		repo.On("GetByID", ctx, parentID).Return(nil, sql.ErrNoRows)

		svc := NewServerFolderService(repo, nil, nil)
		req := &models.CreateServerFolderRequest{
			Name:     "Nested Folder",
			ParentID: &parentID,
		}

		_, err := svc.CreateFolder(ctx, userID, req)

		if err == nil {
			t.Error("expected error, got nil")
		}
		if err != ErrInvalidParentFolder {
			t.Errorf("expected ErrInvalidParentFolder, got %v", err)
		}

		repo.AssertExpectations(t)
	})

	t.Run("max nesting exceeded", func(t *testing.T) {
		repo := new(MockServerFolderRepository)

		repo.On("GetByID", ctx, parentID).Return(&models.ServerFolder{
			ID:     parentID,
			UserID: userID,
			Depth:  models.MaxNestingDepth, // 2
		}, nil)

		svc := NewServerFolderService(repo, nil, nil)
		req := &models.CreateServerFolderRequest{
			Name:     "Nested Folder",
			ParentID: &parentID,
		}

		_, err := svc.CreateFolder(ctx, userID, req)

		if err == nil {
			t.Error("expected error, got nil")
		}
		if err != ErrMaxNestingExceeded {
			t.Errorf("expected ErrMaxNestingExceeded, got %v", err)
		}

		repo.AssertExpectations(t)
	})

	t.Run("parent belongs to different user", func(t *testing.T) {
		repo := new(MockServerFolderRepository)

		repo.On("GetByID", ctx, parentID).Return(&models.ServerFolder{
			ID:     parentID,
			UserID: uuid.New(), // Different user
			Depth:  0,
		}, nil)

		svc := NewServerFolderService(repo, nil, nil)
		req := &models.CreateServerFolderRequest{
			Name:     "Nested Folder",
			ParentID: &parentID,
		}

		_, err := svc.CreateFolder(ctx, userID, req)

		if err == nil {
			t.Error("expected error, got nil")
		}
		if err != ErrInvalidParentFolder {
			t.Errorf("expected ErrInvalidParentFolder, got %v", err)
		}

		repo.AssertExpectations(t)
	})
}

func TestServerFolderService_GetUserFolders(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()
	folderID1 := uuid.New()
	folderID2 := uuid.New()
	serverID := uuid.New()

	t.Run("get folders with servers", func(t *testing.T) {
		repo := new(MockServerFolderRepository)
		eventBus := new(MockEventBus)

		repo.On("GetByUserID", ctx, userID).Return([]*models.ServerFolder{
			{
				ID:       folderID1,
				UserID:   userID,
				Name:     "Folder 1",
				Position: 0,
				Depth:    0,
			},
			{
				ID:       folderID2,
				UserID:   userID,
				ParentID: &folderID1,
				Name:     "Folder 2",
				Position: 0,
				Depth:    1,
			},
		}, nil)

		repo.On("GetAllUserServersWithFolders", ctx, userID).Return([]*models.ServerInFolder{
			{
				ServerID: serverID,
				FolderID: &folderID1,
				Position: 0,
				Server: &models.Server{
					ID:   serverID,
					Name: "Test Server",
				},
			},
		}, nil)

		svc := NewServerFolderService(repo, nil, eventBus)
		tree, err := svc.GetUserFolders(ctx, userID)

		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if tree == nil {
			t.Fatal("expected tree, got nil")
		}
		if len(tree.Folders) != 1 {
			t.Errorf("expected 1 root folder, got %d", len(tree.Folders))
		}
		if len(tree.Folders[0].Children) != 1 {
			t.Errorf("expected 1 child folder, got %d", len(tree.Folders[0].Children))
		}

		repo.AssertExpectations(t)
	})
}

func TestServerFolderService_UpdateFolder(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()
	folderID := uuid.New()
	isCollapsed := true

	t.Run("update folder name", func(t *testing.T) {
		repo := new(MockServerFolderRepository)
		eventBus := new(MockEventBus)

		repo.On("GetByID", ctx, folderID).Return(&models.ServerFolder{
			ID:        folderID,
			UserID:    userID,
			Name:      "Old Name",
			Position:  0,
			Depth:     0,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}, nil)
		repo.On("Update", ctx, mock.AnythingOfType("*models.ServerFolder")).Return(nil)
		eventBus.On("Publish", "server_folder_updated", mock.AnythingOfType("map[string]interface {}"))

		svc := NewServerFolderService(repo, nil, eventBus)
		req := &models.UpdateServerFolderRequest{
			Name: newStringPtr("New Name"),
		}

		folder, err := svc.UpdateFolder(ctx, userID, folderID, req)

		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if folder.Name != "New Name" {
			t.Errorf("expected name 'New Name', got '%s'", folder.Name)
		}

		repo.AssertExpectations(t)
		eventBus.AssertExpectations(t)
	})

	t.Run("update folder collapsed state", func(t *testing.T) {
		repo := new(MockServerFolderRepository)
		eventBus := new(MockEventBus)

		repo.On("GetByID", ctx, folderID).Return(&models.ServerFolder{
			ID:          folderID,
			UserID:      userID,
			Name:        "Folder",
			Position:    0,
			Depth:       0,
			IsCollapsed: false,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}, nil)
		repo.On("Update", ctx, mock.AnythingOfType("*models.ServerFolder")).Return(nil)
		eventBus.On("Publish", "server_folder_updated", mock.AnythingOfType("map[string]interface {}"))

		svc := NewServerFolderService(repo, nil, eventBus)
		req := &models.UpdateServerFolderRequest{
			IsCollapsed: &isCollapsed,
		}

		folder, err := svc.UpdateFolder(ctx, userID, folderID, req)

		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if !folder.IsCollapsed {
			t.Errorf("expected IsCollapsed to be true")
		}

		repo.AssertExpectations(t)
		eventBus.AssertExpectations(t)
	})

	t.Run("folder not found", func(t *testing.T) {
		repo := new(MockServerFolderRepository)

		repo.On("GetByID", ctx, folderID).Return(nil, sql.ErrNoRows)

		svc := NewServerFolderService(repo, nil, nil)
		req := &models.UpdateServerFolderRequest{
			Name: newStringPtr("New Name"),
		}

		_, err := svc.UpdateFolder(ctx, userID, folderID, req)

		if err != ErrFolderNotFound {
			t.Errorf("expected ErrFolderNotFound, got %v", err)
		}

		repo.AssertExpectations(t)
	})

	t.Run("folder belongs to different user", func(t *testing.T) {
		repo := new(MockServerFolderRepository)

		repo.On("GetByID", ctx, folderID).Return(&models.ServerFolder{
			ID:        folderID,
			UserID:    uuid.New(), // Different user
			Name:      "Folder",
			Position:  0,
			Depth:     0,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}, nil)

		svc := NewServerFolderService(repo, nil, nil)
		req := &models.UpdateServerFolderRequest{
			Name: newStringPtr("New Name"),
		}

		_, err := svc.UpdateFolder(ctx, userID, folderID, req)

		if err != ErrFolderNotFound {
			t.Errorf("expected ErrFolderNotFound, got %v", err)
		}

		repo.AssertExpectations(t)
	})
}

func TestServerFolderService_DeleteFolder(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()
	folderID := uuid.New()
	serverID := uuid.New()

	t.Run("delete folder and move servers to unassigned", func(t *testing.T) {
		repo := new(MockServerFolderRepository)
		eventBus := new(MockEventBus)

		repo.On("GetByID", ctx, folderID).Return(&models.ServerFolder{
			ID:        folderID,
			UserID:    userID,
			Name:      "Folder",
			Position:  0,
			Depth:     0,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}, nil)
		repo.On("GetChildFolderIDs", ctx, folderID).Return([]uuid.UUID{}, nil)
		repo.On("GetServersInFolder", ctx, userID, &folderID).Return([]*models.ServerInFolder{
			{
				ServerID: serverID,
				FolderID: &folderID,
				Position: 0,
			},
		}, nil)
		repo.On("AssignServerToFolder", ctx, userID, serverID, (*uuid.UUID)(nil), 0).Return(nil)
		repo.On("Delete", ctx, folderID).Return(nil)
		eventBus.On("Publish", "server_folder_deleted", mock.AnythingOfType("map[string]interface {}"))

		svc := NewServerFolderService(repo, nil, eventBus)
		err := svc.DeleteFolder(ctx, userID, folderID)

		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		repo.AssertExpectations(t)
		eventBus.AssertExpectations(t)
	})

	t.Run("folder not found", func(t *testing.T) {
		repo := new(MockServerFolderRepository)

		repo.On("GetByID", ctx, folderID).Return(nil, sql.ErrNoRows)

		svc := NewServerFolderService(repo, nil, nil)
		err := svc.DeleteFolder(ctx, userID, folderID)

		if err != ErrFolderNotFound {
			t.Errorf("expected ErrFolderNotFound, got %v", err)
		}

		repo.AssertExpectations(t)
	})
}

func TestServerFolderService_MoveServerToFolder(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()
	folderID := uuid.New()
	serverID := uuid.New()

	t.Run("move server to folder successfully", func(t *testing.T) {
		repo := new(MockServerFolderRepository)
		serverRepo := new(MockServerRepositoryForFolderTests)
		eventBus := new(MockEventBus)

		serverRepo.On("GetMember", ctx, serverID, userID).Return(&models.Member{
			UserID:   userID,
			ServerID: serverID,
			JoinedAt: time.Now(),
		}, nil)
		repo.On("GetByID", ctx, folderID).Return(&models.ServerFolder{
			ID:        folderID,
			UserID:    userID,
			Name:      "Folder",
			Position:  0,
			Depth:     0,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}, nil)
		repo.On("GetMaxPositionAtLevel", ctx, userID, 0, &folderID).Return(0, nil)
		repo.On("AssignServerToFolder", ctx, userID, serverID, &folderID, 1).Return(nil)
		eventBus.On("Publish", "server_folder_moved", mock.AnythingOfType("map[string]interface {}"))

		svc := NewServerFolderService(repo, serverRepo, eventBus)
		err := svc.MoveServerToFolder(ctx, userID, serverID, &folderID)

		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		repo.AssertExpectations(t)
		serverRepo.AssertExpectations(t)
		eventBus.AssertExpectations(t)
	})

	t.Run("move server to nil (unassigned)", func(t *testing.T) {
		repo := new(MockServerFolderRepository)
		serverRepo := new(MockServerRepositoryForFolderTests)
		eventBus := new(MockEventBus)

		serverRepo.On("GetMember", ctx, serverID, userID).Return(&models.Member{
			UserID:   userID,
			ServerID: serverID,
			JoinedAt: time.Now(),
		}, nil)
		repo.On("GetMaxPositionAtLevel", ctx, userID, 0, (*uuid.UUID)(nil)).Return(0, nil)
		repo.On("AssignServerToFolder", ctx, userID, serverID, (*uuid.UUID)(nil), 1).Return(nil)
		eventBus.On("Publish", "server_folder_moved", mock.AnythingOfType("map[string]interface {}"))

		svc := NewServerFolderService(repo, serverRepo, eventBus)
		err := svc.MoveServerToFolder(ctx, userID, serverID, nil)

		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		repo.AssertExpectations(t)
		serverRepo.AssertExpectations(t)
		eventBus.AssertExpectations(t)
	})

	t.Run("server not found in user's servers", func(t *testing.T) {
		repo := new(MockServerFolderRepository)
		serverRepo := new(MockServerRepositoryForFolderTests)

		serverRepo.On("GetMember", ctx, serverID, userID).Return(nil, sql.ErrNoRows)

		svc := NewServerFolderService(repo, serverRepo, nil)
		err := svc.MoveServerToFolder(ctx, userID, serverID, &folderID)

		if err != ErrServerNotInFolder {
			t.Errorf("expected ErrServerNotInFolder, got %v", err)
		}

		serverRepo.AssertExpectations(t)
	})
}

func TestServerFolderService_ReorderServers(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()
	folderID := uuid.New()
	serverID1 := uuid.New()
	serverID2 := uuid.New()

	t.Run("reorder servers successfully", func(t *testing.T) {
		repo := new(MockServerFolderRepository)
		eventBus := new(MockEventBus)

		repo.On("GetByID", ctx, folderID).Return(&models.ServerFolder{
			ID:        folderID,
			UserID:    userID,
			Name:      "Folder",
			Position:  0,
			Depth:     0,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}, nil)
		repo.On("UpdateServerPositions", ctx, userID, mock.AnythingOfType("[]models.ServerPosition")).Return(nil)
		eventBus.On("Publish", "server_folder_reordered", mock.AnythingOfType("map[string]interface {}"))

		svc := NewServerFolderService(repo, nil, eventBus)
		req := &models.ReorderServersRequest{
			ServerPositions: []models.ServerPosition{
				{ServerID: serverID1, Position: 1},
				{ServerID: serverID2, Position: 0},
			},
		}

		err := svc.ReorderServers(ctx, userID, &folderID, req)

		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		repo.AssertExpectations(t)
		eventBus.AssertExpectations(t)
	})

	t.Run("folder not found", func(t *testing.T) {
		repo := new(MockServerFolderRepository)

		repo.On("GetByID", ctx, folderID).Return(nil, sql.ErrNoRows)

		svc := NewServerFolderService(repo, nil, nil)
		req := &models.ReorderServersRequest{
			ServerPositions: []models.ServerPosition{
				{ServerID: serverID1, Position: 0},
			},
		}

		err := svc.ReorderServers(ctx, userID, &folderID, req)

		if err != ErrFolderNotFound {
			t.Errorf("expected ErrFolderNotFound, got %v", err)
		}

		repo.AssertExpectations(t)
	})
}

// Helper function
func newStringPtr(s string) *string {
	return &s
}
