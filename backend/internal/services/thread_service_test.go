package services

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"hearth/internal/models"
)

// mockThreadRepository mocks ThreadRepository for service tests
type mockThreadRepository struct {
	createFunc                func(ctx context.Context, thread *models.Thread) error
	getByIDFunc               func(ctx context.Context, id uuid.UUID) (*models.Thread, error)
	updateFunc                func(ctx context.Context, thread *models.Thread) error
	deleteFunc                func(ctx context.Context, id uuid.UUID) error
	getByChannelIDFunc        func(ctx context.Context, channelID uuid.UUID) ([]*models.Thread, error)
	getActiveByChannelIDFunc  func(ctx context.Context, channelID uuid.UUID) ([]*models.Thread, error)
	archiveFunc               func(ctx context.Context, id uuid.UUID) error
	unarchiveFunc             func(ctx context.Context, id uuid.UUID) error
	addMemberFunc             func(ctx context.Context, threadID, userID uuid.UUID) error
	removeMemberFunc          func(ctx context.Context, threadID, userID uuid.UUID) error
	isMemberFunc              func(ctx context.Context, threadID, userID uuid.UUID) (bool, error)
	getMembersFunc            func(ctx context.Context, threadID uuid.UUID) ([]uuid.UUID, error)
	createMessageFunc         func(ctx context.Context, threadID, authorID uuid.UUID, content string) (*models.ThreadMessage, error)
	getMessagesFunc           func(ctx context.Context, threadID uuid.UUID, before *uuid.UUID, limit int) ([]*models.ThreadMessage, error)
	incrementMessageCountFunc func(ctx context.Context, threadID uuid.UUID) error
}

func (m *mockThreadRepository) Create(ctx context.Context, thread *models.Thread) error {
	if m.createFunc != nil {
		return m.createFunc(ctx, thread)
	}
	return nil
}

func (m *mockThreadRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Thread, error) {
	if m.getByIDFunc != nil {
		return m.getByIDFunc(ctx, id)
	}
	return &models.Thread{ID: id, Name: "Test Thread"}, nil
}

func (m *mockThreadRepository) Update(ctx context.Context, thread *models.Thread) error {
	if m.updateFunc != nil {
		return m.updateFunc(ctx, thread)
	}
	return nil
}

func (m *mockThreadRepository) Delete(ctx context.Context, id uuid.UUID) error {
	if m.deleteFunc != nil {
		return m.deleteFunc(ctx, id)
	}
	return nil
}

func (m *mockThreadRepository) GetByChannelID(ctx context.Context, channelID uuid.UUID) ([]*models.Thread, error) {
	if m.getByChannelIDFunc != nil {
		return m.getByChannelIDFunc(ctx, channelID)
	}
	return []*models.Thread{}, nil
}

func (m *mockThreadRepository) GetActiveByChannelID(ctx context.Context, channelID uuid.UUID) ([]*models.Thread, error) {
	if m.getActiveByChannelIDFunc != nil {
		return m.getActiveByChannelIDFunc(ctx, channelID)
	}
	return []*models.Thread{}, nil
}

func (m *mockThreadRepository) Archive(ctx context.Context, id uuid.UUID) error {
	if m.archiveFunc != nil {
		return m.archiveFunc(ctx, id)
	}
	return nil
}

func (m *mockThreadRepository) Unarchive(ctx context.Context, id uuid.UUID) error {
	if m.unarchiveFunc != nil {
		return m.unarchiveFunc(ctx, id)
	}
	return nil
}

func (m *mockThreadRepository) AddMember(ctx context.Context, threadID, userID uuid.UUID) error {
	if m.addMemberFunc != nil {
		return m.addMemberFunc(ctx, threadID, userID)
	}
	return nil
}

func (m *mockThreadRepository) RemoveMember(ctx context.Context, threadID, userID uuid.UUID) error {
	if m.removeMemberFunc != nil {
		return m.removeMemberFunc(ctx, threadID, userID)
	}
	return nil
}

