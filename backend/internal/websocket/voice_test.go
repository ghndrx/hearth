package websocket

import (
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVoiceSignalConstants(t *testing.T) {
	assert.Equal(t, "VOICE_OFFER", VoiceSignalOffer)
	assert.Equal(t, "VOICE_ANSWER", VoiceSignalAnswer)
	assert.Equal(t, "VOICE_ICE_CANDIDATE", VoiceSignalICECandidate)
	assert.Equal(t, "VOICE_JOIN", VoiceSignalJoin)
	assert.Equal(t, "VOICE_LEAVE", VoiceSignalLeave)
	assert.Equal(t, "VOICE_SPEAKING", VoiceSignalSpeaking)
}

func TestVoiceStateEventConstants(t *testing.T) {
	assert.Equal(t, "VOICE_STATE_UPDATE", EventTypeVoiceStateUpdate)
	assert.Equal(t, "VOICE_SERVER_UPDATE", EventTypeVoiceServerUpdate)
}

func TestVoiceSignalMessage_Serialization(t *testing.T) {
	channelID := uuid.New()
	serverID := uuid.New()
	fromUser := uuid.New()
	toUser := uuid.New()

	msg := VoiceSignalMessage{
		Type:       VoiceSignalOffer,
		ChannelID:  channelID,
		ServerID:   serverID,
		FromUserID: fromUser,
		ToUserID:   toUser,
		Data:       json.RawMessage(`{"sdp":"test-sdp"}`),
	}

	bytes, err := json.Marshal(msg)
	require.NoError(t, err)

	var decoded VoiceSignalMessage
	err = json.Unmarshal(bytes, &decoded)
	require.NoError(t, err)

	assert.Equal(t, VoiceSignalOffer, decoded.Type)
	assert.Equal(t, channelID, decoded.ChannelID)
	assert.Equal(t, serverID, decoded.ServerID)
	assert.Equal(t, fromUser, decoded.FromUserID)
	assert.Equal(t, toUser, decoded.ToUserID)
}

func TestVoiceJoinData_Serialization(t *testing.T) {
	data := VoiceJoinData{
		ChannelID:    uuid.New(),
		ServerID:     uuid.New(),
		SelfMuted:    true,
		SelfDeafened: false,
	}

	bytes, err := json.Marshal(data)
	require.NoError(t, err)

	var decoded VoiceJoinData
	err = json.Unmarshal(bytes, &decoded)
	require.NoError(t, err)

	assert.Equal(t, data.ChannelID, decoded.ChannelID)
	assert.True(t, decoded.SelfMuted)
	assert.False(t, decoded.SelfDeafened)
}

func TestVoiceLeaveData_Serialization(t *testing.T) {
	data := VoiceLeaveData{
		ChannelID: uuid.New(),
		ServerID:  uuid.New(),
	}

	bytes, err := json.Marshal(data)
	require.NoError(t, err)

	var decoded VoiceLeaveData
	err = json.Unmarshal(bytes, &decoded)
	require.NoError(t, err)

	assert.Equal(t, data.ChannelID, decoded.ChannelID)
	assert.Equal(t, data.ServerID, decoded.ServerID)
}

func TestVoiceSpeakingData_Serialization(t *testing.T) {
	data := VoiceSpeakingData{
		UserID:    uuid.New(),
		ChannelID: uuid.New(),
		Speaking:  true,
		SSRC:      12345,
	}

	bytes, err := json.Marshal(data)
	require.NoError(t, err)

	var decoded VoiceSpeakingData
	err = json.Unmarshal(bytes, &decoded)
	require.NoError(t, err)

	assert.True(t, decoded.Speaking)
	assert.Equal(t, uint32(12345), decoded.SSRC)
}

func TestVoiceStateData_Serialization(t *testing.T) {
	displayName := "Test User"
	avatar := "https://example.com/avatar.png"

	data := VoiceStateData{
		UserID:       uuid.New(),
		Username:     "testuser",
		DisplayName:  &displayName,
		Avatar:       &avatar,
		ChannelID:    uuid.New(),
		ServerID:     uuid.New(),
		SelfMuted:    true,
		SelfDeafened: false,
		SelfVideo:    true,
		SelfStream:   false,
		Muted:        false,
		Deafened:     false,
		SessionID:    "session-123",
	}

	bytes, err := json.Marshal(data)
	require.NoError(t, err)

	var decoded VoiceStateData
	err = json.Unmarshal(bytes, &decoded)
	require.NoError(t, err)

	assert.Equal(t, "testuser", decoded.Username)
	assert.Equal(t, &displayName, decoded.DisplayName)
	assert.Equal(t, &avatar, decoded.Avatar)
	assert.True(t, decoded.SelfMuted)
	assert.True(t, decoded.SelfVideo)
	assert.Equal(t, "session-123", decoded.SessionID)
}

func TestVoiceOfferData_Serialization(t *testing.T) {
	data := VoiceOfferData{
		SDP:      "v=0\r\no=- 123456 2 IN IP4 127.0.0.1\r\n",
		ToUserID: uuid.New(),
	}

	bytes, err := json.Marshal(data)
	require.NoError(t, err)

	var decoded VoiceOfferData
	err = json.Unmarshal(bytes, &decoded)
	require.NoError(t, err)

	assert.Contains(t, decoded.SDP, "v=0")
	assert.Equal(t, data.ToUserID, decoded.ToUserID)
}

func TestVoiceAnswerData_Serialization(t *testing.T) {
	data := VoiceAnswerData{
		SDP:      "v=0\r\no=answer",
		ToUserID: uuid.New(),
	}

	bytes, err := json.Marshal(data)
	require.NoError(t, err)

	var decoded VoiceAnswerData
	err = json.Unmarshal(bytes, &decoded)
	require.NoError(t, err)

	assert.Equal(t, data.SDP, decoded.SDP)
}

func TestVoiceICECandidateData_Serialization(t *testing.T) {
	data := VoiceICECandidateData{
		Candidate:     "candidate:1 1 UDP 2130706431 192.168.1.1 5000 typ host",
		SDPMid:        "audio",
		SDPMLineIndex: 0,
		ToUserID:      uuid.New(),
	}

	bytes, err := json.Marshal(data)
	require.NoError(t, err)

	var decoded VoiceICECandidateData
	err = json.Unmarshal(bytes, &decoded)
	require.NoError(t, err)

	assert.Contains(t, decoded.Candidate, "candidate:")
	assert.Equal(t, "audio", decoded.SDPMid)
	assert.Equal(t, 0, decoded.SDPMLineIndex)
}

func TestNewVoiceSignalingService(t *testing.T) {
	hub := NewHub()

	// Pass nil for voiceRepo since we can't create a real one without DB
	vs := NewVoiceSignalingService(hub, nil)

	require.NotNil(t, vs)
	assert.NotNil(t, vs.channelPeers)
	assert.NotNil(t, vs.speakingStates)
}
