package services

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"hearth/internal/models"
)

// mockServerFolderRepo implements ServerFolderRepositoryInterface for testing
type mockServerFolderRepo struct {
	createFunc                      func(ctx context.Context, folder *models.ServerFolder) error
	getByIDFunc                     func(ctx context.Context, userID, folderID uuid.UUID) (*models.ServerFolder, error)
	getAllForUserFunc               func(ctx context.Context, userID uuid.UUID) ([]*models.ServerFolder, error)
	updateFunc                      func(ctx context.Context, folder *models.ServerFolder) error
	deleteFunc                      func(ctx context.Context, userID, folderID uuid.UUID) error
	getServersInFolderFunc           func(ctx context.Context, userID, folderID uuid.UUID) ([]*models.ServerInFolder, error)
	getUnassignedServersFunc         func(ctx context.Context, userID uuid.UUID) ([]*models.ServerInFolder, error)
	getAllServerAssignmentsFunc      func(ctx context.Context, userID uuid.UUID) ([]*models.ServerInFolder, error)
	assignServerToFolderFunc        func(ctx context.Context, userID, serverID uuid.UUID, folderID *uuid.UUID) error
	assignServersToFolderFunc       func(ctx context.Context, userID uuid.UUID, serverIDs []uuid.UUID, folderID *uuid.UUID) error
	updateServerPositionsFunc       func(ctx context.Context, userID uuid.UUID, positions []models.ServerPosition) error
	getServerAssignmentFunc         func(ctx context.Context, userID, serverID uuid.UUID) (*models.ServerInFolder, error)
	userIsMemberOfServerFunc        func(ctx context.Context, userID, serverID uuid.UUID) (bool, error)
}

func (m *mockServerFolderRepo) Create(ctx context.Context, folder *models.ServerFolder) error {
	if m.createFunc != nil {
		return m.createFunc(ctx, folder)
	}
	return nil
}

func (m *mockServerFolderRepo) GetByID(ctx context.Context, userID, folderID uuid.UUID) (*models.ServerFolder, error) {
	if m.getByIDFunc != nil {
		return m.getByIDFunc(ctx, userID, folderID)
	}
	return nil, nil
}

func (m *mockServerFolderRepo) GetAllForUser(ctx context.Context, userID uuid.UUID) ([]*models.ServerFolder, error) {
	if m.getAllForUserFunc != nil {
		return m.getAllForUserFunc(ctx, userID)
	}
	return nil, nil
}

func (m *mockServerFolderRepo) Update(ctx context.Context, folder *models.ServerFolder) error {
	if m.updateFunc != nil {
		return m.updateFunc(ctx, folder)
	}
	return nil
}

func (m *mockServerFolderRepo) Delete(ctx context.Context, userID, folderID uuid.UUID) error {
	if m.deleteFunc != nil {
		return m.deleteFunc(ctx, userID, folderID)
	}
	return nil
}

func (m *mockServerFolderRepo) GetServersInFolder(ctx context.Context, userID, folderID uuid.UUID) ([]*models.ServerInFolder, error) {
	if m.getServersInFolderFunc != nil {
		return m.getServersInFolderFunc(ctx, userID, folderID)
	}
	return nil, nil
}

func (m *mockServerFolderRepo) GetUnassignedServers(ctx context.Context, userID uuid.UUID) ([]*models.ServerInFolder, error) {
	if m.getUnassignedServersFunc != nil {
		return m.getUnassignedServersFunc(ctx, userID)
	}
	return nil, nil
}

func (m *mockServerFolderRepo) GetAllServerAssignments(ctx context.Context, userID uuid.UUID) ([]*models.ServerInFolder, error) {
	if m.getAllServerAssignmentsFunc != nil {
		return m.getAllServerAssignmentsFunc(ctx, userID)
	}
	return nil, nil
}

