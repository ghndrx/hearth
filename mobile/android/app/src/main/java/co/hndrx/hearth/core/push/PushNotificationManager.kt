package co.hndrx.hearth.core.push

import android.Manifest
import android.app.NotificationChannel
import android.app.NotificationManager
import android.app.PendingIntent
import android.content.Context
import android.content.Intent
import android.content.pm.PackageManager
import android.os.Build
import androidx.core.app.NotificationCompat
import androidx.core.content.ContextCompat
import co.hndrx.hearth.BuildConfig
import co.hndrx.hearth.R
import co.hndrx.hearth.core.auth.AuthManager
import co.hndrx.hearth.core.network.HearthApiService
import com.google.android.gms.common.ConnectionResult
import com.google.android.gms.common.GoogleApiAvailability
import com.google.firebase.messaging.FirebaseMessaging
import dagger.hilt.android.qualifiers.ApplicationContext
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.launch
import kotlinx.coroutines.tasks.await
import javax.inject.Inject
import javax.inject.Singleton

/**
 * Manages FCM push notification lifecycle on Android:
 * - Initialization and FCM token acquisition
 * - Permission handling (Android 13+ POST_NOTIFICATIONS)
 * - Server-side push token registration
 * - Notification channel creation
 * - Badge count fetching
 */
