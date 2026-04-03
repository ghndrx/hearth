package co.hndrx.hearth.core.models

import com.squareup.moshi.Json
import com.squareup.moshi.JsonClass

@JsonClass(generateAdapter = true)
data class GatewayMessage(
    @Json(name = "op") val op: Int,
    @Json(name = "d") val d: Any? = null,
    @Json(name = "s") val s: Int? = null,
    @Json(name = "t") val t: String? = null,
)

object GatewayOpcode {
    const val DISPATCH = 0
    const val HEARTBEAT = 1
    const val IDENTIFY = 2
    const val PRESENCE_UPDATE = 3
    const val VOICE_STATE_UPDATE = 4
    const val RESUME = 6
    const val RECONNECT = 7
    const val INVALID_SESSION = 9
    const val HELLO = 10
    const val HEARTBEAT_ACK = 11
}

data class GatewayEvent(
    val type: String,
    val data: Any?,
    val sequence: Int?,
)

enum class ConnectionState {
    DISCONNECTED,
    CONNECTING,
    CONNECTED,
    RECONNECTING,
}
