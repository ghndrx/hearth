package co.hndrx.hearth.core.models

import com.squareup.moshi.Json
import com.squareup.moshi.JsonClass

@JsonClass(generateAdapter = true)
data class Message(
    @Json(name = "id") val id: String,
    @Json(name = "channel_id") val channelId: String,
    @Json(name = "author_id") val authorId: String,
    @Json(name = "content") val content: String,
    @Json(name = "type") val type: Int = 0,
    @Json(name = "reply_to") val replyTo: String? = null,
    @Json(name = "pinned") val pinned: Boolean = false,
    @Json(name = "tts") val tts: Boolean = false,
    @Json(name = "flags") val flags: Int = 0,
    @Json(name = "created_at") val createdAt: String,
    @Json(name = "edited_at") val editedAt: String? = null,
    @Json(name = "author") val author: User? = null,
    @Json(name = "attachments") val attachments: List<Attachment> = emptyList(),
    @Json(name = "reactions") val reactions: List<Reaction> = emptyList(),
)

@JsonClass(generateAdapter = true)
data class Attachment(
    @Json(name = "id") val id: String,
    @Json(name = "filename") val filename: String,
    @Json(name = "content_type") val contentType: String? = null,
    @Json(name = "size") val size: Long = 0,
    @Json(name = "url") val url: String,
    @Json(name = "proxy_url") val proxyUrl: String? = null,
    @Json(name = "width") val width: Int? = null,
    @Json(name = "height") val height: Int? = null,
)

@JsonClass(generateAdapter = true)
data class Reaction(
    @Json(name = "emoji") val emoji: String,
    @Json(name = "count") val count: Int = 0,
    @Json(name = "me") val me: Boolean = false,
)