func (m *mockThreadRepository) IsMember(ctx context.Context, threadID, userID uuid.UUID) (bool, error) {
	if m.isMemberFunc != nil {
		return m.isMemberFunc(ctx, threadID, userID)
	}
	return true, nil
}

func (m *mockThreadRepository) GetMembers(ctx context.Context, threadID uuid.UUID) ([]uuid.UUID, error) {
	if m.getMembersFunc != nil {
		return m.getMembersFunc(ctx, threadID)
	}
	return []uuid.UUID{}, nil
}

func (m *mockThreadRepository) CreateMessage(ctx context.Context, threadID, authorID uuid.UUID, content string) (*models.ThreadMessage, error) {
	if m.createMessageFunc != nil {
		return m.createMessageFunc(ctx, threadID, authorID, content)
	}
	return &models.ThreadMessage{
		ID:        uuid.New(),
		ThreadID:  threadID,
		AuthorID:  authorID,
		Content:   content,
		CreatedAt: time.Now(),
	}, nil
}

func (m *mockThreadRepository) GetMessages(ctx context.Context, threadID uuid.UUID, before *uuid.UUID, limit int) ([]*models.ThreadMessage, error) {
	if m.getMessagesFunc != nil {
		return m.getMessagesFunc(ctx, threadID, before, limit)
	}
	return []*models.ThreadMessage{}, nil
}

func (m *mockThreadRepository) IncrementMessageCount(ctx context.Context, threadID uuid.UUID) error {
	if m.incrementMessageCountFunc != nil {
		return m.incrementMessageCountFunc(ctx, threadID)
	}
	return nil
}

// mockChannelRepoForThread mocks ChannelRepository for thread service tests
type mockChannelRepoForThread struct {
	getByIDFunc func(ctx context.Context, id uuid.UUID) (*models.Channel, error)
}

func (m *mockChannelRepoForThread) Create(ctx context.Context, channel *models.Channel) error {
	return nil
}

func (m *mockChannelRepoForThread) GetByID(ctx context.Context, id uuid.UUID) (*models.Channel, error) {
	if m.getByIDFunc != nil {
		return m.getByIDFunc(ctx, id)
	}
	serverID := uuid.New()
	return &models.Channel{ID: id, ServerID: &serverID, Name: "test-channel", Type: models.ChannelTypeText}, nil
}

func (m *mockChannelRepoForThread) Update(ctx context.Context, channel *models.Channel) error {
	return nil
}

func (m *mockChannelRepoForThread) Delete(ctx context.Context, id uuid.UUID) error {
	return nil
}

func (m *mockChannelRepoForThread) GetByServerID(ctx context.Context, serverID uuid.UUID) ([]*models.Channel, error) {
	return []*models.Channel{}, nil
}

func (m *mockChannelRepoForThread) GetDMChannel(ctx context.Context, user1ID, user2ID uuid.UUID) (*models.Channel, error) {
	return nil, nil
}

func (m *mockChannelRepoForThread) GetUserDMs(ctx context.Context, userID uuid.UUID) ([]*models.Channel, error) {
	return []*models.Channel{}, nil
}

func (m *mockChannelRepoForThread) UpdateLastMessage(ctx context.Context, channelID, messageID uuid.UUID, at time.Time) error {
	return nil
}

// mockServerRepoForThread mocks ServerRepository for thread service tests
type mockServerRepoForThread struct {
	getMemberFunc func(ctx context.Context, serverID, userID uuid.UUID) (*models.Member, error)
}

func (m *mockServerRepoForThread) Create(ctx context.Context, server *models.Server) error {
	return nil
}

func (m *mockServerRepoForThread) GetByID(ctx context.Context, id uuid.UUID) (*models.Server, error) {
	return &models.Server{ID: id, Name: "Test Server"}, nil
}

func (m *mockServerRepoForThread) Update(ctx context.Context, server *models.Server) error {
	return nil
}

func (m *mockServerRepoForThread) Delete(ctx context.Context, id uuid.UUID) error {
	return nil
}

