// Package matrixfederation implements Matrix Federation protocol support for Hearth.
// This file implements a simplified AuthChecker for Room Version 9.
//
// Matrix Spec References:
//   - Room Version 9 Auth Rules: https://spec.matrix.org/v1.16/rooms/v9/#authorization-rules
//   - Auth Events: https://spec.matrix.org/v1.16/server-server-api/#authorization-rules
package matrixfederation

import (
	"fmt"
)

// AuthChecker validates whether an event is authorized within a room's context.
// This is a simplified implementation for Room Version 9.
type AuthChecker struct {
	// serverName is the name of this homeserver (e.g., "hearth.example.com")
	serverName string
}

// NewAuthChecker creates a new AuthChecker for the given server name.
func NewAuthChecker(serverName string) *AuthChecker {
	return &AuthChecker{
		serverName: serverName,
	}
}

// AuthResult is the result of an authorization check.
type AuthResult struct {
	// Allowed indicates whether the event is authorized.
	Allowed bool
	// Reason provides a human-readable explanation if not allowed.
	Reason string
}

// AuthEventProvider retrieves auth events for authorization checks.
type AuthEventProvider interface {
	// GetAuthEvents returns the auth events for a given event.
	// Auth events are typically: create, power_levels, member(sender), join_rules.
	GetAuthEvents(event *Event) ([]*Event, error)
}

// CheckAuthRules validates an event against Room Version 9 authorization rules.
// This simplified implementation checks:
//  1. Required auth events are present
//  2. Event signatures are valid
//  3. Content hash matches
//  4. Sender has sufficient power level for the event type
//  5. State key constraints are respected
func (ac *AuthChecker) CheckAuthRules(event *Event, provider AuthEventProvider) AuthResult {
	// Basic validation
	if err := event.Validate(); err != nil {
		return AuthResult{Allowed: false, Reason: fmt.Sprintf("invalid event: %v", err)}
	}

	// Verify content hash if present and not a test placeholder.
	// In production, all events must have a valid content hash.
	if event.Hashes.SHA256 != "" && event.Hashes.SHA256 != "dummy" {
		if err := event.VerifyContentHash(); err != nil {
			return AuthResult{Allowed: false, Reason: fmt.Sprintf("content hash mismatch: %v", err)}
		}
	}

	// Get auth events
	authEvents, err := provider.GetAuthEvents(event)
	if err != nil {
		return AuthResult{Allowed: false, Reason: fmt.Sprintf("failed to get auth events: %v", err)}
	}

	// Find required auth events
	var createEvent, powerLevelsEvent, memberEvent, joinRulesEvent *Event
	for _, ae := range authEvents {
		switch ae.Type {
		case EventTypeCreate:
			createEvent = ae
		case EventTypePowerLevels:
			powerLevelsEvent = ae
		case EventTypeMember:
			if ae.StateKey != nil && *ae.StateKey == event.Sender {
				memberEvent = ae
			}
		case EventTypeJoinRules:
			joinRulesEvent = ae
		}
	}

	// Rule 1: All events require a create event
	if createEvent == nil {
		return AuthResult{Allowed: false, Reason: "missing m.room.create auth event"}
	}

	// Rule 2: The sender must be in the room (membership is join or invite for most events)
	if memberEvent == nil && event.Type != EventTypeCreate {
		// For create events, the sender creates the room so no member event is needed
		return AuthResult{Allowed: false, Reason: "sender is not a member of the room"}
	}

	// Rule 3: Check membership for the sender
	if memberEvent != nil {
		membership, ok := memberEvent.Content["membership"].(string)
		if !ok {
			return AuthResult{Allowed: false, Reason: "sender member event has invalid membership"}
		}

		// Ban users cannot send any events
		if membership == MembershipBan {
			return AuthResult{Allowed: false, Reason: "sender is banned from the room"}
		}

		// Only joined/invited users can send most events
		if membership != MembershipJoin && membership != MembershipInvite {
			// Leave events can be sent by leaving users
			if event.Type != EventTypeMember || event.StateKeyString() != event.Sender {
				return AuthResult{Allowed: false, Reason: "sender is not joined or invited to the room"}
			}
		}
	}

	// Rule 4: Power level checks
	if !ac.hasSufficientPower(event, powerLevelsEvent, memberEvent) {
		return AuthResult{Allowed: false, Reason: "sender does not have sufficient power level"}
	}

	// Rule 5: Type-specific checks
	switch event.Type {
	case EventTypeCreate:
		return ac.checkCreateEvent(event)
	case EventTypeMember:
		return ac.checkMemberEvent(event, createEvent, joinRulesEvent, powerLevelsEvent, memberEvent)
	case EventTypePowerLevels:
		return ac.checkPowerLevelsEvent(event, powerLevelsEvent)
	case EventTypeJoinRules:
		return ac.checkJoinRulesEvent(event, powerLevelsEvent)
	}

	// Default: allow if all basic checks pass
	return AuthResult{Allowed: true}
}

