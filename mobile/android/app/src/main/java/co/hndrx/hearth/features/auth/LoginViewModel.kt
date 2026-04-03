package co.hndrx.hearth.features.auth

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import co.hndrx.hearth.core.auth.AuthManager
import dagger.hilt.android.lifecycle.HiltViewModel
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.launch
import javax.inject.Inject

data class LoginUiState(
    val email: String = "",
    val password: String = "",
    val mfaCode: String = "",
    val showMfa: Boolean = false,
    val isLoading: Boolean = false,
    val error: String? = null,
    val loginSuccess: Boolean = false,
)

@HiltViewModel
class LoginViewModel @Inject constructor(
    private val authManager: AuthManager,
) : ViewModel() {

    private val _uiState = MutableStateFlow(LoginUiState())
    val uiState: StateFlow<LoginUiState> = _uiState.asStateFlow()

    fun onEmailChanged(email: String) {
        _uiState.update { it.copy(email = email, error = null) }
    }

    fun onPasswordChanged(password: String) {
        _uiState.update { it.copy(password = password, error = null) }
    }

    fun onMfaCodeChanged(code: String) {
        _uiState.update { it.copy(mfaCode = code, error = null) }
    }

    fun login() {
        val state = _uiState.value
        if (state.email.isBlank() || state.password.isBlank()) {
            _uiState.update { it.copy(error = "Email and password are required") }
            return
        }

        viewModelScope.launch {
            _uiState.update { it.copy(isLoading = true, error = null) }

            val mfaCode = state.mfaCode.takeIf { it.isNotBlank() }
            val result = authManager.login(state.email, state.password, mfaCode)

            result.fold(
                onSuccess = {
                    _uiState.update { it.copy(isLoading = false, loginSuccess = true) }
                },
                onFailure = { throwable ->
                    val errorMessage = throwable.message ?: "Login failed"
                    // Check if MFA is required
                    val needsMfa = errorMessage.contains("mfa", ignoreCase = true)
                            || errorMessage.contains("two-factor", ignoreCase = true)
                    _uiState.update {
                        it.copy(
                            isLoading = false,
                            error = if (needsMfa) null else errorMessage,
                            showMfa = needsMfa || it.showMfa,
                        )
                    }
                },
            )
        }
    }

    fun clearError() {
        _uiState.update { it.copy(error = null) }
    }
}
