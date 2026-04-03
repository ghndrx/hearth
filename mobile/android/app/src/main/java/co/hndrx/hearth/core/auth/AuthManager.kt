package co.hndrx.hearth.core.auth

import android.content.Context
import android.content.SharedPreferences
import androidx.security.crypto.EncryptedSharedPreferences
import androidx.security.crypto.MasterKeys
import co.hndrx.hearth.core.models.AuthResponse
import co.hndrx.hearth.core.models.LoginRequest
import co.hndrx.hearth.core.models.RefreshTokenRequest
import co.hndrx.hearth.core.models.RegisterRequest
import co.hndrx.hearth.core.models.User
import co.hndrx.hearth.core.network.HearthApiService
import dagger.hilt.android.qualifiers.ApplicationContext
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import javax.inject.Inject
import javax.inject.Singleton

@Singleton
class AuthManager @Inject constructor(
    @ApplicationContext private val context: Context,
) {

    companion object {
        private const val PREFS_NAME = "hearth_auth_prefs"
        private const val KEY_ACCESS_TOKEN = "access_token"
        private const val KEY_REFRESH_TOKEN = "refresh_token"
        private const val KEY_USER_ID = "user_id"
    }

    private val masterKeyAlias = MasterKeys.getOrCreate(MasterKeys.AES256_GCM_SPEC)

    private val prefs: SharedPreferences by lazy {
        EncryptedSharedPreferences.create(
            PREFS_NAME,
            masterKeyAlias,
            context,
            EncryptedSharedPreferences.PrefKeyEncryptionScheme.AES256_SIV,
            EncryptedSharedPreferences.PrefValueEncryptionScheme.AES256_GCM,
        )
    }

    private val _isAuthenticated = MutableStateFlow(getAccessToken() != null)
    val isAuthenticated: StateFlow<Boolean> = _isAuthenticated.asStateFlow()

    private val _currentUser = MutableStateFlow<User?>(null)
    val currentUser: StateFlow<User?> = _currentUser.asStateFlow()

    /**
     * Late-initialized reference to the API service.
     * Set by AppModule after Retrofit is created to avoid circular dependency.
     */
    internal lateinit var apiService: HearthApiService

    fun saveTokens(accessToken: String, refreshToken: String) {
        prefs.edit()
            .putString(KEY_ACCESS_TOKEN, accessToken)
            .putString(KEY_REFRESH_TOKEN, refreshToken)
            .apply()
        _isAuthenticated.value = true
    }

    fun clearTokens() {
        prefs.edit()
            .remove(KEY_ACCESS_TOKEN)
            .remove(KEY_REFRESH_TOKEN)
            .remove(KEY_USER_ID)
            .apply()
        _isAuthenticated.value = false
        _currentUser.value = null
    }

    fun getAccessToken(): String? = prefs.getString(KEY_ACCESS_TOKEN, null)

    fun getRefreshToken(): String? = prefs.getString(KEY_REFRESH_TOKEN, null)

    suspend fun login(email: String, password: String, mfaCode: String? = null): Result<User> {
        return try {
            val response: AuthResponse = apiService.login(
                LoginRequest(email = email, password = password, mfaCode = mfaCode)
            )
            saveTokens(response.token, response.refreshToken)
            _currentUser.value = response.user
            Result.success(response.user)
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun register(email: String, username: String, password: String): Result<User> {
        return try {
            val response: AuthResponse = apiService.register(
                RegisterRequest(email = email, username = username, password = password)
            )
            saveTokens(response.token, response.refreshToken)
            _currentUser.value = response.user
            Result.success(response.user)
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun refreshToken(): Result<String> {
        val currentRefreshToken = getRefreshToken()
            ?: return Result.failure(IllegalStateException("No refresh token available"))
        return try {
            val response: AuthResponse = apiService.refreshToken(
                RefreshTokenRequest(refreshToken = currentRefreshToken)
            )
            saveTokens(response.token, response.refreshToken)
            _currentUser.value = response.user
            Result.success(response.token)
        } catch (e: Exception) {
            clearTokens()
            Result.failure(e)
        }
    }

    suspend fun fetchCurrentUser(): Result<User> {
        return try {
            val user = apiService.getCurrentUser()
            _currentUser.value = user
            Result.success(user)
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    fun logout() {
        clearTokens()
    }
}
