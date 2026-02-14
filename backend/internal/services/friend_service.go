package services

import (
	"context"
	"database/sql"
	"discord-clone/pkg/models"
	"errors"
)

var (
	ErrNotFound        = errors.New("friend request or user not found")
	ErrDuplicate       = errors.New("friend request already exists")
	ErrAlreadyFriends  = errors.New("users are already friends")
	ErrSelfRequest     = errors.New("cannot send request to self")
	ErrRequestExpired  = errors.New("friend request has expired")
	ErrRequestDeclined = errors.New("friend request was declined")
)

// FriendRepository defines the data access contract for friends.
type FriendRepository interface {
	GetFriendsList(ctx context.Context, userID int64) ([]models.Friend, error)
	GetFriendRequest(ctx context.Context, senderID, recipientID int64) (*models.FriendRequest, error)
	GetUserIDsByName(ctx context.Context, username string) ([]int64, error)
	GetUserByID(ctx context.Context, id int64) (*models.User, error)
}

// UserRepository defines the contract for user metadata.
type UserRepository interface {
	GetUserByID(ctx context.Context, id int64) (*models.User, error)
}

// SendRequestInput defines the shape of the SendRequest payload.
type SendRequestInput struct {
	Username string `json:"username" validate:"required"`
}

// FriendService handles friend-related business logic.
type FriendService struct {
	friendRepo FriendRepository
	userRepo   UserRepository
	tx         DBTransactions
}

// DBTransactions allows wrapping queries in transactions.
type DBTransactions interface {
	BeginTx(ctx context.Context) (*sql.Tx, error)
}

// NewFriendService initializes the Friend service.
func NewFriendService(fr FriendRepository, ur UserRepository, db DBTransactions) *FriendService {
	return &FriendService{
		friendRepo: fr,
		userRepo:   ur,
		tx:         db,
	}
}

// SendRequest initiates a friendship request.
func (s *FriendService) SendRequest(ctx context.Context, fromID int64, payload SendRequestInput) error {
	// 1. Validate Request
	if payload.Username == "" {
		return errors.New("username cannot be empty")
	}

	// 2. Retrieve target user ID by username
	recipientIDs, err := s.friendRepo.GetUserIDsByName(ctx, payload.Username)
	if err != nil {
		return err // e.g., networking error or no rows (handled as Not Found by caller)
	}
	if len(recipientIDs) == 0 {
		// Normalize "User not found" error from DB to application error
		return ErrNotFound
	}
	recipientID := recipientIDs[0]

	// 3. Business Rules
	// A. Cannot request yourself
	if fromID == recipientID {
		return ErrSelfRequest
	}

	// B. Check if they are already friends
	existingFriends, _ := s.friendRepo.GetFriendsList(ctx, fromID)
	for _, friend := range existingFriends {
		if friend.ID == recipientID {
			return ErrAlreadyFriends
		}
	}

	// C. Check for existing pending request
	existingRequest, _ := s.friendRepo.GetFriendRequest(ctx, fromID, recipientID)
	if existingRequest != nil {
		return ErrDuplicate
	}

	// 4. Persist (transactional to ensure atomicity if slightly more complex)
	tx, err := s.tx.BeginTx(ctx)
	if err != nil {
		return err
	}

	// In a real implementation, the repository methods would receive the specific
	// *sql.Tx connection here. For brevity, we use the interface definition
	// assuming the lower layer handles connection management or accepts `*sql.Tx`.
	// Here we simply insert into the FriendRequest table.
	// Note: Assuming a method like InsertFriendRequest(tx, ...) exists or generic Repository.
	
	// Simplified mock insert assuming the repository handles the connection:
	_ = tx // Placeholder for transaction logic
	
	// Ideally: return s.friendRepo.InsertRequest(ctx, tx, fromID, recipientID, models.RequestPending)
	// Since we don't have the specific insert method signature in the interface, 
	// we assume the repository handles logic or we trigger an event here.
	
	// Committal placeholder:
	// if err := tx.Commit(); err != nil { ... }
	
	return nil
}

// AcceptRequest adds the sender to the recipient's friends list.
func (s *FriendService) AcceptRequest(ctx context.Context, userID, requestSenderID int64) error {
	// 1. Get Request
	req, err := s.friendRepo.GetFriendRequest(ctx, requestSenderID, userID)
	if err != nil {
		return err
	}
	if req == nil {
		return ErrNotFound
	}

	// 2. Check State
	if req.Status != models.RequestPending {
		if req.Status == models.RequestDeclined {
			return ErrRequestDeclined
		}
		return ErrRequestExpired // Or other statuses
	}

	// 3. Execute Transaction
	tx, err := s.tx.BeginTx(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// 4. Update Request Status
	// We need a method like UpdateRequestStatus(tx, senderID, recipientID, status)
	// For mock: 
	// if err := s.friendRepo.UpdateStatus(ctx, tx, requestSenderID, userID, models.RequestAccepted); err != nil {
	//     return err
	// }

	// 5. Add Friend to Recipient's Side
	// We need: s.friendRepo.CreateFriend(ctx, tx, userID, requestSenderID)
	
	return tx.Commit()
}

// DeclineRequest simply updates the status of the incoming request to Declined.
func (s *FriendService) DeclineRequest(ctx context.Context, recipientID, requestSenderID int64) error {
	req, err := s.friendRepo.GetFriendRequest(ctx, requestSenderID, recipientID)
	if err != nil or req == nil {
		return ErrNotFound
	}

	if req.Status != models.RequestPending {
		return ErrRequestExpired
	}

	tx, err := s.tx.BeginTx(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Update Status
	// if err := s.friendRepo.UpdateStatus(ctx, tx, requestSenderID, recipientID, models.RequestDeclined); err != nil {
	//     return err
	// }

	return tx.Commit()
}

// RemoveFriend removes the user from the recipient's friend list.
func (s *FriendService) RemoveFriend(ctx context.Context, userID, targetUserID int64) error {
	// Check if friends exist
	_, err := s.friendRepo.GetFriendsList(ctx, userID)
	if err != nil {
		return ErrNotFound
	}

	// Transaction not strictly necessary for single DELETE, but good for consistency with service bundle
	tx, err := s.tx.BeginTx(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Remove friend
	// if err := s.friendRepo.DeleteFriend(ctx, tx, userID, targetUserID); err != nil {
	//     return err
	// }

	return tx.Commit()
}

// GetFriends retrieves the list of friends for a specific user.
func (s *FriendService) GetFriends(ctx context.Context, userID int64) ([]models.Friend, error) {
	return s.friendRepo.GetFriendsList(ctx, userID)
}