func (m *mockServerRepoForThread) TransferOwnership(ctx context.Context, serverID, newOwnerID uuid.UUID) error {
	return nil
}

func (m *mockServerRepoForThread) GetMember(ctx context.Context, serverID, userID uuid.UUID) (*models.Member, error) {
	if m.getMemberFunc != nil {
		return m.getMemberFunc(ctx, serverID, userID)
	}
	return &models.Member{ServerID: serverID, UserID: userID}, nil
}

func (m *mockServerRepoForThread) AddMember(ctx context.Context, member *models.Member) error {
	return nil
}

func (m *mockServerRepoForThread) RemoveMember(ctx context.Context, serverID, userID uuid.UUID) error {
	return nil
}

func (m *mockServerRepoForThread) UpdateMember(ctx context.Context, member *models.Member) error {
	return nil
}

func (m *mockServerRepoForThread) GetMembers(ctx context.Context, serverID uuid.UUID, limit, offset int) ([]*models.Member, error) {
	return []*models.Member{}, nil
}

func (m *mockServerRepoForThread) GetMemberCount(ctx context.Context, serverID uuid.UUID) (int, error) {
	return 0, nil
}

func (m *mockServerRepoForThread) GetUserServers(ctx context.Context, userID uuid.UUID) ([]*models.Server, error) {
	return []*models.Server{}, nil
}

func (m *mockServerRepoForThread) GetOwnedServersCount(ctx context.Context, userID uuid.UUID) (int, error) {
	return 0, nil
}

func (m *mockServerRepoForThread) GetBan(ctx context.Context, serverID, userID uuid.UUID) (*models.Ban, error) {
	return nil, nil
}

func (m *mockServerRepoForThread) AddBan(ctx context.Context, ban *models.Ban) error {
	return nil
}

func (m *mockServerRepoForThread) RemoveBan(ctx context.Context, serverID, userID uuid.UUID) error {
	return nil
}

func (m *mockServerRepoForThread) GetBans(ctx context.Context, serverID uuid.UUID) ([]*models.Ban, error) {
	return []*models.Ban{}, nil
}

func (m *mockServerRepoForThread) CreateInvite(ctx context.Context, invite *models.Invite) error {
	return nil
}

func (m *mockServerRepoForThread) GetInvite(ctx context.Context, code string) (*models.Invite, error) {
	return nil, nil
}

func (m *mockServerRepoForThread) GetInvites(ctx context.Context, serverID uuid.UUID) ([]*models.Invite, error) {
	return []*models.Invite{}, nil
}

func (m *mockServerRepoForThread) DeleteInvite(ctx context.Context, code string) error {
	return nil
}

func (m *mockServerRepoForThread) IncrementInviteUses(ctx context.Context, code string) error {
	return nil
}

// mockEventBusForThread mocks EventBus for thread service tests
type mockEventBusForThread struct {
	events []string
}

func (m *mockEventBusForThread) Publish(event string, data interface{}) {
	m.events = append(m.events, event)
}

func (m *mockEventBusForThread) Subscribe(event string, handler func(data interface{})) {}

func (m *mockEventBusForThread) Unsubscribe(event string, handler func(data interface{})) {}

