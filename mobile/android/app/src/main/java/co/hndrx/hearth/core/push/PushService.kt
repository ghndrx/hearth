package co.hndrx.hearth.core.push

import android.app.PendingIntent
import android.content.Intent
import android.os.Build
import androidx.core.app.NotificationCompat
import androidx.core.app.NotificationManagerCompat
import co.hndrx.hearth.R
import com.google.firebase.messaging.FirebaseMessagingService
import com.google.firebase.messaging.RemoteMessage
import dagger.hilt.android.AndroidEntryPoint
import javax.inject.Inject

/**
 * Firebase Cloud Messaging service for handling incoming push notifications.
 *
 * Handles:
 * - MESSAGE_CREATE: New message notifications
 * - MENTION: @mention / @everyone notifications
 * - DM_CREATE: Direct message notifications
 * - SERVER_ACTIVITY: Low-priority server event notifications
 *
 * Forward declarations (no-op for now — wired up when features land):
 * - Voice channel notifications
 * - Server invite notifications
 */
@AndroidEntryPoint
class PushService : FirebaseMessagingService() {

    @Inject
    lateinit var pushNotificationManager: PushNotificationManager

    override fun onNewToken(token: String) {
        super.onNewToken(token)
        // Re-register the new token with the backend
        pushNotificationManager.registerIfAuthenticated()
    }

    override fun onMessageReceived(remoteMessage: RemoteMessage) {
        super.onMessageReceived(remoteMessage)

        // Log but don't surface data-only messages (those are handled by the app directly)
        if (remoteMessage.data.isNotEmpty()) {
            handleDataMessage(remoteMessage.data)
        }

        remoteMessage.notification?.let { notification ->
            handleNotificationPayload(
                title = notification.title ?: return@let,
                body = notification.body ?: return@let,
                data = remoteMessage.data,
            )
        }
    }

    // MARK: - Data Messages

    /**
     * Handles silent data messages sent from the Hearth backend.
     * The `hearth` data map contains:
     *   - type: MESSAGE_CREATE | MENTION | DM_CREATE | SERVER_ACTIVITY
     *   - channelId: String
     *   - serverId: String?
     *   - messageId: String
     *   - senderId: String
     *   - senderName: String
     *   - senderAvatar: String?
     *   - content: String (truncated)
     *   - mentionCount: Int?
     */
    private fun handleDataMessage(data: Map<String, String>) {
        val type = data["type"] ?: return

        when (type) {
            "MESSAGE_CREATE" -> handleMessageCreate(data)
            "MENTION" -> handleMention(data)
            "DM_CREATE" -> handleDMCreate(data)
            "SERVER_ACTIVITY" -> handleServerActivity(data)
            else -> {
                // Unknown type — log and ignore
            }
        }
    }

    private fun handleMessageCreate(data: Map<String, String>) {
        val channelId = data["channelId"] ?: return
        val serverName = data["serverName"] ?: "Hearth"
        val senderName = data["senderName"] ?: "Someone"
        val content = data["content"] ?: ""

        showNotification(
            channelId = PushNotificationManager.CHANNEL_MESSAGES,
            title = "$serverName",
            body = "$senderName: $content",
            notificationId = channelId.hashCode(),
            data = data,
        )
    }

    private fun handleMention(data: Map<String, String>) {
        val channelId = data["channelId"] ?: return
        val serverName = data["serverName"] ?: "Hearth"
        val senderName = data["senderName"] ?: "Someone"
        val content = data["content"] ?: ""

        showNotification(
            channelId = PushNotificationManager.CHANNEL_MENTIONS,
            title = "$senderName mentioned you in $serverName",
            body = content,
            notificationId = ("mention_$channelId").hashCode(),
            data = data,
        )
    }

    private fun handleDMCreate(data: Map<String, String>) {
        val channelId = data["channelId"] ?: return
        val senderName = data["senderName"] ?: "Someone"
        val content = data["content"] ?: ""

        showNotification(
            channelId = PushNotificationManager.CHANNEL_DMS,
            title = senderName,
            body = content,
            notificationId = ("dm_$channelId").hashCode(),
            data = data,
        )
    }

    private fun handleServerActivity(data: Map<String, String>) {
        val serverName = data["serverName"] ?: "Hearth"
        val activityText = data["activityText"] ?: return

        showNotification(
            channelId = PushNotificationManager.CHANNEL_ACTIVITY,
            title = serverName,
            body = activityText,
            notificationId = ("activity_$serverName").hashCode(),
            data = data,
        )
    }

    // MARK: - Notification Payload (from APNs-style remote notification)

    private fun handleNotificationPayload(
        title: String,
        body: String,
        data: Map<String, String>,
    ) {
        val channelId = when (data["type"]) {
            "MENTION" -> PushNotificationManager.CHANNEL_MENTIONS
            "DM_CREATE" -> PushNotificationManager.CHANNEL_DMS
            "SERVER_ACTIVITY" -> PushNotificationManager.CHANNEL_ACTIVITY
            else -> PushNotificationManager.CHANNEL_MESSAGES
        }

        showNotification(
            channelId = channelId,
            title = title,
            body = body,
            notificationId = (title + body).hashCode(),
            data = data,
        )
    }

    // MARK: - Show Notification

    private fun showNotification(
        channelId: String,
        title: String,
        body: String,
        notificationId: Int,
        data: Map<String, String>,
    ) {
        val launchIntent = packageManager.getLaunchIntentForPackage(packageName)?.apply {
            flags = Intent.FLAG_ACTIVITY_NEW_TASK or Intent.FLAG_ACTIVITY_CLEAR_TOP
            // Pass notification data to the app via intent extras
            data.forEach { (key, value) ->
                putExtra("hearth_$key", value)
            }
        }

        val pendingIntentFlags = if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.M) {
            PendingIntent.FLAG_IMMUTABLE or PendingIntent.FLAG_UPDATE_CURRENT
        } else {
            PendingIntent.FLAG_UPDATE_CURRENT
        }

        val pendingIntent = PendingIntent.getActivity(
            this,
            notificationId,
            launchIntent,
            pendingIntentFlags,
        )

        val notification = NotificationCompat.Builder(this, channelId)
            .setSmallIcon(R.drawable.ic_notification)
            .setContentTitle(title)
            .setContentText(body)
            .setStyle(NotificationCompat.BigTextStyle().bigText(body))
            .setAutoCancel(true)
            .setPriority(NotificationCompat.PRIORITY_HIGH)
            .setContentIntent(pendingIntent)
            .build()

        try {
            NotificationManagerCompat.from(this).notify(notificationId, notification)
        } catch (e: SecurityException) {
            // User revoked notification permission — ignore
        }
    }
}
