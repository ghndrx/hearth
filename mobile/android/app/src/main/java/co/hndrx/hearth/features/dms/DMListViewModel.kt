package co.hndrx.hearth.features.dms

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import co.hndrx.hearth.core.models.Channel
import co.hndrx.hearth.core.network.HearthApiService
import dagger.hilt.android.lifecycle.HiltViewModel
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.launch
import javax.inject.Inject

data class DMListUiState(
    val channels: List<Channel> = emptyList(),
    val isLoading: Boolean = false,
    val error: String? = null,
)

@HiltViewModel
class DMListViewModel @Inject constructor(
    private val apiService: HearthApiService,
) : ViewModel() {

    private val _uiState = MutableStateFlow(DMListUiState())
    val uiState: StateFlow<DMListUiState> = _uiState.asStateFlow()

    init {
        loadDMs()
    }

    fun loadDMs() {
        viewModelScope.launch {
            _uiState.update { it.copy(isLoading = true) }
            try {
                val channels = apiService.getDMs()
                _uiState.update {
                    it.copy(channels = channels, isLoading = false, error = null)
                }
            } catch (e: Exception) {
                _uiState.update {
                    it.copy(isLoading = false, error = e.message ?: "Failed to load DMs")
                }
            }
        }
    }

    fun createDM(recipientId: String) {
        viewModelScope.launch {
            _uiState.update { it.copy(isLoading = true) }
            try {
                val channel = apiService.createDM(
                    co.hndrx.hearth.core.models.CreateDMRequest(recipientId = recipientId)
                )
                _uiState.update { state ->
                    state.copy(
                        channels = state.channels + channel,
                        isLoading = false,
                        error = null,
                    )
                }
            } catch (e: Exception) {
                _uiState.update {
                    it.copy(isLoading = false, error = e.message ?: "Failed to create DM")
                }
            }
        }
    }
}
