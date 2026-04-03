package co.hndrx.hearth.core.gateway

import co.hndrx.hearth.core.auth.AuthManager
import co.hndrx.hearth.core.models.ConnectionState
import co.hndrx.hearth.core.models.GatewayEvent
import co.hndrx.hearth.core.models.GatewayMessage
import co.hndrx.hearth.core.models.GatewayOpcode
import com.squareup.moshi.Moshi
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.Job
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.delay
import kotlinx.coroutines.flow.MutableSharedFlow
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.SharedFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asSharedFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.launch
import okhttp3.OkHttpClient
import okhttp3.Request
import okhttp3.Response
import okhttp3.WebSocket
import okhttp3.WebSocketListener
import javax.inject.Inject
import javax.inject.Singleton
import kotlin.math.min
import kotlin.random.Random

@Singleton
class GatewayClient @Inject constructor(
    private val okHttpClient: OkHttpClient,
    private val moshi: Moshi,
    private val authManager: AuthManager,
) {

    companion object {
        private const val MAX_RECONNECT_ATTEMPTS = 10
        private const val BASE_RECONNECT_DELAY_MS = 1000L
        private const val MAX_RECONNECT_DELAY_MS = 60_000L
    }

    private val scope = CoroutineScope(Dispatchers.IO + SupervisorJob())
    private val gatewayMessageAdapter = moshi.adapter(GatewayMessage::class.java)

    private var webSocket: WebSocket? = null
    private var heartbeatJob: Job? = null
    private var heartbeatIntervalMs: Long = 0L
    private var lastHeartbeatAcked: Boolean = true
    private var sequence: Int? = null
    private var sessionId: String? = null
    private var reconnectAttempts: Int = 0
    private var gatewayUrl: String = ""

    private val _connectionState = MutableStateFlow(ConnectionState.DISCONNECTED)
    val connectionState: StateFlow<ConnectionState> = _connectionState.asStateFlow()

    private val _events = MutableSharedFlow<GatewayEvent>(extraBufferCapacity = 256)
    val events: SharedFlow<GatewayEvent> = _events.asSharedFlow()

    fun connect(url: String) {
        gatewayUrl = url
        reconnectAttempts = 0
        openWebSocket()
    }

    fun disconnect() {
        heartbeatJob?.cancel()
        heartbeatJob = null
        webSocket?.close(1000, "Client disconnect")
        webSocket = null
        _connectionState.value = ConnectionState.DISCONNECTED
        sessionId = null
        sequence = null
    }

    private fun openWebSocket() {
        _connectionState.value = ConnectionState.CONNECTING

        val token = authManager.getAccessToken() ?: return
        val wsUrl = if (gatewayUrl.contains("?")) {
            "$gatewayUrl&token=$token"
        } else {
            "$gatewayUrl?token=$token"
        }

        val request = Request.Builder()
            .url(wsUrl)
            .build()

        webSocket = okHttpClient.newWebSocket(request, object : WebSocketListener() {
            override fun onOpen(webSocket: WebSocket, response: Response) {
                _connectionState.value = ConnectionState.CONNECTED
                reconnectAttempts = 0
            }

            override fun onMessage(webSocket: WebSocket, text: String) {
                handleMessage(text)
            }

            override fun onClosing(webSocket: WebSocket, code: Int, reason: String) {
                webSocket.close(code, reason)
            }

            override fun onClosed(webSocket: WebSocket, code: Int, reason: String) {
                heartbeatJob?.cancel()
                if (_connectionState.value != ConnectionState.DISCONNECTED) {
                    attemptReconnect()
                }
            }

            override fun onFailure(webSocket: WebSocket, t: Throwable, response: Response?) {
                heartbeatJob?.cancel()
                if (_connectionState.value != ConnectionState.DISCONNECTED) {
                    attemptReconnect()
                }
            }
        })
    }

    private fun handleMessage(text: String) {
        val message = try {
            gatewayMessageAdapter.fromJson(text) ?: return
        } catch (_: Exception) {
            return
        }

        // Track sequence number
        message.s?.let { sequence = it }

        when (message.op) {
            GatewayOpcode.HELLO -> {
                handleHello(message)
            }
            GatewayOpcode.HEARTBEAT -> {
                sendHeartbeat()
            }
            GatewayOpcode.HEARTBEAT_ACK -> {
                lastHeartbeatAcked = true
            }
            GatewayOpcode.DISPATCH -> {
                handleDispatch(message)
            }
            GatewayOpcode.RECONNECT -> {
                webSocket?.close(4000, "Server requested reconnect")
                attemptReconnect()
            }
            GatewayOpcode.INVALID_SESSION -> {
                val resumable = message.d as? Boolean ?: false
                if (!resumable) {
                    sessionId = null
                    sequence = null
                }
                scope.launch {
                    delay(Random.nextLong(1000, 5000))
                    openWebSocket()
                }
            }
        }
    }

    @Suppress("UNCHECKED_CAST")
    private fun handleHello(message: GatewayMessage) {
        val data = message.d as? Map<String, Any?> ?: return
        val interval = (data["heartbeat_interval"] as? Number)?.toLong() ?: 41250L
        heartbeatIntervalMs = interval
        startHeartbeat()

        // Send IDENTIFY or RESUME
        if (sessionId != null && sequence != null) {
            sendResume()
        } else {
            sendIdentify()
        }
    }

    @Suppress("UNCHECKED_CAST")
    private fun handleDispatch(message: GatewayMessage) {
        val eventType = message.t ?: return

        // Capture session ID from READY event
        if (eventType == "READY") {
            val data = message.d as? Map<String, Any?> ?: return
            sessionId = data["session_id"] as? String
        }

        scope.launch {
            _events.emit(
                GatewayEvent(
                    type = eventType,
                    data = message.d,
                    sequence = message.s,
                )
            )
        }
    }

    private fun startHeartbeat() {
        heartbeatJob?.cancel()
        lastHeartbeatAcked = true
        heartbeatJob = scope.launch {
            // Initial jitter delay
            delay(Random.nextLong(0, heartbeatIntervalMs))
            while (true) {
                if (!lastHeartbeatAcked) {
                    // Missed heartbeat ACK -- zombie connection
                    webSocket?.close(4009, "Heartbeat ACK missed")
                    attemptReconnect()
                    return@launch
                }
                sendHeartbeat()
                delay(heartbeatIntervalMs)
            }
        }
    }

    private fun sendHeartbeat() {
        lastHeartbeatAcked = false
        val payload = GatewayMessage(op = GatewayOpcode.HEARTBEAT, d = sequence)
        send(payload)
    }

    private fun sendIdentify() {
        val token = authManager.getAccessToken() ?: return
        val identifyData = mapOf(
            "token" to token,
            "properties" to mapOf(
                "os" to "Android",
                "browser" to "Hearth Android",
                "device" to "Hearth Android",
            ),
        )
        val payload = GatewayMessage(op = GatewayOpcode.IDENTIFY, d = identifyData)
        send(payload)
    }

    private fun sendResume() {
        val token = authManager.getAccessToken() ?: return
        val resumeData = mapOf(
            "token" to token,
            "session_id" to sessionId,
            "seq" to sequence,
        )
        val payload = GatewayMessage(op = GatewayOpcode.RESUME, d = resumeData)
        send(payload)
    }

    fun sendPresenceUpdate(status: String) {
        val presenceData = mapOf(
            "status" to status,
            "afk" to false,
        )
        val payload = GatewayMessage(op = GatewayOpcode.PRESENCE_UPDATE, d = presenceData)
        send(payload)
    }

    private fun send(message: GatewayMessage) {
        val json = try {
            gatewayMessageAdapter.toJson(message)
        } catch (_: Exception) {
            return
        }
        webSocket?.send(json)
    }

    private fun attemptReconnect() {
        if (reconnectAttempts >= MAX_RECONNECT_ATTEMPTS) {
            _connectionState.value = ConnectionState.DISCONNECTED
            return
        }

        _connectionState.value = ConnectionState.RECONNECTING
        reconnectAttempts++

        scope.launch {
            val baseDelay = BASE_RECONNECT_DELAY_MS * (1L shl min(reconnectAttempts - 1, 10))
            val cappedDelay = min(baseDelay, MAX_RECONNECT_DELAY_MS)
            val jitter = Random.nextLong(0, cappedDelay / 2)
            delay(cappedDelay + jitter)
            openWebSocket()
        }
    }
}