func (m *mockServerFolderRepo) AssignServerToFolder(ctx context.Context, userID, serverID uuid.UUID, folderID *uuid.UUID) error {
	if m.assignServerToFolderFunc != nil {
		return m.assignServerToFolderFunc(ctx, userID, serverID, folderID)
	}
	return nil
}

func (m *mockServerFolderRepo) AssignServersToFolder(ctx context.Context, userID uuid.UUID, serverIDs []uuid.UUID, folderID *uuid.UUID) error {
	if m.assignServersToFolderFunc != nil {
		return m.assignServersToFolderFunc(ctx, userID, serverIDs, folderID)
	}
	return nil
}

func (m *mockServerFolderRepo) UpdateServerPositions(ctx context.Context, userID uuid.UUID, positions []models.ServerPosition) error {
	if m.updateServerPositionsFunc != nil {
		return m.updateServerPositionsFunc(ctx, userID, positions)
	}
	return nil
}

func (m *mockServerFolderRepo) GetServerAssignment(ctx context.Context, userID, serverID uuid.UUID) (*models.ServerInFolder, error) {
	if m.getServerAssignmentFunc != nil {
		return m.getServerAssignmentFunc(ctx, userID, serverID)
	}
	return nil, nil
}

func (m *mockServerFolderRepo) UserIsMemberOfServer(ctx context.Context, userID, serverID uuid.UUID) (bool, error) {
	if m.userIsMemberOfServerFunc != nil {
		return m.userIsMemberOfServerFunc(ctx, userID, serverID)
	}
	return false, nil
}

func TestServerFolderService_CreateFolder(t *testing.T) {
	userID := uuid.New()

	t.Run("creates folder successfully", func(t *testing.T) {
		repo := &mockServerFolderRepo{
			createFunc: func(ctx context.Context, folder *models.ServerFolder) error {
				assert.Equal(t, userID, folder.UserID)
				assert.Equal(t, "My Folder", folder.Name)
				assert.Equal(t, 0, folder.Position)
				assert.False(t, folder.IsCollapsed)
				return nil
			},
		}
		svc := NewServerFolderService(repo)

		req := &models.CreateServerFolderRequest{Name: "My Folder"}
		folder, err := svc.CreateFolder(context.Background(), userID, req)

		assert.NoError(t, err)
		assert.NotNil(t, folder)
		assert.Equal(t, "My Folder", folder.Name)
		assert.Equal(t, userID, folder.UserID)
	})

	t.Run("returns error for empty name", func(t *testing.T) {
		repo := &mockServerFolderRepo{}
		svc := NewServerFolderService(repo)

		req := &models.CreateServerFolderRequest{Name: ""}
		folder, err := svc.CreateFolder(context.Background(), userID, req)

		assert.Error(t, err)
		assert.Equal(t, ErrFolderNameRequired, err)
		assert.Nil(t, folder)
	})

	t.Run("creates folder with parent", func(t *testing.T) {
		parentID := uuid.New()
		repo := &mockServerFolderRepo{
			getByIDFunc: func(ctx context.Context, uid, fid uuid.UUID) (*models.ServerFolder, error) {
				assert.Equal(t, parentID, fid)
				return &models.ServerFolder{ID: parentID, UserID: userID, Name: "Parent"}, nil
			},
			createFunc: func(ctx context.Context, folder *models.ServerFolder) error {
				assert.NotNil(t, folder.ParentID)
				assert.Equal(t, parentID, *folder.ParentID)
				return nil
			},
		}
		svc := NewServerFolderService(repo)

		parentIDStr := parentID.String()
		req := &models.CreateServerFolderRequest{Name: "Child", ParentID: &parentIDStr}
		folder, err := svc.CreateFolder(context.Background(), userID, req)

		assert.NoError(t, err)
		assert.NotNil(t, folder)
	})

	t.Run("creates folder with position", func(t *testing.T) {
		repo := &mockServerFolderRepo{
			createFunc: func(ctx context.Context, folder *models.ServerFolder) error {
				assert.Equal(t, 5, folder.Position)
				return nil
			},
		}
		svc := NewServerFolderService(repo)

		pos := 5
		req := &models.CreateServerFolderRequest{Name: "Test", Position: &pos}
		folder, err := svc.CreateFolder(context.Background(), userID, req)

		assert.NoError(t, err)
		assert.NotNil(t, folder)
		assert.Equal(t, 5, folder.Position)
	})
}