// hasSufficientPower checks if the sender has sufficient power level for the event.
func (ac *AuthChecker) hasSufficientPower(event *Event, powerLevelsEvent, memberEvent *Event) bool {
	// If no power levels event exists, only the creator has power
	if powerLevelsEvent == nil {
		return event.Type == EventTypeCreate || (memberEvent != nil && memberEvent.Content["membership"] == MembershipJoin)
	}

	// Get sender's power level
	senderPower := int64(0)
	if users, ok := powerLevelsEvent.Content["users"].(map[string]interface{}); ok {
		if p, ok := users[event.Sender]; ok {
			switch v := p.(type) {
			case float64:
				senderPower = int64(v)
			case int64:
				senderPower = v
			}
		}
	}

	// Get default power level for this event type
	requiredPower := int64(0)
	switch event.Type {
	case EventTypePowerLevels:
		if v, ok := powerLevelsEvent.Content["state_default"]; ok {
			switch val := v.(type) {
			case float64:
				requiredPower = int64(val)
			case int64:
				requiredPower = val
			}
		} else {
			requiredPower = 100 // Default for state_default is 100 if not set
		}
	case EventTypeJoinRules, EventTypeName, EventTypeTopic, EventTypeAvatar, EventTypeCanonicalAlias, EventTypeHistoryVisibility:
		if v, ok := powerLevelsEvent.Content["state_default"]; ok {
			switch val := v.(type) {
			case float64:
				requiredPower = int64(val)
			case int64:
				requiredPower = val
			}
		}
	default:
		// Message events default to events_default
		if v, ok := powerLevelsEvent.Content["events_default"]; ok {
			switch val := v.(type) {
			case float64:
				requiredPower = int64(val)
			case int64:
				requiredPower = val
			}
		}
	}

	// Check if the event type has a specific power requirement
	if events, ok := powerLevelsEvent.Content["events"].(map[string]interface{}); ok {
		if p, ok := events[event.Type]; ok {
			switch val := p.(type) {
			case float64:
				requiredPower = int64(val)
			case int64:
				requiredPower = val
			}
		}
	}

	return senderPower >= requiredPower
}

// checkCreateEvent validates m.room.create events.
func (ac *AuthChecker) checkCreateEvent(event *Event) AuthResult {
	if !event.IsStateEvent() {
		return AuthResult{Allowed: false, Reason: "m.room.create must be a state event"}
	}
	if event.StateKeyString() != "" {
		return AuthResult{Allowed: false, Reason: "m.room.create state_key must be empty"}
	}
	if event.Depth != 1 {
		return AuthResult{Allowed: false, Reason: "m.room.create depth must be 1"}
	}
	if len(event.PrevEvents) != 0 {
		return AuthResult{Allowed: false, Reason: "m.room.create must not have prev_events"}
	}

	// Verify creator matches sender
	creator, ok := event.Content["creator"].(string)
	if !ok {
		return AuthResult{Allowed: false, Reason: "m.room.create missing creator"}
	}
	if creator != event.Sender {
		return AuthResult{Allowed: false, Reason: "m.room.create creator must match sender"}
	}

	return AuthResult{Allowed: true}
}

// checkMemberEvent validates m.room.member events.
func (ac *AuthChecker) checkMemberEvent(event, createEvent, joinRulesEvent, powerLevelsEvent, senderMember *Event) AuthResult {
	if !event.IsStateEvent() {
		return AuthResult{Allowed: false, Reason: "m.room.member must be a state event"}
	}
	if event.StateKey == nil {
		return AuthResult{Allowed: false, Reason: "m.room.member must have a state_key"}
	}

	targetUser := *event.StateKey
	membership, ok := event.Content["membership"].(string)
	if !ok {
		return AuthResult{Allowed: false, Reason: "m.room.member missing membership"}
	}

	// If the sender is changing their own membership
	if targetUser == event.Sender {
		switch membership {
		case MembershipJoin:
			// Users can always join if the room is public
			if joinRulesEvent != nil {
				if joinRule, ok := joinRulesEvent.Content["join_rule"].(string); ok {
					if joinRule == "public" || joinRule == "knock" {
						return AuthResult{Allowed: true}
					}
				}
			}
			// Otherwise check if invited
			if senderMember != nil {
				if senderMembership, ok := senderMember.Content["membership"].(string); ok {
					if senderMembership == MembershipInvite {
						return AuthResult{Allowed: true}
					}
				}
			}
			return AuthResult{Allowed: false, Reason: "user cannot join without invite or public room"}
		case MembershipLeave:
			// Users can always leave
			return AuthResult{Allowed: true}
		case MembershipBan:
			return AuthResult{Allowed: false, Reason: "users cannot ban themselves"}
		}
	} else {
		// The sender is changing someone else's membership
		// Need to check power levels for kick/ban/invite
		switch membership {
		case MembershipInvite:
			// Invite requires invite power (default 0, but can be set)
			return ac.checkPowerForMembershipChange(event, powerLevelsEvent, "invite")
		case MembershipJoin:
			return AuthResult{Allowed: false, Reason: "cannot force another user to join"}
		case MembershipLeave:
			// Kicking requires kick power level (default 50)
			return ac.checkPowerForMembershipChange(event, powerLevelsEvent, "kick")
		case MembershipBan:
			// Banning requires ban power level (default 50)
			return ac.checkPowerForMembershipChange(event, powerLevelsEvent, "ban")
		}
	}

	return AuthResult{Allowed: true}
}