func TestThreadService_CreateThread(t *testing.T) {
	ctx := context.Background()
	channelID := uuid.New()
	userID := uuid.New()
	serverID := uuid.New()

	tests := []struct {
		name          string
		channelID     uuid.UUID
		creatorID     uuid.UUID
		threadName    string
		autoArchive   *int
		setupMocks    func(*mockThreadRepository, *mockChannelRepoForThread, *mockServerRepoForThread)
		wantErr       error
		checkThread   func(*testing.T, *models.Thread)
	}{
		{
			name:        "success",
			channelID:   channelID,
			creatorID:   userID,
			threadName:  "Test Thread",
			autoArchive: nil,
			setupMocks: func(tr *mockThreadRepository, cr *mockChannelRepoForThread, sr *mockServerRepoForThread) {
				cr.getByIDFunc = func(ctx context.Context, id uuid.UUID) (*models.Channel, error) {
					return &models.Channel{ID: id, ServerID: &serverID, Type: models.ChannelTypeText}, nil
				}
				sr.getMemberFunc = func(ctx context.Context, sID, uID uuid.UUID) (*models.Member, error) {
					return &models.Member{ServerID: sID, UserID: uID}, nil
				}
			},
			wantErr: nil,
			checkThread: func(t *testing.T, thread *models.Thread) {
				if thread.Name != "Test Thread" {
					t.Errorf("expected name %q, got %q", "Test Thread", thread.Name)
				}
				if thread.AutoArchive != models.AutoArchive24Hour {
					t.Errorf("expected auto archive %d, got %d", models.AutoArchive24Hour, thread.AutoArchive)
				}
			},
		},
		{
			name:        "success with custom auto archive",
			channelID:   channelID,
			creatorID:   userID,
			threadName:  "Test Thread",
			autoArchive: intPtr(models.AutoArchive1Hour),
			setupMocks: func(tr *mockThreadRepository, cr *mockChannelRepoForThread, sr *mockServerRepoForThread) {
				cr.getByIDFunc = func(ctx context.Context, id uuid.UUID) (*models.Channel, error) {
					return &models.Channel{ID: id, ServerID: &serverID, Type: models.ChannelTypeText}, nil
				}
				sr.getMemberFunc = func(ctx context.Context, sID, uID uuid.UUID) (*models.Member, error) {
					return &models.Member{ServerID: sID, UserID: uID}, nil
				}
			},
			wantErr: nil,
			checkThread: func(t *testing.T, thread *models.Thread) {
				if thread.AutoArchive != models.AutoArchive1Hour {
					t.Errorf("expected auto archive %d, got %d", models.AutoArchive1Hour, thread.AutoArchive)
				}
			},
		},
		{
			name:        "channel not found",
			channelID:   channelID,
			creatorID:   userID,
			threadName:  "Test Thread",
			autoArchive: nil,
			setupMocks: func(tr *mockThreadRepository, cr *mockChannelRepoForThread, sr *mockServerRepoForThread) {
				cr.getByIDFunc = func(ctx context.Context, id uuid.UUID) (*models.Channel, error) {
					return nil, nil
				}
			},
			wantErr: ErrChannelNotFound,
		},
		{
			name:        "not server member",
			channelID:   channelID,
			creatorID:   userID,
			threadName:  "Test Thread",
			autoArchive: nil,
			setupMocks: func(tr *mockThreadRepository, cr *mockChannelRepoForThread, sr *mockServerRepoForThread) {
				cr.getByIDFunc = func(ctx context.Context, id uuid.UUID) (*models.Channel, error) {
					return &models.Channel{ID: id, ServerID: &serverID, Type: models.ChannelTypeText}, nil
				}
				sr.getMemberFunc = func(ctx context.Context, sID, uID uuid.UUID) (*models.Member, error) {
					return nil, nil
				}
			},
			wantErr: ErrNotServerMember,
		},
		{
			name:        "invalid auto archive",
			channelID:   channelID,
			creatorID:   userID,
			threadName:  "Test Thread",
			autoArchive: intPtr(999),
			setupMocks: func(tr *mockThreadRepository, cr *mockChannelRepoForThread, sr *mockServerRepoForThread) {
				cr.getByIDFunc = func(ctx context.Context, id uuid.UUID) (*models.Channel, error) {
					return &models.Channel{ID: id, ServerID: &serverID, Type: models.ChannelTypeText}, nil
				}
				sr.getMemberFunc = func(ctx context.Context, sID, uID uuid.UUID) (*models.Member, error) {
					return &models.Member{ServerID: sID, UserID: uID}, nil
				}
			},
			wantErr: ErrInvalidAutoArchive,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			threadRepo := &mockThreadRepository{}
			channelRepo := &mockChannelRepoForThread{}
			serverRepo := &mockServerRepoForThread{}
			eventBus := &mockEventBusForThread{}

			tt.setupMocks(threadRepo, channelRepo, serverRepo)

			svc := NewThreadService(threadRepo, channelRepo, serverRepo, eventBus)

			thread, err := svc.CreateThread(ctx, tt.channelID, tt.creatorID, tt.threadName, tt.autoArchive)

			if tt.wantErr != nil {
				if err != tt.wantErr {
					t.Errorf("expected error %v, got %v", tt.wantErr, err)
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if tt.checkThread != nil {
				tt.checkThread(t, thread)
			}
		})
	}
}

func TestThreadService_GetThread(t *testing.T) {
	ctx := context.Background()
	threadID := uuid.New()

	tests := []struct {
		name       string
		threadID   uuid.UUID
		setupMocks func(*mockThreadRepository)
		wantErr    error
	}{
		{
			name:     "success",
			threadID: threadID,
			setupMocks: func(tr *mockThreadRepository) {
				tr.getByIDFunc = func(ctx context.Context, id uuid.UUID) (*models.Thread, error) {
					return &models.Thread{ID: id, Name: "Test Thread"}, nil
				}
			},
			wantErr: nil,
		},
		{
			name:     "not found",
			threadID: threadID,
			setupMocks: func(tr *mockThreadRepository) {
				tr.getByIDFunc = func(ctx context.Context, id uuid.UUID) (*models.Thread, error) {
					return nil, nil
				}
			},
			wantErr: ErrThreadNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			threadRepo := &mockThreadRepository{}
			channelRepo := &mockChannelRepoForThread{}
			serverRepo := &mockServerRepoForThread{}
			eventBus := &mockEventBusForThread{}

			tt.setupMocks(threadRepo)

			svc := NewThreadService(threadRepo, channelRepo, serverRepo, eventBus)

			thread, err := svc.GetThread(ctx, tt.threadID)

			if tt.wantErr != nil {
				if err != tt.wantErr {
					t.Errorf("expected error %v, got %v", tt.wantErr, err)
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if thread.ID != tt.threadID {
				t.Errorf("expected thread ID %v, got %v", tt.threadID, thread.ID)
			}
		})
	}
}

func TestThreadService_SendThreadMessage(t *testing.T) {
	ctx := context.Background()
	threadID := uuid.New()
	userID := uuid.New()
	channelID := uuid.New()
	serverID := uuid.New()

	tests := []struct {
		name       string
		threadID   uuid.UUID
		authorID   uuid.UUID
		content    string
		setupMocks func(*mockThreadRepository, *mockChannelRepoForThread, *mockServerRepoForThread)
		wantErr    error
	}{
		{
			name:     "success",
			threadID: threadID,
			authorID: userID,
			content:  "Hello thread!",
			setupMocks: func(tr *mockThreadRepository, cr *mockChannelRepoForThread, sr *mockServerRepoForThread) {
				tr.getByIDFunc = func(ctx context.Context, id uuid.UUID) (*models.Thread, error) {
					return &models.Thread{ID: id, ParentChannelID: channelID, Archived: false, Locked: false}, nil
				}
				cr.getByIDFunc = func(ctx context.Context, id uuid.UUID) (*models.Channel, error) {
					return &models.Channel{ID: id, ServerID: &serverID}, nil
				}
				sr.getMemberFunc = func(ctx context.Context, sID, uID uuid.UUID) (*models.Member, error) {
					return &models.Member{ServerID: sID, UserID: uID}, nil
				}
				tr.isMemberFunc = func(ctx context.Context, tID, uID uuid.UUID) (bool, error) {
					return true, nil
				}
			},
			wantErr: nil,
		},
		{
			name:     "thread not found",
			threadID: threadID,
			authorID: userID,
			content:  "Hello",
			setupMocks: func(tr *mockThreadRepository, cr *mockChannelRepoForThread, sr *mockServerRepoForThread) {
				tr.getByIDFunc = func(ctx context.Context, id uuid.UUID) (*models.Thread, error) {
					return nil, nil
				}
			},
			wantErr: ErrThreadNotFound,
		},
		{
			name:     "thread archived",
			threadID: threadID,
			authorID: userID,
			content:  "Hello",
			setupMocks: func(tr *mockThreadRepository, cr *mockChannelRepoForThread, sr *mockServerRepoForThread) {
				tr.getByIDFunc = func(ctx context.Context, id uuid.UUID) (*models.Thread, error) {
					return &models.Thread{ID: id, ParentChannelID: channelID, Archived: true}, nil
				}
			},
			wantErr: ErrThreadArchived,
		},
		{
			name:     "thread locked",
			threadID: threadID,
			authorID: userID,
			content:  "Hello",
			setupMocks: func(tr *mockThreadRepository, cr *mockChannelRepoForThread, sr *mockServerRepoForThread) {
				tr.getByIDFunc = func(ctx context.Context, id uuid.UUID) (*models.Thread, error) {
					return &models.Thread{ID: id, ParentChannelID: channelID, Archived: false, Locked: true}, nil
				}
			},
			wantErr: ErrThreadLocked,
		},
		{
			name:     "adds user as member if not already",
			threadID: threadID,
			authorID: userID,
			content:  "Hello",
			setupMocks: func(tr *mockThreadRepository, cr *mockChannelRepoForThread, sr *mockServerRepoForThread) {
				addMemberCalled := false
				tr.getByIDFunc = func(ctx context.Context, id uuid.UUID) (*models.Thread, error) {
					return &models.Thread{ID: id, ParentChannelID: channelID, Archived: false, Locked: false}, nil
				}
				cr.getByIDFunc = func(ctx context.Context, id uuid.UUID) (*models.Channel, error) {
					return &models.Channel{ID: id, ServerID: &serverID}, nil
				}
				sr.getMemberFunc = func(ctx context.Context, sID, uID uuid.UUID) (*models.Member, error) {
					return &models.Member{ServerID: sID, UserID: uID}, nil
				}
				tr.isMemberFunc = func(ctx context.Context, tID, uID uuid.UUID) (bool, error) {
					return false, nil
				}
				tr.addMemberFunc = func(ctx context.Context, tID, uID uuid.UUID) error {
					addMemberCalled = true
					return nil
				}
				tr.createMessageFunc = func(ctx context.Context, tID, aID uuid.UUID, content string) (*models.ThreadMessage, error) {
					if !addMemberCalled {
						t.Error("expected addMember to be called before createMessage")
					}
					return &models.ThreadMessage{ID: uuid.New(), ThreadID: tID, AuthorID: aID, Content: content}, nil
				}
			},
			wantErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			threadRepo := &mockThreadRepository{}
			channelRepo := &mockChannelRepoForThread{}
			serverRepo := &mockServerRepoForThread{}
			eventBus := &mockEventBusForThread{}

			tt.setupMocks(threadRepo, channelRepo, serverRepo)

			svc := NewThreadService(threadRepo, channelRepo, serverRepo, eventBus)

			msg, err := svc.SendThreadMessage(ctx, tt.threadID, tt.authorID, tt.content)

			if tt.wantErr != nil {
				if err != tt.wantErr {
					t.Errorf("expected error %v, got %v", tt.wantErr, err)
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if msg == nil {
				t.Error("expected message to be returned")
			}
		})
	}
}

func TestThreadService_ArchiveThread(t *testing.T) {
	ctx := context.Background()
	threadID := uuid.New()
	ownerID := uuid.New()
	otherUserID := uuid.New()
	channelID := uuid.New()
	serverID := uuid.New()

	tests := []struct {
		name        string
		threadID    uuid.UUID
		requesterID uuid.UUID
		setupMocks  func(*mockThreadRepository, *mockChannelRepoForThread, *mockServerRepoForThread)
		wantErr     error
	}{
		{
			name:        "success as owner",
			threadID:    threadID,
			requesterID: ownerID,
			setupMocks: func(tr *mockThreadRepository, cr *mockChannelRepoForThread, sr *mockServerRepoForThread) {
				tr.getByIDFunc = func(ctx context.Context, id uuid.UUID) (*models.Thread, error) {
					return &models.Thread{ID: id, OwnerID: ownerID, ParentChannelID: channelID}, nil
				}
			},
			wantErr: nil,
		},
		{
			name:        "thread not found",
			threadID:    threadID,
			requesterID: ownerID,
			setupMocks: func(tr *mockThreadRepository, cr *mockChannelRepoForThread, sr *mockServerRepoForThread) {
				tr.getByIDFunc = func(ctx context.Context, id uuid.UUID) (*models.Thread, error) {
					return nil, nil
				}
			},
			wantErr: ErrThreadNotFound,
		},
		{
			name:        "not owner - checks server membership",
			threadID:    threadID,
			requesterID: otherUserID,
			setupMocks: func(tr *mockThreadRepository, cr *mockChannelRepoForThread, sr *mockServerRepoForThread) {
				tr.getByIDFunc = func(ctx context.Context, id uuid.UUID) (*models.Thread, error) {
					return &models.Thread{ID: id, OwnerID: ownerID, ParentChannelID: channelID}, nil
				}
				cr.getByIDFunc = func(ctx context.Context, id uuid.UUID) (*models.Channel, error) {
					return &models.Channel{ID: id, ServerID: &serverID}, nil
				}
				sr.getMemberFunc = func(ctx context.Context, sID, uID uuid.UUID) (*models.Member, error) {
					return &models.Member{ServerID: sID, UserID: uID}, nil // Has membership
				}
			},
			wantErr: nil, // Should succeed if user is server member (with MANAGE_THREADS perm in real impl)
		},
		{
			name:        "not owner and not member",
			threadID:    threadID,
			requesterID: otherUserID,
			setupMocks: func(tr *mockThreadRepository, cr *mockChannelRepoForThread, sr *mockServerRepoForThread) {
				tr.getByIDFunc = func(ctx context.Context, id uuid.UUID) (*models.Thread, error) {
					return &models.Thread{ID: id, OwnerID: ownerID, ParentChannelID: channelID}, nil
				}
				cr.getByIDFunc = func(ctx context.Context, id uuid.UUID) (*models.Channel, error) {
					return &models.Channel{ID: id, ServerID: &serverID}, nil
				}
				sr.getMemberFunc = func(ctx context.Context, sID, uID uuid.UUID) (*models.Member, error) {
					return nil, nil // Not a member
				}
			},
			wantErr: ErrNotServerMember,
		},
		{
			name:        "not owner and DM channel",
			threadID:    threadID,
			requesterID: otherUserID,
			setupMocks: func(tr *mockThreadRepository, cr *mockChannelRepoForThread, sr *mockServerRepoForThread) {
				tr.getByIDFunc = func(ctx context.Context, id uuid.UUID) (*models.Thread, error) {
					return &models.Thread{ID: id, OwnerID: ownerID, ParentChannelID: channelID}, nil
				}
				cr.getByIDFunc = func(ctx context.Context, id uuid.UUID) (*models.Channel, error) {
					return &models.Channel{ID: id, ServerID: nil, Type: models.ChannelTypeDM}, nil // DM channel has no server
				}
			},
			wantErr: ErrNotThreadOwner,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			threadRepo := &mockThreadRepository{}
			channelRepo := &mockChannelRepoForThread{}
			serverRepo := &mockServerRepoForThread{}
			eventBus := &mockEventBusForThread{}

			tt.setupMocks(threadRepo, channelRepo, serverRepo)

			svc := NewThreadService(threadRepo, channelRepo, serverRepo, eventBus)

			err := svc.ArchiveThread(ctx, tt.threadID, tt.requesterID)

			if tt.wantErr != nil {
				if err != tt.wantErr {
					t.Errorf("expected error %v, got %v", tt.wantErr, err)
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestThreadService_GetChannelThreads(t *testing.T) {
	ctx := context.Background()
	channelID := uuid.New()
	userID := uuid.New()
	serverID := uuid.New()

	tests := []struct {
		name            string
		channelID       uuid.UUID
		requesterID     uuid.UUID
		includeArchived bool
		setupMocks      func(*mockThreadRepository, *mockChannelRepoForThread, *mockServerRepoForThread)
		wantErr         error
		wantCount       int
	}{
		{
			name:            "success - active only",
			channelID:       channelID,
			requesterID:     userID,
			includeArchived: false,
			setupMocks: func(tr *mockThreadRepository, cr *mockChannelRepoForThread, sr *mockServerRepoForThread) {
				cr.getByIDFunc = func(ctx context.Context, id uuid.UUID) (*models.Channel, error) {
					return &models.Channel{ID: id, ServerID: &serverID}, nil
				}
				sr.getMemberFunc = func(ctx context.Context, sID, uID uuid.UUID) (*models.Member, error) {
					return &models.Member{ServerID: sID, UserID: uID}, nil
				}
				tr.getActiveByChannelIDFunc = func(ctx context.Context, cID uuid.UUID) ([]*models.Thread, error) {
					return []*models.Thread{
						{ID: uuid.New(), ParentChannelID: cID, Name: "Thread 1"},
						{ID: uuid.New(), ParentChannelID: cID, Name: "Thread 2"},
					}, nil
				}
			},
			wantErr:   nil,
			wantCount: 2,
		},
		{
			name:            "success - include archived",
			channelID:       channelID,
			requesterID:     userID,
			includeArchived: true,
			setupMocks: func(tr *mockThreadRepository, cr *mockChannelRepoForThread, sr *mockServerRepoForThread) {
				cr.getByIDFunc = func(ctx context.Context, id uuid.UUID) (*models.Channel, error) {
					return &models.Channel{ID: id, ServerID: &serverID}, nil
				}
				sr.getMemberFunc = func(ctx context.Context, sID, uID uuid.UUID) (*models.Member, error) {
					return &models.Member{ServerID: sID, UserID: uID}, nil
				}
				tr.getByChannelIDFunc = func(ctx context.Context, cID uuid.UUID) ([]*models.Thread, error) {
					return []*models.Thread{
						{ID: uuid.New(), ParentChannelID: cID, Name: "Thread 1", Archived: false},
						{ID: uuid.New(), ParentChannelID: cID, Name: "Thread 2", Archived: true},
						{ID: uuid.New(), ParentChannelID: cID, Name: "Thread 3", Archived: false},
					}, nil
				}
			},
			wantErr:   nil,
			wantCount: 3,
		},
		{
			name:            "channel not found",
			channelID:       channelID,
			requesterID:     userID,
			includeArchived: false,
			setupMocks: func(tr *mockThreadRepository, cr *mockChannelRepoForThread, sr *mockServerRepoForThread) {
				cr.getByIDFunc = func(ctx context.Context, id uuid.UUID) (*models.Channel, error) {
					return nil, nil
				}
			},
			wantErr: ErrChannelNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			threadRepo := &mockThreadRepository{}
			channelRepo := &mockChannelRepoForThread{}
			serverRepo := &mockServerRepoForThread{}
			eventBus := &mockEventBusForThread{}

			tt.setupMocks(threadRepo, channelRepo, serverRepo)

			svc := NewThreadService(threadRepo, channelRepo, serverRepo, eventBus)

			threads, err := svc.GetChannelThreads(ctx, tt.channelID, tt.requesterID, tt.includeArchived)

			if tt.wantErr != nil {
				if err != tt.wantErr {
					t.Errorf("expected error %v, got %v", tt.wantErr, err)
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if len(threads) != tt.wantCount {
				t.Errorf("expected %d threads, got %d", tt.wantCount, len(threads))
			}
		})
	}
}

// Helper function
func intPtr(i int) *int {
	return &i
}
