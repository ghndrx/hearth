package co.hndrx.hearth.core.models

import com.squareup.moshi.Json
import com.squareup.moshi.JsonClass

@JsonClass(generateAdapter = true)
data class AuthResponse(
    @Json(name = "token") val token: String,
    @Json(name = "refresh_token") val refreshToken: String,
    @Json(name = "user") val user: User,
)

@JsonClass(generateAdapter = true)
data class LoginRequest(
    @Json(name = "email") val email: String,
    @Json(name = "password") val password: String,
    @Json(name = "mfa_code") val mfaCode: String? = null,
)

@JsonClass(generateAdapter = true)
data class RegisterRequest(
    @Json(name = "email") val email: String,
    @Json(name = "username") val username: String,
    @Json(name = "password") val password: String,
)

@JsonClass(generateAdapter = true)
data class RefreshTokenRequest(
    @Json(name = "refresh_token") val refreshToken: String,
)

@JsonClass(generateAdapter = true)
data class SendMessageRequest(
    @Json(name = "content") val content: String,
    @Json(name = "reply_to") val replyTo: String? = null,
    @Json(name = "tts") val tts: Boolean = false,
)

@JsonClass(generateAdapter = true)
data class CreateDMRequest(
    @Json(name = "recipient_id") val recipientId: String,
)
