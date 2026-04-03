package co.hndrx.hearth.features.servers

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import co.hndrx.hearth.core.gateway.GatewayClient
import co.hndrx.hearth.core.models.Server
import co.hndrx.hearth.core.network.HearthApiService
import co.hndrx.hearth.core.storage.CachedServer
import co.hndrx.hearth.core.storage.ServerDao
import dagger.hilt.android.lifecycle.HiltViewModel
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.launch
import javax.inject.Inject

data class ServerListUiState(
    val servers: List<Server> = emptyList(),
    val isLoading: Boolean = false,
    val error: String? = null,
    val unreadServerIds: Set<String> = emptySet(),
)

@HiltViewModel
class ServerListViewModel @Inject constructor(
    private val apiService: HearthApiService,
    private val serverDao: ServerDao,
    private val gatewayClient: GatewayClient,
) : ViewModel() {

    private val _uiState = MutableStateFlow(ServerListUiState())
    val uiState: StateFlow<ServerListUiState> = _uiState.asStateFlow()

    init {
        loadServers()
        observeCachedServers()
        observeGatewayEvents()
    }

    private fun observeCachedServers() {
        viewModelScope.launch {
            serverDao.getAllServers().collect { cachedServers ->
                if (cachedServers.isNotEmpty()) {
                    val servers = cachedServers.map { cached ->
                        Server(
                            id = cached.id,
                            name = cached.name,
                            iconUrl = cached.iconUrl,
                            bannerUrl = cached.bannerUrl,
                            description = cached.description,
                            ownerId = cached.ownerId,
                            createdAt = cached.createdAt,
                        )
                    }
                    _uiState.update { it.copy(servers = servers) }
                }
            }
        }
    }

    private fun observeGatewayEvents() {
        viewModelScope.launch {
            gatewayClient.events.collect { event ->
                when (event.type) {
                    "MESSAGE_CREATE" -> {
                        // Mark server as having unread messages
                        // In a full implementation, extract serverId from the event data
                    }
                    "GUILD_CREATE", "GUILD_UPDATE" -> {
                        loadServers()
                    }
                    "GUILD_DELETE" -> {
                        loadServers()
                    }
                }
            }
        }
    }

    fun loadServers() {
        viewModelScope.launch {
            _uiState.update { it.copy(isLoading = true) }
            try {
                val servers = apiService.getServers()
                _uiState.update {
                    it.copy(servers = servers, isLoading = false, error = null)
                }
                // Cache servers locally
                val cached = servers.map { server ->
                    CachedServer(
                        id = server.id,
                        name = server.name,
                        iconUrl = server.iconUrl,
                        bannerUrl = server.bannerUrl,
                        description = server.description,
                        ownerId = server.ownerId,
                        createdAt = server.createdAt,
                    )
                }
                serverDao.insertServers(cached)
            } catch (e: Exception) {
                _uiState.update {
                    it.copy(isLoading = false, error = e.message ?: "Failed to load servers")
                }
            }
        }
    }
}
