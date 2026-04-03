package co.hndrx.hearth.core.models

import com.squareup.moshi.Json
import com.squareup.moshi.JsonClass

@JsonClass(generateAdapter = true)
data class Channel(
    @Json(name = "id") val id: String,
    @Json(name = "server_id") val serverId: String? = null,
    @Json(name = "name") val name: String,
    @Json(name = "type") val type: ChannelType = ChannelType.TEXT,
    @Json(name = "topic") val topic: String? = null,
    @Json(name = "position") val position: Int = 0,
    @Json(name = "nsfw") val nsfw: Boolean = false,
    @Json(name = "last_message_id") val lastMessageId: String? = null,
    @Json(name = "parent_id") val parentId: String? = null,
    @Json(name = "created_at") val createdAt: String? = null,
)

@JsonClass(generateAdapter = false)
enum class ChannelType {
    @Json(name = "text") TEXT,
    @Json(name = "voice") VOICE,
    @Json(name = "category") CATEGORY,
    @Json(name = "dm") DM,
    @Json(name = "group_dm") GROUP_DM,
    @Json(name = "announcement") ANNOUNCEMENT,
}
