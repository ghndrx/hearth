package co.hndrx.hearth.core.push

import com.squareup.moshi.Json
import com.squareup.moshi.JsonClass

/**
 * Request body for registering a device push token with the Hearth backend.
 * POST /api/v1/users/@me/push-tokens
 */
@JsonClass(generateAdapter = true)
data class PushTokenRequest(
    @Json(name = "token") val token: String,
    @Json(name = "platform") val platform: String, // "ios" | "android"
    @Json(name = "device_id") val deviceId: String,
    @Json(name = "app_version") val appVersion: String,
)

/**
 * Response from GET /api/v1/users/@me/notifications/unread-count
 */
@JsonClass(generateAdapter = true)
data class UnreadNotificationCount(
    @Json(name = "count") val count: Int,
)
