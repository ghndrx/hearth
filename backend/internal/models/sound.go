package models

import "time"

type Sound struct {
	ID        string    `json:"id"`
	ServerID  string    `json:"server_id"`
	Name      string    `json:"name"`
	AudioURL  string    `json:"audio_url"`
	Emoji     string    `json:"emoji"`
	CreatedBy string    `json:"created_by"`
	CreatedAt time.Time `json:"created_at"`
}

type SoundPack struct {
	ID        string    `json:"id"`
	ServerID  string    `json:"server_id"`
	Name      string    `json:"name"`
	Sounds    []Sound   `json:"sounds"`
	CreatedAt time.Time `json:"created_at"`
}