@Singleton
class PushNotificationManager @Inject constructor(
    @ApplicationContext private val context: Context,
    private val apiService: HearthApiService,
    private val authManager: AuthManager,
) {
    companion object {
        /** Channel IDs — must match channels created in createNotificationChannels(). */
        const val CHANNEL_MESSAGES = "messages"
        const val CHANNEL_MENTIONS = "mentions"
        const val CHANNEL_DMS = "dms"
        const val CHANNEL_ACTIVITY = "activity"

        private const val CHANNEL_MESSAGES_NAME = "Messages"
        private const val CHANNEL_MENTIONS_NAME = "Mentions"
        private const val CHANNEL_DMS_NAME = "Direct Messages"
        private const val CHANNEL_ACTIVITY_NAME = "Server Activity"
    }

    private val scope = CoroutineScope(SupervisorJob() + Dispatchers.IO)

    private val _fcmToken = MutableStateFlow<String?>(null)
    val fcmToken: StateFlow<String?> = _fcmToken.asStateFlow()

    private val _permissionGranted = MutableStateFlow(false)
    val permissionGranted: StateFlow<Boolean> = _permissionGranted.asStateFlow()

    private val _unreadCount = MutableStateFlow(0)
    val unreadCount: StateFlow<Int> = _unreadCount.asStateFlow()

    /** Call once from HearthApplication.onCreate() */
    fun initialize() {
        createNotificationChannels()
        refreshPermissionState()
    }

    // MARK: - Permission Handling

    /**
     * Returns true if POST_NOTIFICATIONS permission is already granted.
     * Call this before requesting to determine if you need to show rationale.
     */
    fun hasNotificationPermission(): Boolean {
        return if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.TIRAMISU) {
            ContextCompat.checkSelfPermission(
                context,
                Manifest.permission.POST_NOTIFICATIONS
            ) == PackageManager.PERMISSION_GRANTED
        } else {
            true // Permission not needed pre-Android 13
        }
    }

    /**
     * Refreshes the internal permission state from the system.
     * Call after permission result is received.
     */
    fun refreshPermissionState() {
        _permissionGranted.value = hasNotificationPermission()
    }

    /**
     * Fetches the FCM registration token and registers it with the backend.
     * Call after:
     *   1. User has granted POST_NOTIFICATIONS (Android 13+)
     *   2. Google Play Services are available
     *   3. User is authenticated
     */
    fun registerIfAuthenticated() {
        if (!authManager.isAuthenticated.value) return
        scope.launch {
            try {
                val token = fetchAndStoreToken()
                if (token != null) {
                    sendTokenToServer(token)
                }
            } catch (e: Exception) {
                // Log but don't crash — push is non-critical
                e.printStackTrace()
            }
        }
    }

    /**
     * Deletes the current FCM token from both FCM and the backend.
     * Call on logout.
     */
    fun unregister() {
        scope.launch {
            try {
                _fcmToken.value?.let { token ->
                    apiService.deletePushToken(token)
                }
                FirebaseMessaging.getInstance().deleteToken().await()
                _fcmToken.value = null
            } catch (e: Exception) {
                e.printStackTrace()
            }
        }
    }

    // MARK: - Badge / Unread Count

    /** Fetches the current unread notification count from the backend. */
    suspend fun refreshUnreadCount() {
        try {
            val response = apiService.getUnreadNotificationCount()
            _unreadCount.value = response.count
        } catch (e: Exception) {
            e.printStackTrace()
        }
    }

    // MARK: - FCM Token

    /**
     * Fetches a fresh FCM token and caches it in _fcmToken.
     * Returns null if Play Services are unavailable.
     */
    private suspend fun fetchAndStoreToken(): String? {
        val playAvailable = GoogleApiAvailability.getInstance()
            .isGooglePlayServicesAvailable(context)
        if (playAvailable != ConnectionResult.SUCCESS) {
            return null
        }

        return try {
            val token = FirebaseMessaging.getInstance().token.await()
            _fcmToken.value = token
            token
        } catch (e: Exception) {
            e.printStackTrace()
            null
        }
    }

    /**
     * Sends the FCM token to the Hearth backend.
     * Idempotent — safe to call multiple times; server will upsert.
     */
    private suspend fun sendTokenToServer(token: String) {
        val request = PushTokenRequest(
            token = token,
            platform = "android",
            deviceId = getOrCreateDeviceId(),
            appVersion = BuildConfig.VERSION_NAME,
        )
        apiService.registerPushToken(request)
    }

    /** Returns a stable device UUID, creating one on first call. */
    @Suppress("HardwareIds")
    private fun getOrCreateDeviceId(): String {
        val prefs = context.getSharedPreferences("hearth_push_prefs", Context.MODE_PRIVATE)
        var deviceId = prefs.getString("device_id", null)
        if (deviceId == null) {
            deviceId = java.util.UUID.randomUUID().toString()
            prefs.edit().putString("device_id", deviceId).apply()
        }
        return deviceId
    }

    // MARK: - Notification Channels

    private fun createNotificationChannels() {
        val notificationManager = context
            .getSystemService(Context.NOTIFICATION_SERVICE) as NotificationManager

        val channels = listOf(
            NotificationChannel(
                CHANNEL_MESSAGES,
                CHANNEL_MESSAGES_NAME,
                NotificationManager.IMPORTANCE_HIGH,
            ).apply {
                description = "Notifications for new messages in text channels"
                enableVibration(true)
                enableLights(true)
            },
            NotificationChannel(
                CHANNEL_MENTIONS,
                CHANNEL_MENTIONS_NAME,
                NotificationManager.IMPORTANCE_HIGH,
            ).apply {
                description = "Notifications for @mentions and @everyone"
                enableVibration(true)
                enableLights(true)
            },
            NotificationChannel(
                CHANNEL_DMS,
                CHANNEL_DMS_NAME,
                NotificationManager.IMPORTANCE_DEFAULT,
            ).apply {
                description = "Notifications for direct messages"
                enableVibration(true)
            },
            NotificationChannel(
                CHANNEL_ACTIVITY,
                CHANNEL_ACTIVITY_NAME,
                NotificationManager.IMPORTANCE_LOW,
            ).apply {
                description = "Server events such as member joins and role changes"
                enableLights(false)
                enableVibration(false)
            },
        )

        channels.forEach { channel ->
            notificationManager.createNotificationChannel(channel)
        }
    }

    /**
     * Shows a notification locally (for testing or foreground notifications).
     * In production, notifications arrive via FCM; this is for foreground-only use.
     */
    fun showLocalNotification(
        channelId: String,
        title: String,
        body: String,
        notificationId: Int,
    ) {
        val intent = context.packageManager
            .getLaunchIntentForPackage(context.packageName)
            ?.apply {
                flags = Intent.FLAG_ACTIVITY_NEW_TASK or Intent.FLAG_ACTIVITY_CLEAR_TOP
            }

        val pendingIntent = PendingIntent.getActivity(
            context,
            0,
            intent,
            PendingIntent.FLAG_IMMUTABLE or PendingIntent.FLAG_UPDATE_CURRENT,
        )

        val notification = NotificationCompat.Builder(context, channelId)
            .setSmallIcon(R.drawable.ic_notification)
            .setContentTitle(title)
            .setContentText(body)
            .setAutoCancel(true)
            .setPriority(NotificationCompat.PRIORITY_HIGH)
            .setContentIntent(pendingIntent)
            .build()

        val manager = context.getSystemService(Context.NOTIFICATION_SERVICE) as NotificationManager
        manager.notify(notificationId, notification)
    }
}
