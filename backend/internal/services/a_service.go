package services // Package defined as "services" as requested

import (
	"context"
	"errors"
	"fmt"
	"net"
	"sync"
	"time"
)

// --- Domain / Models ---

// Represents a Discord-like Guild core structure
type Guild struct {
	ID    string
	Name  string
	MemberCount int
}

// Represents the handler for a specific connection
// This aligns with Discord's Gateway pattern
type GuildHandler interface {
	GetGuilds(ctx context.Context, limit int) ([]Guild, error)
}

// --- Interfaces ---

// Gateway defines the contract for connecting clients
// It mimics net.Conn properties for simplicity in testing
type Gateway interface {
	WriteJSON(v interface{}) error
	Close() error
	RemoteAddr() net.Addr
}

// UserService handles application-specific logic
type UserService interface {
	GetMemberGuilds(userID int64) ([]Guild, error)
}

// --- Services ---

// HearthService orchestrates the connection lifecycle
type HearthService struct {
	us UserService
	wg sync.WaitGroup
}

func NewHearthService(userSvc UserService) *HearthService {
	return &HearthService{
		us: userSvc,
	}
}

// ManageClient handles an incoming connection
// This is a simplified listener that accepts TCP connections
func (s *HearthService) ManageClient(ctx context.Context, conn Gateway) {
	defer s.wg.Done()

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	logger.Printf("[Service] New client connected from: %v", conn.RemoteAddr())

	// Treat the connection as a periodic heartbeat loop
	// In a real production service, this would upgrade to WebSocket here
	for {
		select {
		case <-ctx.Done():
			logger.Printf("[Service] Client %v context cancelled, closing.", conn.RemoteAddr())
			return
		default:
			if err := s.handleLoop(ctx, conn); err != nil {
				logger.Printf("[Service] Error with client %v: %v", conn.RemoteAddr(), err)
				return
			}
			time.Sleep(1 * time.Second) // Simulate polling duty cycle
		}
	}
}

// handleLoop performs the logic specific to the connection
func (s *HearthService) handleLoop(ctx context.Context, conn Gateway) error {
	// Determine remote IP to simulate "User ID"
	host, _, _ := net.SplitHostPort(conn.RemoteAddr().String())
	// Parse ID (if hex string)
	var userID int64
	fmt.Sscanf(host, "%d", &userID) // Basic conversion for demo

	// 1. Heartbeat Logic (Simulated)
	if err := conn.WriteJSON(map[string]string{"type": "heartbeat"}); err != nil {
		return fmt.Errorf("failed to send heartbeat: %w", err)
	}

	// 2. Fetch Application Logic
	guilds, err := s.us.GetMemberGuilds(userID)
	if err != nil {
		if errors.Is(err, context.Canceled) {
			return nil
		}
		return fmt.Errorf("failed to fetch guilds: %w", err)
	}

	// 3. Send data to client
	if err := conn.WriteJSON(map[string]interface{}{
		"type":    "guild_update",
		"user_id": userID,
		"data":    guilds,
	}); err != nil {
		return fmt.Errorf("failed to send data: %w", err)
	}

	return nil
}

// Serve starts the listener service
func (s *HearthService) Serve(ctx context.Context, address string) error {
	listener, err := net.Listen("tcp", address)
	if err != nil {
		return err
	}
	defer listener.Close()

	logger.Println("Hearth Service listening on", address)

	for {
		select {
		case <-ctx.Done():
			logger.Println("Hearth Service shutting down")
			return ctx.Err()
		default:
			conn, err := listener.Accept()
			if err != nil {
				// Handle non-critical accept errors
				if !errors.Is(err, context.Canceled) {
					return fmt.Errorf("accept error: %w", err)
				}
				return err
			}

			s.wg.Add(1)
			// Handle the client in a goroutine
			go s.ManageClient(ctx, &loggingWrapper{Gateway: conn})
		}
	}
}

// --- Adaptors / Implementations ---

type loggingWrapper struct {
	Gateway
}

func (l *loggingWrapper) WriteJSON(v interface{}) error {
	if err := l.Gateway.WriteJSON(v); err != nil {
		return err
	}
	logger.Printf("[Log] JSON sent to %s: %+v", l.RemoteAddr(), v)
	return nil
}

// In-memory implementation of UserService for demo purposes
type InMemoryUserService struct {
	guilds map[int64][]Guild
}

func NewInMemoryUserService() *InMemoryUserService {
	return &InMemoryUserService{
		guilds: map[int64][]Guild{
			1: {ID: "801", Name: "Community Hub", MemberCount: 120},
			2: {ID: "802", Name: "Dev Team", MemberCount: 5},
		},
	}
}

func (m *InMemoryUserService) GetMemberGuilds(userID int64) ([]Guild, error) {
	time.Sleep(10 * time.Millisecond) // Simulate DB latency

	gs, ok := m.guilds[userID]
	if !ok {
		return nil, fmt.Errorf("user not found")
	}
	return gs, nil
}