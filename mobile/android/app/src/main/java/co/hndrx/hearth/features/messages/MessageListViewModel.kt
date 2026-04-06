package co.hndrx.hearth.features.messages

import androidx.lifecycle.SavedStateHandle
import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import co.hndrx.hearth.core.gateway.GatewayClient
import co.hndrx.hearth.core.models.Message
import co.hndrx.hearth.core.network.HearthApiService
import co.hndrx.hearth.core.storage.CachedMessage
import co.hndrx.hearth.core.storage.MessageDao
import dagger.hilt.android.lifecycle.HiltViewModel
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.launch
import javax.inject.Inject

data class MessageListUiState(
    val messages: List<Message> = emptyList(),
    val isLoading: Boolean = false,
    val isSending: Boolean = false,
    val hasMoreMessages: Boolean = true,
    val error: String? = null,
)

@HiltViewModel
class MessageListViewModel @Inject constructor(
    savedStateHandle: SavedStateHandle,
    private val apiService: HearthApiService,
    private val messageDao: MessageDao,
    private val gatewayClient: GatewayClient,
) : ViewModel() {

    private val serverId: String = savedStateHandle.get<String>("serverId") ?: ""
    private val channelId: String = savedStateHandle.get<String>("channelId") ?: ""

    private val _uiState = MutableStateFlow(MessageListUiState())
    val uiState: StateFlow<MessageListUiState> = _uiState.asStateFlow()

    init {
        loadMessages()
        observeCachedMessages()
        observeGatewayEvents()
    }

    private fun observeCachedMessages() {
        viewModelScope.launch {
            messageDao.getMessagesForChannel(channelId).collect { cachedMessages ->
                if (cachedMessages.isNotEmpty()) {
                    val messages = cachedMessages.map { cached ->
                        Message(
                            id = cached.id,
                            channelId = cached.channelId,
                            authorId = cached.authorId,
                            content = cached.content,
                            type = cached.type,
                            replyTo = cached.replyTo,
                            pinned = cached.pinned,
                            createdAt = cached.createdAt,
                            editedAt = cached.editedAt,
                        )
                    }.sortedBy { it.createdAt }
                    _uiState.update { it.copy(messages = messages) }
                }
            }
        }
    }

    private fun observeGatewayEvents() {
        viewModelScope.launch {
            gatewayClient.events.collect { event ->
                when (event.type) {
                    "MESSAGE_CREATE" -> {
                        // Full implementation would extract channelId and verify this message belongs to us
                        // Then add it to the list if it does
                    }
                    "MESSAGE_UPDATE" -> {
                        // Update the message in the list
                    }
                    "MESSAGE_DELETE" -> {
                        // Remove the message from the list
                    }
                }
            }
        }
    }

    fun loadMessages() {
        viewModelScope.launch {
            _uiState.update { it.copy(isLoading = true) }
            try {
                val fetched = apiService.getMessages(serverId, channelId)
                _uiState.update {
                    it.copy(
                        messages = fetched.sortedBy { m -> m.createdAt },
                        isLoading = false,
                        hasMoreMessages = fetched.size >= 50,
                        error = null,
                    )
                }
                cacheMessages(fetched)
            } catch (e: Exception) {
                _uiState.update {
                    it.copy(isLoading = false, error = e.message ?: "Failed to load messages")
                }
            }
        }
    }

    fun loadOlderMessages() {
        val currentMessages = _uiState.value.messages
        if (currentMessages.isEmpty() || !_uiState.value.hasMoreMessages) return

        viewModelScope.launch {
            _uiState.update { it.copy(isLoading = true) }
            try {
                val oldestId = currentMessages.firstOrNull()?.id
                val older = apiService.getMessages(serverId, channelId, before = oldestId)
                val sorted = (older + currentMessages).distinctBy { it.id }.sortedBy { m -> m.createdAt }
                _uiState.update {
                    it.copy(
                        messages = sorted,
                        isLoading = false,
                        hasMoreMessages = older.size >= 50,
                    )
                }
            } catch (e: Exception) {
                _uiState.update {
                    it.copy(isLoading = false, error = e.message)
                }
            }
        }
    }

    fun sendMessage(content: String) {
        if (content.isBlank()) return

        viewModelScope.launch {
            _uiState.update { it.copy(isSending = true) }
            try {
                val message = apiService.sendMessage(
                    serverId = serverId,
                    channelId = channelId,
                    request = co.hndrx.hearth.core.models.SendMessageRequest(content = content),
                )
                _uiState.update { state ->
                    state.copy(
                        messages = state.messages + message,
                        isSending = false,
                        error = null,
                    )
                }
                messageDao.insertMessage(message.toCached())
            } catch (e: Exception) {
                _uiState.update {
                    it.copy(isSending = false, error = e.message ?: "Failed to send message")
                }
            }
        }
    }

    private fun cacheMessages(messages: List<Message>) {
        viewModelScope.launch {
            messageDao.insertMessages(messages.map { it.toCached() })
        }
    }

    private fun Message.toCached() = CachedMessage(
        id = id,
        channelId = channelId,
        authorId = authorId,
        authorUsername = author?.username,
        authorAvatarUrl = author?.avatarUrl,
        content = content,
        type = type,
        replyTo = replyTo,
        pinned = pinned,
        createdAt = createdAt,
        editedAt = editedAt,
    )
}