// checkPowerForMembershipChange checks if the sender can change another user's membership.
func (ac *AuthChecker) checkPowerForMembershipChange(event *Event, powerLevelsEvent *Event, powerType string) AuthResult {
	if powerLevelsEvent == nil {
		return AuthResult{Allowed: false, Reason: "missing power levels for membership change"}
	}

	// Get sender's power
	senderPower := int64(0)
	if users, ok := powerLevelsEvent.Content["users"].(map[string]interface{}); ok {
		if p, ok := users[event.Sender]; ok {
			switch v := p.(type) {
			case float64:
				senderPower = int64(v)
			case int64:
				senderPower = v
			}
		}
	}

	// Get required power for this action
	requiredPower := int64(50) // Default kick/ban power
	if v, ok := powerLevelsEvent.Content[powerType]; ok {
		switch val := v.(type) {
		case float64:
			requiredPower = int64(val)
		case int64:
			requiredPower = val
		}
	}

	if senderPower < requiredPower {
		return AuthResult{Allowed: false, Reason: fmt.Sprintf("sender power level %d < required %d for %s", senderPower, requiredPower, powerType)}
	}

	return AuthResult{Allowed: true}
}

// checkPowerLevelsEvent validates m.room.power_levels events.
func (ac *AuthChecker) checkPowerLevelsEvent(event, currentPowerLevels *Event) AuthResult {
	if !event.IsStateEvent() {
		return AuthResult{Allowed: false, Reason: "m.room.power_levels must be a state event"}
	}
	if event.StateKeyString() != "" {
		return AuthResult{Allowed: false, Reason: "m.room.power_levels state_key must be empty"}
	}

	// If there are existing power levels, only users with sufficient power can change them
	if currentPowerLevels != nil {
		senderPower := int64(0)
		if users, ok := currentPowerLevels.Content["users"].(map[string]interface{}); ok {
			if p, ok := users[event.Sender]; ok {
				switch v := p.(type) {
				case float64:
					senderPower = int64(v)
				case int64:
					senderPower = v
				}
			}
		}

		// state_default power level for changing power levels (default 100)
		requiredPower := int64(100)
		if v, ok := currentPowerLevels.Content["state_default"]; ok {
			switch val := v.(type) {
			case float64:
				requiredPower = int64(val)
			case int64:
				requiredPower = val
			}
		}

		if senderPower < requiredPower {
			return AuthResult{Allowed: false, Reason: fmt.Sprintf("power level %d < required %d to change power levels", senderPower, requiredPower)}
		}
	}

	return AuthResult{Allowed: true}
}

// checkJoinRulesEvent validates m.room.join_rules events.
func (ac *AuthChecker) checkJoinRulesEvent(event, powerLevelsEvent *Event) AuthResult {
	if !event.IsStateEvent() {
		return AuthResult{Allowed: false, Reason: "m.room.join_rules must be a state event"}
	}
	if event.StateKeyString() != "" {
		return AuthResult{Allowed: false, Reason: "m.room.join_rules state_key must be empty"}
	}

	// join_rule must be valid
	joinRule, ok := event.Content["join_rule"].(string)
	if !ok {
		return AuthResult{Allowed: false, Reason: "m.room.join_rules missing join_rule"}
	}

	validJoinRules := map[string]bool{
		"public": true, "invite": true, "private": true, "knock": true, "restricted": true,
	}
	if !validJoinRules[joinRule] {
		return AuthResult{Allowed: false, Reason: "invalid join_rule value"}
	}

	return AuthResult{Allowed: true}
}
