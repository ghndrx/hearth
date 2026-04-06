package co.hndrx.hearth.core.models

import com.squareup.moshi.Json
import com.squareup.moshi.JsonClass

@JsonClass(generateAdapter = true)
data class User(
    @Json(name = "id") val id: String,
    @Json(name = "username") val username: String,
    @Json(name = "email") val email: String,
    @Json(name = "display_name") val displayName: String? = null,
    @Json(name = "avatar") val avatar: String? = null,
    @Json(name = "avatar_url") val avatarUrl: String? = null,
    @Json(name = "status") val status: UserStatus = UserStatus.OFFLINE,
    @Json(name = "flags") val flags: Int = 0,
    @Json(name = "created_at") val createdAt: String? = null,
)

@JsonClass(generateAdapter = false)
enum class UserStatus {
    @Json(name = "online") ONLINE,
    @Json(name = "idle") IDLE,
    @Json(name = "dnd") DND,
    @Json(name = "offline") OFFLINE,
}
