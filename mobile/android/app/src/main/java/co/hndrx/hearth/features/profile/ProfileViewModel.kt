package co.hndrx.hearth.features.profile

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import co.hndrx.hearth.core.auth.AuthManager
import co.hndrx.hearth.core.models.User
import dagger.hilt.android.lifecycle.HiltViewModel
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.launch
import javax.inject.Inject

data class ProfileUiState(
    val user: User? = null,
    val isLoading: Boolean = false,
    val error: String? = null,
)

@HiltViewModel
class ProfileViewModel @Inject constructor(
    private val authManager: AuthManager,
) : ViewModel() {

    private val _uiState = MutableStateFlow(ProfileUiState())
    val uiState: StateFlow<ProfileUiState> = _uiState.asStateFlow()

    init {
        observeCurrentUser()
        fetchCurrentUser()
    }

    private fun observeCurrentUser() {
        viewModelScope.launch {
            authManager.currentUser.collect { user ->
                _uiState.update { it.copy(user = user) }
            }
        }
    }

    fun fetchCurrentUser() {
        viewModelScope.launch {
            _uiState.update { it.copy(isLoading = true) }
            val result = authManager.fetchCurrentUser()
            result.fold(
                onSuccess = { user ->
                    _uiState.update { it.copy(user = user, isLoading = false, error = null) }
                },
                onFailure = { throwable ->
                    _uiState.update {
                        it.copy(isLoading = false, error = throwable.message ?: "Failed to load profile")
                    }
                },
            )
        }
    }

    fun logout() {
        authManager.logout()
    }
}