func TestServerFolderService_GetFolder(t *testing.T) {
	userID := uuid.New()
	folderID := uuid.New()

	t.Run("returns folder by id", func(t *testing.T) {
		repo := &mockServerFolderRepo{
			getByIDFunc: func(ctx context.Context, uid, fid uuid.UUID) (*models.ServerFolder, error) {
				assert.Equal(t, userID, uid)
				assert.Equal(t, folderID, fid)
				return &models.ServerFolder{
					ID:          fid,
					UserID:      uid,
					Name:        "Test Folder",
					Position:    0,
					IsCollapsed: false,
					CreatedAt:   time.Now(),
					UpdatedAt:   time.Now(),
				}, nil
			},
		}
		svc := NewServerFolderService(repo)

		folder, err := svc.GetFolder(context.Background(), userID, folderID)

		assert.NoError(t, err)
		assert.NotNil(t, folder)
		assert.Equal(t, "Test Folder", folder.Name)
	})

	t.Run("returns ErrFolderNotFound for missing folder", func(t *testing.T) {
		repo := &mockServerFolderRepo{
			getByIDFunc: func(ctx context.Context, uid, fid uuid.UUID) (*models.ServerFolder, error) {
				return nil, nil
			},
		}
		svc := NewServerFolderService(repo)

		folder, err := svc.GetFolder(context.Background(), userID, folderID)

		assert.Error(t, err)
		assert.Equal(t, ErrFolderNotFound, err)
		assert.Nil(t, folder)
	})
}

func TestServerFolderService_UpdateFolder(t *testing.T) {
	userID := uuid.New()
	folderID := uuid.New()

	t.Run("updates folder name", func(t *testing.T) {
		repo := &mockServerFolderRepo{
			getByIDFunc: func(ctx context.Context, uid, fid uuid.UUID) (*models.ServerFolder, error) {
				return &models.ServerFolder{
					ID:        fid,
					UserID:    uid,
					Name:      "Old Name",
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				}, nil
			},
			updateFunc: func(ctx context.Context, folder *models.ServerFolder) error {
				assert.Equal(t, "New Name", folder.Name)
				return nil
			},
		}
		svc := NewServerFolderService(repo)

		newName := "New Name"
		req := &models.UpdateServerFolderRequest{Name: &newName}
		folder, err := svc.UpdateFolder(context.Background(), userID, folderID, req)

		assert.NoError(t, err)
		assert.NotNil(t, folder)
		assert.Equal(t, "New Name", folder.Name)
	})

	t.Run("updates folder collapsed state", func(t *testing.T) {
		repo := &mockServerFolderRepo{
			getByIDFunc: func(ctx context.Context, uid, fid uuid.UUID) (*models.ServerFolder, error) {
				return &models.ServerFolder{
					ID:          fid,
					UserID:      uid,
					Name:        "Test",
					IsCollapsed: false,
					CreatedAt:   time.Now(),
					UpdatedAt:   time.Now(),
				}, nil
			},
			updateFunc: func(ctx context.Context, folder *models.ServerFolder) error {
				assert.True(t, folder.IsCollapsed)
				return nil
			},
		}
		svc := NewServerFolderService(repo)

		collapsed := true
		req := &models.UpdateServerFolderRequest{IsCollapsed: &collapsed}
		folder, err := svc.UpdateFolder(context.Background(), userID, folderID, req)

		assert.NoError(t, err)
		assert.NotNil(t, folder)
		assert.True(t, folder.IsCollapsed)
	})

	t.Run("clears parent id when empty string", func(t *testing.T) {
		parentID := uuid.New()
		repo := &mockServerFolderRepo{
			getByIDFunc: func(ctx context.Context, uid, fid uuid.UUID) (*models.ServerFolder, error) {
				return &models.ServerFolder{
					ID:        fid,
					UserID:    uid,
					Name:      "Test",
					ParentID:  &parentID,
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				}, nil
			},
			updateFunc: func(ctx context.Context, folder *models.ServerFolder) error {
				assert.Nil(t, folder.ParentID)
				return nil
			},
		}
		svc := NewServerFolderService(repo)

		emptyParent := ""
		req := &models.UpdateServerFolderRequest{ParentID: &emptyParent}
		folder, err := svc.UpdateFolder(context.Background(), userID, folderID, req)

		assert.NoError(t, err)
		assert.NotNil(t, folder)
		assert.Nil(t, folder.ParentID)
	})

	t.Run("returns ErrFolderNotFound for missing folder", func(t *testing.T) {
		repo := &mockServerFolderRepo{
			getByIDFunc: func(ctx context.Context, uid, fid uuid.UUID) (*models.ServerFolder, error) {
				return nil, nil
			},
		}
		svc := NewServerFolderService(repo)

		newName := "New Name"
		req := &models.UpdateServerFolderRequest{Name: &newName}
		folder, err := svc.UpdateFolder(context.Background(), userID, folderID, req)

		assert.Error(t, err)
		assert.Equal(t, ErrFolderNotFound, err)
		assert.Nil(t, folder)
	})
}

