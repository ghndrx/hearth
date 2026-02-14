package services

import (
	"sync"
	"testing"
	"time"
)

func TestNewTypingService(t *testing.T) {
	s := NewTypingService()
	if s == nil {
		t.Fatal("Expected NewTypingService to return a service, got nil")
	}
	if s.eventChan == nil {
		t.Fatal("Event channel was not initialized")
	}
}

func TestStartTyping_FlushesToCache(t *testing.T) {
	s := NewTypingService()

	// Give the Go routine a moment to process the initial buffer
	time.Sleep(10 * time.Millisecond)

 users := s.GetTypingUsers("test-channel")
 if len(users) != 0 {
     t.Errorf("Expected empty users initially, got %v", users)
 }

 s.StartTyping("test-channel", "user-1")

 users = s.GetTypingUsers("test-channel")
 if len(users) != 1 {
 t.Errorf("Expected 1 user, got %d", len(users))
 }
}

func TestGetTypingUsers_FiltersStaleEvents(t *testing.T) {
	s := NewTypingService()

 time.Sleep(10 * time.Millisecond)

 // Start typing
 s.StartTyping("test-channel", "user-stale")

 // Immediately (within 5 seconds) get users
 time.Sleep(2 * time.Millisecond)
 users := s.GetTypingUsers("test-channel")
 if len(users) != 1 {
 t.Errorf("Expected user-stale to be typing, got %d", len(users))
 }

 // Wait 6 seconds to force a stale event
 time.Sleep(6 * time.Second)

 // User should be gone now
 users = s.GetTypingUsers("test-channel")
 if len(users) != 0 {
 t.Errorf("Expected no users after stale timeout, got %d", len(users))
 }
}

func TestConcurrency(t *testing.T) {
	s := NewTypingService()
	time.Sleep(10 * time.Millisecond)

	var wg sync.WaitGroup
	const numUsers = 100

	for i := 0; i < numUsers; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			s.StartTyping("test-channel", "user-"+string(rune('0'+id)))
		}(i)
	}
	wg.Wait()

 users := s.GetTypingUsers("test-channel")
 if len(users) != numUsers {
 t.Errorf("Expected %d users, got %d", numUsers, len(users))
 }

 for _, u := range users {
  if u == "user-5" { // Check if specific index is present
     found := false
     for i := 0; i < numUsers; i++ {
         if u == "user-"+string(rune('0'+i)) {
             found = true
             break
         }
     }
     if !found {
         t.Errorf("User %v missing from map", u)
     }
  }
 }
}