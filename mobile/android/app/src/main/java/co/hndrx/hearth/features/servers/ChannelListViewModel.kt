package co.hndrx.hearth.features.servers

import androidx.lifecycle.SavedStateHandle
import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import co.hndrx.hearth.core.gateway.GatewayClient
import co.hndrx.hearth.core.models.Channel
import co.hndrx.hearth.core.models.ChannelType
import co.hndrx.hearth.core.network.HearthApiService
import co.hndrx.hearth.core.storage.CachedChannel
import co.hndrx.hearth.core.storage.ChannelDao
import dagger.hilt.android.lifecycle.HiltViewModel
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.launch
import javax.inject.Inject

data class ChannelListUiState(
    val channels: List<Channel> = emptyList(),
    val categories: List<Channel> = emptyList(),
    val textChannels: List<Channel> = emptyList(),
    val voiceChannels: List<Channel> = emptyList(),
    val isLoading: Boolean = false,
    val error: String? = null,
)

@HiltViewModel
class ChannelListViewModel @Inject constructor(
    savedStateHandle: SavedStateHandle,
    private val apiService: HearthApiService,
    private val channelDao: ChannelDao,
    private val gatewayClient: GatewayClient,
) : ViewModel() {

    private val serverId: String = savedStateHandle.get<String>("serverId") ?: ""

    private val _uiState = MutableStateFlow(ChannelListUiState())
    val uiState: StateFlow<ChannelListUiState> = _uiState.asStateFlow()

    init {
        loadChannels()
        observeCachedChannels()
        observeGatewayEvents()
    }

    private fun observeCachedChannels() {
        viewModelScope.launch {
            channelDao.getChannelsForServer(serverId).collect { cachedChannels ->
                if (cachedChannels.isNotEmpty()) {
                    val channels = cachedChannels.map { cached ->
                        Channel(
                            id = cached.id,
                            serverId = cached.serverId,
                            name = cached.name,
                            type = ChannelType.valueOf(cached.type),
                            topic = cached.topic,
                            position = cached.position,
                            nsfw = cached.nsfw,
                            lastMessageId = cached.lastMessageId,
                            parentId = cached.parentId,
                            createdAt = cached.createdAt,
                        )
                    }
                    processChannels(channels)
                }
            }
        }
    }

    private fun observeGatewayEvents() {
        viewModelScope.launch {
            gatewayClient.events.collect { event ->
                when (event.type) {
                    "CHANNEL_CREATE", "CHANNEL_UPDATE", "CHANNEL_DELETE" -> {
                        loadChannels()
                    }
                    "MESSAGE_CREATE" -> {
                        // Mark channel as having unread messages
                        // Full implementation would extract channelId from event.data
                    }
                }
            }
        }
    }

    fun loadChannels() {
        viewModelScope.launch {
            _uiState.update { it.copy(isLoading = true) }
            try {
                val channels = apiService.getChannels(serverId)
                _uiState.update {
                    it.copy(channels = channels, isLoading = false, error = null)
                }
                processChannels(channels)
                cacheChannels(channels)
            } catch (e: Exception) {
                _uiState.update {
                    it.copy(isLoading = false, error = e.message ?: "Failed to load channels")
                }
            }
        }
    }

    private fun processChannels(channels: List<Channel>) {
        val categories = channels.filter { it.type == ChannelType.CATEGORY }
        val textChannels = channels.filter { it.type == ChannelType.TEXT }
        val voiceChannels = channels.filter { it.type == ChannelType.VOICE }

        _uiState.update {
            it.copy(
                categories = categories,
                textChannels = textChannels,
                voiceChannels = voiceChannels,
            )
        }
    }

    private suspend fun cacheChannels(channels: List<Channel>) {
        val cached = channels.map { channel ->
            CachedChannel(
                id = channel.id,
                serverId = channel.serverId,
                name = channel.name,
                type = channel.type.name,
                topic = channel.topic,
                position = channel.position,
                nsfw = channel.nsfw,
                lastMessageId = channel.lastMessageId,
                parentId = channel.parentId,
                createdAt = channel.createdAt,
            )
        }
        channelDao.insertChannels(cached)
    }
}