func TestServerFolderService_DeleteFolder(t *testing.T) {
	userID := uuid.New()
	folderID := uuid.New()

	t.Run("deletes folder successfully", func(t *testing.T) {
		repo := &mockServerFolderRepo{
			getByIDFunc: func(ctx context.Context, uid, fid uuid.UUID) (*models.ServerFolder, error) {
				return &models.ServerFolder{
					ID:      fid,
					UserID:  uid,
					Name:    "Test",
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				}, nil
			},
			deleteFunc: func(ctx context.Context, uid, fid uuid.UUID) error {
				assert.Equal(t, userID, uid)
				assert.Equal(t, folderID, fid)
				return nil
			},
		}
		svc := NewServerFolderService(repo)

		err := svc.DeleteFolder(context.Background(), userID, folderID)

		assert.NoError(t, err)
	})

	t.Run("returns ErrFolderNotFound for missing folder", func(t *testing.T) {
		repo := &mockServerFolderRepo{
			getByIDFunc: func(ctx context.Context, uid, fid uuid.UUID) (*models.ServerFolder, error) {
				return nil, nil
			},
		}
		svc := NewServerFolderService(repo)

		err := svc.DeleteFolder(context.Background(), userID, folderID)

		assert.Error(t, err)
		assert.Equal(t, ErrFolderNotFound, err)
	})
}

