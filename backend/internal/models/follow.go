package models

import "time"

type FollowedChannel struct {
    ID                 string    `json:"id"`
    ChannelID          string    `json:"channel_id"`
    FollowerChannelID  string    `json:"follower_channel_id"`
    CreatedAt         time.Time `json:"created_at"`
}
