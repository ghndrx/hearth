package co.hndrx.hearth.app

import android.app.Application
import co.hndrx.hearth.core.push.PushNotificationManager
import dagger.hilt.android.HiltAndroidApp
import javax.inject.Inject

@HiltAndroidApp
class HearthApplication : Application() {

    @Inject
    lateinit var pushNotificationManager: PushNotificationManager

    override fun onCreate() {
        super.onCreate()
        // Initialize notification channels and push notification manager.
        // Token fetching and server registration are deferred until the
        // user authenticates (see AuthManager / MainActivity).
        pushNotificationManager.initialize()
    }
}