func TestServerFolderService_MoveServerToFolder(t *testing.T) {
	userID := uuid.New()
	serverID := uuid.New()
	folderID := uuid.New()

	t.Run("moves server to folder", func(t *testing.T) {
		repo := &mockServerFolderRepo{
			userIsMemberOfServerFunc: func(ctx context.Context, uid, sid uuid.UUID) (bool, error) {
				return true, nil
			},
			getByIDFunc: func(ctx context.Context, uid, fid uuid.UUID) (*models.ServerFolder, error) {
				return &models.ServerFolder{ID: fid, UserID: uid}, nil
			},
			assignServerToFolderFunc: func(ctx context.Context, uid, sid uuid.UUID, fid *uuid.UUID) error {
				assert.Equal(t, userID, uid)
				assert.Equal(t, serverID, sid)
				assert.NotNil(t, fid)
				assert.Equal(t, folderID, *fid)
				return nil
			},
		}
		svc := NewServerFolderService(repo)

		err := svc.MoveServerToFolder(context.Background(), userID, serverID, &folderID)

		assert.NoError(t, err)
	})

	t.Run("moves server to root (nil folder)", func(t *testing.T) {
		repo := &mockServerFolderRepo{
			userIsMemberOfServerFunc: func(ctx context.Context, uid, sid uuid.UUID) (bool, error) {
				return true, nil
			},
			assignServerToFolderFunc: func(ctx context.Context, uid, sid uuid.UUID, fid *uuid.UUID) error {
				assert.Nil(t, fid)
				return nil
			},
		}
		svc := NewServerFolderService(repo)

		err := svc.MoveServerToFolder(context.Background(), userID, serverID, nil)

		assert.NoError(t, err)
	})

	t.Run("returns ErrNotServerMember when not member", func(t *testing.T) {
		repo := &mockServerFolderRepo{
			userIsMemberOfServerFunc: func(ctx context.Context, uid, sid uuid.UUID) (bool, error) {
				return false, nil
			},
		}
		svc := NewServerFolderService(repo)

		err := svc.MoveServerToFolder(context.Background(), userID, serverID, &folderID)

		assert.Error(t, err)
		assert.Equal(t, ErrNotServerMember, err)
	})

	t.Run("returns ErrFolderNotFound for missing folder", func(t *testing.T) {
		repo := &mockServerFolderRepo{
			userIsMemberOfServerFunc: func(ctx context.Context, uid, sid uuid.UUID) (bool, error) {
				return true, nil
			},
			getByIDFunc: func(ctx context.Context, uid, fid uuid.UUID) (*models.ServerFolder, error) {
				return nil, nil
			},
		}
		svc := NewServerFolderService(repo)

		err := svc.MoveServerToFolder(context.Background(), userID, serverID, &folderID)

		assert.Error(t, err)
		assert.Equal(t, ErrFolderNotFound, err)
	})
}

func TestServerFolderService_MoveServersToFolder(t *testing.T) {
	userID := uuid.New()
	serverID1 := uuid.New()
	serverID2 := uuid.New()
	folderID := uuid.New()

	t.Run("moves multiple servers to folder", func(t *testing.T) {
		repo := &mockServerFolderRepo{
			getByIDFunc: func(ctx context.Context, uid, fid uuid.UUID) (*models.ServerFolder, error) {
				return &models.ServerFolder{ID: fid, UserID: uid}, nil
			},
			userIsMemberOfServerFunc: func(ctx context.Context, uid, sid uuid.UUID) (bool, error) {
				return true, nil
			},
			assignServersToFolderFunc: func(ctx context.Context, uid uuid.UUID, serverIDs []uuid.UUID, fid *uuid.UUID) error {
				assert.Equal(t, userID, uid)
				assert.Len(t, serverIDs, 2)
				assert.Equal(t, serverID1, serverIDs[0])
				assert.Equal(t, serverID2, serverIDs[1])
				assert.NotNil(t, fid)
				assert.Equal(t, folderID, *fid)
				return nil
			},
		}
		svc := NewServerFolderService(repo)

		req := &models.MoveServersToFolderRequest{
			ServerIDs: []string{serverID1.String(), serverID2.String()},
			FolderID:  folderSvcTestStringPtr(folderID.String()),
		}
		err := svc.MoveServersToFolder(context.Background(), userID, req)

		assert.NoError(t, err)
	})

	t.Run("returns error for invalid server id", func(t *testing.T) {
		repo := &mockServerFolderRepo{}
		svc := NewServerFolderService(repo)

		req := &models.MoveServersToFolderRequest{
			ServerIDs: []string{"invalid-uuid"},
		}
		err := svc.MoveServersToFolder(context.Background(), userID, req)

		assert.Error(t, err)
	})

	t.Run("returns ErrNotServerMember when not member of one server", func(t *testing.T) {
		repo := &mockServerFolderRepo{
			userIsMemberOfServerFunc: func(ctx context.Context, uid, sid uuid.UUID) (bool, error) {
				if sid == serverID2 {
					return false, nil
				}
				return true, nil
			},
		}
		svc := NewServerFolderService(repo)

		req := &models.MoveServersToFolderRequest{
			ServerIDs: []string{serverID1.String(), serverID2.String()},
		}
		err := svc.MoveServersToFolder(context.Background(), userID, req)

		assert.Error(t, err)
		assert.Equal(t, ErrNotServerMember, err)
	})
}

