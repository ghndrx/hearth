package co.hndrx.hearth.core.network

import co.hndrx.hearth.core.models.AuthResponse
import co.hndrx.hearth.core.models.Channel
import co.hndrx.hearth.core.models.CreateDMRequest
import co.hndrx.hearth.core.models.LoginRequest
import co.hndrx.hearth.core.models.Message
import co.hndrx.hearth.core.models.RefreshTokenRequest
import co.hndrx.hearth.core.models.RegisterRequest
import co.hndrx.hearth.core.models.SendMessageRequest
import co.hndrx.hearth.core.models.Server
import co.hndrx.hearth.core.models.User
import retrofit2.http.Body
import retrofit2.http.GET
import retrofit2.http.POST
import retrofit2.http.Path
import retrofit2.http.Query

interface HearthApiService {

    @POST("auth/login")
    suspend fun login(@Body request: LoginRequest): AuthResponse

    @POST("auth/register")
    suspend fun register(@Body request: RegisterRequest): AuthResponse

    @POST("auth/refresh")
    suspend fun refreshToken(@Body request: RefreshTokenRequest): AuthResponse

    @GET("users/@me")
    suspend fun getCurrentUser(): User

    @GET("users/@me/servers")
    suspend fun getServers(): List<Server>

    @GET("servers/{id}/channels")
    suspend fun getChannels(@Path("id") serverId: String): List<Channel>

    @GET("servers/{serverId}/channels/{channelId}/messages")
    suspend fun getMessages(
        @Path("serverId") serverId: String,
        @Path("channelId") channelId: String,
        @Query("before") before: String? = null,
        @Query("limit") limit: Int = 50,
    ): List<Message>

    @POST("servers/{serverId}/channels/{channelId}/messages")
    suspend fun sendMessage(
        @Path("serverId") serverId: String,
        @Path("channelId") channelId: String,
        @Body request: SendMessageRequest,
    ): Message

    @GET("users/@me/channels")
    suspend fun getDMs(): List<Channel>

    @POST("users/@me/channels")
    suspend fun createDM(@Body request: CreateDMRequest): Channel

    // Push Notification endpoints

    @POST("users/@me/push-tokens")
    suspend fun registerPushToken(@Body request: co.hndrx.hearth.core.push.PushTokenRequest)

    @HTTP(method = "DELETE", path = "users/@me/push-tokens/{token}", hasBody = false)
    suspend fun deletePushToken(@Path("token") token: String)

    @GET("users/@me/notifications/unread-count")
    suspend fun getUnreadNotificationCount(): co.hndrx.hearth.core.push.UnreadNotificationCount
}

/**
 * OkHttp interceptor that attaches the Bearer token to every outgoing request
 * and automatically retries on 401 by refreshing the token.
 */
class AuthInterceptor(
    private val authManager: co.hndrx.hearth.core.auth.AuthManager,
) : okhttp3.Interceptor {

    override fun intercept(chain: okhttp3.Interceptor.Chain): okhttp3.Response {
        val originalRequest = chain.request()

        // Skip auth header for login/register/refresh endpoints
        val path = originalRequest.url.encodedPath
        if (path.contains("auth/login") || path.contains("auth/register") || path.contains("auth/refresh")) {
            return chain.proceed(originalRequest)
        }

        val accessToken = authManager.getAccessToken()
        val authenticatedRequest = if (accessToken != null) {
            originalRequest.newBuilder()
                .header("Authorization", "Bearer $accessToken")
                .build()
        } else {
            originalRequest
        }

        val response = chain.proceed(authenticatedRequest)

        // If we get a 401, try refreshing the token and retry once
        if (response.code == 401 && accessToken != null) {
            response.close()

            val refreshResult = runRefreshTokenBlocking()
            if (refreshResult) {
                val newToken = authManager.getAccessToken()
                val retriedRequest = originalRequest.newBuilder()
                    .header("Authorization", "Bearer $newToken")
                    .build()
                return chain.proceed(retriedRequest)
            } else {
                // Refresh failed, clear tokens
                authManager.clearTokens()
            }
        }

        return response
    }

    /**
     * Synchronously refreshes the token. OkHttp interceptors run on OkHttp's
     * thread pool, so we block here.
     */
    private fun runRefreshTokenBlocking(): Boolean {
        return try {
            kotlinx.coroutines.runBlocking {
                authManager.refreshToken().isSuccess
            }
        } catch (_: Exception) {
            false
        }
    }
}