func TestServerFolderService_ReorderServers(t *testing.T) {
	userID := uuid.New()
	folderID := uuid.New()
	serverID1 := uuid.New()
	serverID2 := uuid.New()

	t.Run("reorders servers successfully", func(t *testing.T) {
		repo := &mockServerFolderRepo{
			getByIDFunc: func(ctx context.Context, uid, fid uuid.UUID) (*models.ServerFolder, error) {
				return &models.ServerFolder{ID: fid, UserID: uid}, nil
			},
			updateServerPositionsFunc: func(ctx context.Context, uid uuid.UUID, positions []models.ServerPosition) error {
				assert.Equal(t, userID, uid)
				assert.Len(t, positions, 2)
				return nil
			},
		}
		svc := NewServerFolderService(repo)

		req := &models.ReorderServersRequest{
			ServerPositions: []models.ServerPosition{
				{ServerID: serverID1.String(), Position: 0},
				{ServerID: serverID2.String(), Position: 1},
			},
		}
		err := svc.ReorderServers(context.Background(), userID, &folderID, req)

		assert.NoError(t, err)
	})

	t.Run("reorders root servers with nil folder id", func(t *testing.T) {
		repo := &mockServerFolderRepo{
			updateServerPositionsFunc: func(ctx context.Context, uid uuid.UUID, positions []models.ServerPosition) error {
				assert.Equal(t, userID, uid)
				return nil
			},
		}
		svc := NewServerFolderService(repo)

		req := &models.ReorderServersRequest{
			ServerPositions: []models.ServerPosition{
				{ServerID: serverID1.String(), Position: 0},
			},
		}
		err := svc.ReorderServers(context.Background(), userID, nil, req)

		assert.NoError(t, err)
	})

	t.Run("returns ErrFolderNotFound for missing folder", func(t *testing.T) {
		repo := &mockServerFolderRepo{
			getByIDFunc: func(ctx context.Context, uid, fid uuid.UUID) (*models.ServerFolder, error) {
				return nil, nil
			},
		}
		svc := NewServerFolderService(repo)

		req := &models.ReorderServersRequest{
			ServerPositions: []models.ServerPosition{
				{ServerID: serverID1.String(), Position: 0},
			},
		}
		err := svc.ReorderServers(context.Background(), userID, &folderID, req)

		assert.Error(t, err)
		assert.Equal(t, ErrFolderNotFound, err)
	})
}

func TestServerFolderService_GetUserFolders(t *testing.T) {
	userID := uuid.New()
	folderID1 := uuid.New()
	folderID2 := uuid.New()
	serverID := uuid.New()

	t.Run("returns empty tree when no folders", func(t *testing.T) {
		repo := &mockServerFolderRepo{
			getAllForUserFunc: func(ctx context.Context, uid uuid.UUID) ([]*models.ServerFolder, error) {
				return []*models.ServerFolder{}, nil
			},
			getAllServerAssignmentsFunc: func(ctx context.Context, uid uuid.UUID) ([]*models.ServerInFolder, error) {
				return []*models.ServerInFolder{}, nil
			},
		}
		svc := NewServerFolderService(repo)

		tree, err := svc.GetUserFolders(context.Background(), userID)

		assert.NoError(t, err)
		assert.NotNil(t, tree)
		assert.Len(t, tree.Folders, 0)
		assert.Len(t, tree.Servers, 0)
	})

	t.Run("returns folders with servers", func(t *testing.T) {
		repo := &mockServerFolderRepo{
			getAllForUserFunc: func(ctx context.Context, uid uuid.UUID) ([]*models.ServerFolder, error) {
				return []*models.ServerFolder{
					{
						ID:          folderID1,
						UserID:      uid,
						Name:        "Folder 1",
						Position:    0,
						IsCollapsed: false,
						CreatedAt:   time.Now(),
						UpdatedAt:   time.Now(),
					},
				}, nil
			},
			getAllServerAssignmentsFunc: func(ctx context.Context, uid uuid.UUID) ([]*models.ServerInFolder, error) {
				return []*models.ServerInFolder{
					{
						ServerID:   serverID,
						FolderID:   &folderID1,
						UserID:     uid,
						Position:   0,
						AssignedAt: time.Now(),
					},
				}, nil
			},
		}
		svc := NewServerFolderService(repo)

		tree, err := svc.GetUserFolders(context.Background(), userID)

		assert.NoError(t, err)
		assert.NotNil(t, tree)
		assert.Len(t, tree.Folders, 1)
		assert.Equal(t, "Folder 1", tree.Folders[0].Name)
		assert.Len(t, tree.Folders[0].Servers, 1)
		assert.Equal(t, serverID, tree.Folders[0].Servers[0].ServerID)
	})

	t.Run("separates unassigned servers", func(t *testing.T) {
		serverID2 := uuid.New()
		repo := &mockServerFolderRepo{
			getAllForUserFunc: func(ctx context.Context, uid uuid.UUID) ([]*models.ServerFolder, error) {
				return []*models.ServerFolder{
					{
						ID:          folderID1,
						UserID:      uid,
						Name:        "Folder 1",
						Position:    0,
						IsCollapsed: false,
						CreatedAt:   time.Now(),
						UpdatedAt:   time.Now(),
					},
				}, nil
			},
			getAllServerAssignmentsFunc: func(ctx context.Context, uid uuid.UUID) ([]*models.ServerInFolder, error) {
				return []*models.ServerInFolder{
					{
						ServerID:   serverID,
						FolderID:   &folderID1,
						UserID:     uid,
						Position:   0,
						AssignedAt: time.Now(),
					},
					{
						ServerID:   serverID2,
						FolderID:   nil,
						UserID:     uid,
						Position:   0,
						AssignedAt: time.Now(),
					},
				}, nil
			},
		}
		svc := NewServerFolderService(repo)

		tree, err := svc.GetUserFolders(context.Background(), userID)

		assert.NoError(t, err)
		assert.NotNil(t, tree)
		assert.Len(t, tree.Folders, 1)
		assert.Len(t, tree.Servers, 1)
		assert.Equal(t, serverID2, tree.Servers[0].ServerID)
	})

	t.Run("builds nested folder hierarchy", func(t *testing.T) {
		repo := &mockServerFolderRepo{
			getAllForUserFunc: func(ctx context.Context, uid uuid.UUID) ([]*models.ServerFolder, error) {
				return []*models.ServerFolder{
					{
						ID:          folderID1,
						UserID:      uid,
						Name:        "Parent",
						Position:    0,
						IsCollapsed: false,
						CreatedAt:   time.Now(),
						UpdatedAt:   time.Now(),
					},
					{
						ID:          folderID2,
						UserID:      uid,
						ParentID:    &folderID1,
						Name:        "Child",
						Position:    0,
						IsCollapsed: false,
						CreatedAt:   time.Now(),
						UpdatedAt:   time.Now(),
					},
				}, nil
			},
			getAllServerAssignmentsFunc: func(ctx context.Context, uid uuid.UUID) ([]*models.ServerInFolder, error) {
				return []*models.ServerInFolder{}, nil
			},
		}
		svc := NewServerFolderService(repo)

		tree, err := svc.GetUserFolders(context.Background(), userID)

		assert.NoError(t, err)
		assert.NotNil(t, tree)
		assert.Len(t, tree.Folders, 1)
		assert.Equal(t, "Parent", tree.Folders[0].Name)
		assert.Len(t, tree.Folders[0].Children, 1)
		assert.Equal(t, "Child", tree.Folders[0].Children[0].Name)
	})
}

// Helper function
func folderSvcTestStringPtr(s string) *string {
	return &s
}
