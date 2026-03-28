package auth

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"hearth/internal/models"
)

func testProvider(t *testing.T, handler http.Handler) *FusionAuthProvider {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)

	p, err := NewFusionAuthProvider(&FusionAuthConfig{
		Host:          srv.URL,
		APIKey:        "test-api-key",
		ApplicationID: "test-app-id",
		ClientID:      "test-client-id",
		ClientSecret:  "test-client-secret",
		RedirectURI:   "http://localhost/callback",
	})
	require.NoError(t, err)
	return p
}

func TestNewFusionAuthProvider(t *testing.T) {
	t.Run("valid config", func(t *testing.T) {
		p, err := NewFusionAuthProvider(&FusionAuthConfig{
			Host:          "https://auth.example.com",
			APIKey:        "key",
			ApplicationID: "app-id",
		})
		require.NoError(t, err)
		assert.NotNil(t, p)
		assert.Equal(t, "https://auth.example.com", p.config.Host)
	})

	t.Run("missing host", func(t *testing.T) {
		_, err := NewFusionAuthProvider(&FusionAuthConfig{APIKey: "key", ApplicationID: "app"})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "host is required")
	})

	t.Run("missing api key", func(t *testing.T) {
		_, err := NewFusionAuthProvider(&FusionAuthConfig{Host: "https://a.com", ApplicationID: "app"})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "api_key is required")
	})

	t.Run("missing application id", func(t *testing.T) {
		_, err := NewFusionAuthProvider(&FusionAuthConfig{Host: "https://a.com", APIKey: "key"})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "application_id is required")
	})

	t.Run("trailing slash stripped", func(t *testing.T) {
		p, err := NewFusionAuthProvider(&FusionAuthConfig{
			Host:          "https://auth.example.com/",
			APIKey:        "key",
			ApplicationID: "app",
		})
		require.NoError(t, err)
		assert.Equal(t, "https://auth.example.com", p.config.Host)
	})
}

func TestFusionAuthProvider_Name(t *testing.T) {
	p := &FusionAuthProvider{}
	assert.Equal(t, "FusionAuth", p.Name())
}

func TestFusionAuthProvider_Type(t *testing.T) {
	p := &FusionAuthProvider{}
	assert.Equal(t, ProviderFusionAuth, p.Type())
}

func TestFusionAuthProvider_Register(t *testing.T) {
	userID := uuid.New()

	t.Run("success", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/api/user/registration", r.URL.Path)
			assert.Equal(t, http.MethodPost, r.Method)
			assert.Equal(t, "test-api-key", r.Header.Get("Authorization"))

			var body map[string]interface{}
			json.NewDecoder(r.Body).Decode(&body)
			user := body["user"].(map[string]interface{})
			assert.Equal(t, "test@example.com", user["email"])
			assert.Equal(t, "testuser", user["username"])

			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(faRegistrationResponse{
				Token:                  "access-token-123",
				RefreshToken:           "refresh-token-123",
				TokenExpirationInstant: 0,
				User: faUser{
					ID:       userID.String(),
					Email:    "test@example.com",
					Username: "testuser",
					Verified: true,
				},
			})
		})

		p := testProvider(t, handler)
		result, err := p.Register(context.Background(), &RegisterRequest{
			Email:    "test@example.com",
			Username: "testuser",
			Password: "password123",
		})

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, "access-token-123", result.AccessToken)
		assert.Equal(t, "refresh-token-123", result.RefreshToken)
		assert.Equal(t, "Bearer", result.TokenType)
		assert.Equal(t, userID, result.User.ID)
		assert.Equal(t, "testuser", result.User.Username)
	})

	t.Run("error response", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"fieldErrors": map[string]interface{}{
					"user.email": []map[string]string{
						{"message": "email already in use"},
					},
				},
			})
		})

		p := testProvider(t, handler)
		_, err := p.Register(context.Background(), &RegisterRequest{
			Email:    "test@example.com",
			Username: "testuser",
			Password: "password123",
		})

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "email already in use")
	})
}

func TestFusionAuthProvider_Login(t *testing.T) {
	userID := uuid.New()

	t.Run("success", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/api/login", r.URL.Path)
			assert.Equal(t, http.MethodPost, r.Method)

			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(faLoginResponse{
				Token:                  "access-token-456",
				RefreshToken:           "refresh-token-456",
				TokenExpirationInstant: 0,
				User: faUser{
					ID:       userID.String(),
					Email:    "test@example.com",
					Username: "testuser",
				},
			})
		})

		p := testProvider(t, handler)
		result, err := p.Login(context.Background(), &LoginRequest{
			Email:    "test@example.com",
			Password: "password123",
		})

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, "access-token-456", result.AccessToken)
		assert.Equal(t, userID, result.User.ID)
		assert.False(t, result.MFARequired)
	})

	t.Run("mfa required", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(242)
			json.NewEncoder(w).Encode(faLoginResponse{
				TwoFactorID: "mfa-token-789",
			})
		})

		p := testProvider(t, handler)
		result, err := p.Login(context.Background(), &LoginRequest{
			Email:    "test@example.com",
			Password: "password123",
		})

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.True(t, result.MFARequired)
		assert.Equal(t, "mfa-token-789", result.MFAToken)
	})

	t.Run("invalid credentials", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"generalErrors": []map[string]string{
					{"message": "Invalid login credentials"},
				},
			})
		})

		p := testProvider(t, handler)
		_, err := p.Login(context.Background(), &LoginRequest{
			Email:    "wrong@example.com",
			Password: "bad",
		})

		assert.Error(t, err)
	})
}

func TestFusionAuthProvider_Logout(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/api/logout", r.URL.Path)
			assert.Equal(t, http.MethodPost, r.Method)
			w.WriteHeader(http.StatusOK)
		})

		p := testProvider(t, handler)
		err := p.Logout(context.Background(), uuid.New())
		assert.NoError(t, err)
	})
}

func TestFusionAuthProvider_RefreshToken(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/oauth2/token", r.URL.Path)
			assert.Equal(t, "application/x-www-form-urlencoded", r.Header.Get("Content-Type"))

			r.ParseForm()
			assert.Equal(t, "refresh_token", r.FormValue("grant_type"))
			assert.Equal(t, "old-refresh-token", r.FormValue("refresh_token"))

			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(faTokenResponse{
				AccessToken:  "new-access-token",
				RefreshToken: "new-refresh-token",
				ExpiresIn:    3600,
				TokenType:    "Bearer",
			})
		})

		p := testProvider(t, handler)
		result, err := p.RefreshToken(context.Background(), "old-refresh-token")

		require.NoError(t, err)
		assert.Equal(t, "new-access-token", result.AccessToken)
		assert.Equal(t, "new-refresh-token", result.RefreshToken)
		assert.Equal(t, 3600, result.ExpiresIn)
		assert.Equal(t, "Bearer", result.TokenType)
	})
}

func TestFusionAuthProvider_ChangePassword(t *testing.T) {
	userID := uuid.New()

	t.Run("success", func(t *testing.T) {
		callCount := 0
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			callCount++
			if callCount == 1 {
				// Login verification
				assert.Equal(t, "/api/login", r.URL.Path)
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(faLoginResponse{
					User: faUser{ID: userID.String()},
				})
				return
			}
			// Password change
			assert.Equal(t, "/api/user/change-password/"+userID.String(), r.URL.Path)
			assert.Equal(t, http.MethodPut, r.Method)
			w.WriteHeader(http.StatusOK)
		})

		p := testProvider(t, handler)
		err := p.ChangePassword(context.Background(), userID, &ChangePasswordRequest{
			CurrentPassword: "oldpass",
			NewPassword:     "newpass123",
		})

		assert.NoError(t, err)
	})

	t.Run("wrong current password", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte(`{}`))
		})

		p := testProvider(t, handler)
		err := p.ChangePassword(context.Background(), userID, &ChangePasswordRequest{
			CurrentPassword: "wrong",
			NewPassword:     "newpass123",
		})

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "current password is incorrect")
	})
}

func TestFusionAuthProvider_RequestPasswordReset(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/api/user/forgot-password", r.URL.Path)
			assert.Equal(t, http.MethodPost, r.Method)

			var body map[string]interface{}
			json.NewDecoder(r.Body).Decode(&body)
			assert.Equal(t, "test@example.com", body["loginId"])

			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]string{"changePasswordId": "token-123"})
		})

		p := testProvider(t, handler)
		err := p.RequestPasswordReset(context.Background(), "test@example.com")
		assert.NoError(t, err)
	})
}

func TestFusionAuthProvider_ConfirmPasswordReset(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/api/user/change-password/reset-token-123", r.URL.Path)
			assert.Equal(t, http.MethodPost, r.Method)
			w.WriteHeader(http.StatusOK)
		})

		p := testProvider(t, handler)
		err := p.ConfirmPasswordReset(context.Background(), "reset-token-123", "newpassword")
		assert.NoError(t, err)
	})
}

func TestFusionAuthProvider_EnableMFA(t *testing.T) {
	userID := uuid.New()

	t.Run("success", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/api/user/two-factor/"+userID.String(), r.URL.Path)
			assert.Equal(t, http.MethodPost, r.Method)

			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(faMFAEnableResponse{
				SecretBase32: "JBSWY3DPEHPK3PXP",
			})
		})

		p := testProvider(t, handler)
		result, err := p.EnableMFA(context.Background(), userID)

		require.NoError(t, err)
		assert.Equal(t, "JBSWY3DPEHPK3PXP", result.Secret)
		assert.Contains(t, result.QRCodeURL, "otpauth://totp/Hearth:")
		assert.Contains(t, result.QRCodeURL, "JBSWY3DPEHPK3PXP")
	})
}

func TestFusionAuthProvider_VerifyMFA(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/api/two-factor/login", r.URL.Path)
			w.WriteHeader(http.StatusOK)
		})

		p := testProvider(t, handler)
		err := p.VerifyMFA(context.Background(), uuid.New(), "123456")
		assert.NoError(t, err)
	})

	t.Run("invalid code", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusForbidden)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"generalErrors": []map[string]string{
					{"message": "Invalid two-factor code"},
				},
			})
		})

		p := testProvider(t, handler)
		err := p.VerifyMFA(context.Background(), uuid.New(), "000000")
		assert.Error(t, err)
	})
}

func TestFusionAuthProvider_DisableMFA(t *testing.T) {
	userID := uuid.New()

	t.Run("success", func(t *testing.T) {
		callCount := 0
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			callCount++
			if callCount == 1 {
				// Get user to find MFA method
				assert.Equal(t, "/api/user/"+userID.String(), r.URL.Path)
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(faUserResponse{
					User: faUser{
						ID: userID.String(),
						TwoFactor: faMFA{
							Methods: []faMFAMethod{
								{ID: "mfa-method-1", Method: "authenticator"},
							},
						},
					},
				})
				return
			}
			// Delete MFA method
			assert.Contains(t, r.URL.Path, "/api/user/two-factor/"+userID.String())
			assert.Equal(t, "mfa-method-1", r.URL.Query().Get("methodId"))
			assert.Equal(t, http.MethodDelete, r.Method)
			w.WriteHeader(http.StatusOK)
		})

		p := testProvider(t, handler)
		err := p.DisableMFA(context.Background(), userID)
		assert.NoError(t, err)
	})

	t.Run("mfa not enabled", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(faUserResponse{
				User: faUser{
					ID:        userID.String(),
					TwoFactor: faMFA{Methods: []faMFAMethod{}},
				},
			})
		})

		p := testProvider(t, handler)
		err := p.DisableMFA(context.Background(), userID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "MFA is not enabled")
	})
}

func TestFusionAuthProvider_GetSessions(t *testing.T) {
	userID := uuid.New()
	sessionID := uuid.New()

	t.Run("success", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/api/jwt/refresh", r.URL.Path)
			assert.Equal(t, userID.String(), r.URL.Query().Get("userId"))
			assert.Equal(t, http.MethodGet, r.Method)

			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(faRefreshTokensResponse{
				RefreshTokens: []faRefreshToken{
					{
						ID:            sessionID.String(),
						InsertInstant: 1700000000000,
						MetaData: faMetaData{
							Device: struct {
								Name        string `json:"name"`
								Type        string `json:"type"`
								Description string `json:"description"`
							}{
								Name: "Chrome",
								Type: "DESKTOP",
							},
							OperatingSystemName: "Linux",
							UserAgent:           "Mozilla/5.0",
							LastAccessInstant:   1700001000000,
						},
					},
				},
			})
		})

		p := testProvider(t, handler)
		sessions, err := p.GetSessions(context.Background(), userID)

		require.NoError(t, err)
		require.Len(t, sessions, 1)
		assert.Equal(t, sessionID, sessions[0].ID)
		assert.Equal(t, userID, sessions[0].UserID)
		assert.Equal(t, models.DeviceTypeDesktop, sessions[0].DeviceType)
		assert.NotNil(t, sessions[0].OS)
		assert.Equal(t, "Linux", *sessions[0].OS)
	})

	t.Run("empty sessions", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(faRefreshTokensResponse{
				RefreshTokens: []faRefreshToken{},
			})
		})

		p := testProvider(t, handler)
		sessions, err := p.GetSessions(context.Background(), userID)
		require.NoError(t, err)
		assert.Empty(t, sessions)
	})
}

func TestFusionAuthProvider_RevokeSession(t *testing.T) {
	sessionID := uuid.New()

	t.Run("success", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/api/jwt/refresh/"+sessionID.String(), r.URL.Path)
			assert.Equal(t, http.MethodDelete, r.Method)
			w.WriteHeader(http.StatusOK)
		})

		p := testProvider(t, handler)
		err := p.RevokeSession(context.Background(), sessionID)
		assert.NoError(t, err)
	})
}

func TestFusionAuthProvider_RevokeAllSessions(t *testing.T) {
	userID := uuid.New()

	t.Run("success", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/api/jwt/refresh", r.URL.Path)
			assert.Equal(t, userID.String(), r.URL.Query().Get("userId"))
			assert.Equal(t, http.MethodDelete, r.Method)
			w.WriteHeader(http.StatusOK)
		})

		p := testProvider(t, handler)
		err := p.RevokeAllSessions(context.Background(), userID)
		assert.NoError(t, err)
	})
}

func TestFusionAuthProvider_GetUser(t *testing.T) {
	userID := uuid.New()

	t.Run("success", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/api/user/"+userID.String(), r.URL.Path)
			assert.Equal(t, http.MethodGet, r.Method)

			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(faUserResponse{
				User: faUser{
					ID:            userID.String(),
					Email:         "test@example.com",
					Username:      "testuser",
					FirstName:     "Test",
					LastName:      "User",
					ImageURL:      "https://example.com/avatar.png",
					Verified:      true,
					InsertInstant: 1700000000000,
					Data: faData{
						Bio:      "hello world",
						Pronouns: "they/them",
					},
					TwoFactor: faMFA{
						Methods: []faMFAMethod{
							{ID: "m1", Method: "authenticator"},
						},
					},
				},
			})
		})

		p := testProvider(t, handler)
		user, err := p.GetUser(context.Background(), userID)

		require.NoError(t, err)
		assert.Equal(t, userID, user.ID)
		assert.Equal(t, "test@example.com", user.Email)
		assert.Equal(t, "testuser", user.Username)
		assert.True(t, user.MFAEnabled)
		assert.True(t, user.Verified)
		assert.NotNil(t, user.DisplayName)
		assert.Equal(t, "Test User", *user.DisplayName)
		assert.NotNil(t, user.AvatarURL)
		assert.Equal(t, "https://example.com/avatar.png", *user.AvatarURL)
		assert.NotNil(t, user.Bio)
		assert.Equal(t, "hello world", *user.Bio)
	})

	t.Run("not found", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		})

		p := testProvider(t, handler)
		_, err := p.GetUser(context.Background(), userID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "user not found")
	})
}

func TestFusionAuthProvider_UpdateUser(t *testing.T) {
	userID := uuid.New()
	username := "newname"
	bio := "updated bio"

	t.Run("success", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/api/user/"+userID.String(), r.URL.Path)
			assert.Equal(t, http.MethodPut, r.Method)

			var body map[string]interface{}
			json.NewDecoder(r.Body).Decode(&body)
			userData := body["user"].(map[string]interface{})
			assert.Equal(t, "newname", userData["username"])
			dataFields := userData["data"].(map[string]interface{})
			assert.Equal(t, "updated bio", dataFields["bio"])

			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(faUserResponse{
				User: faUser{
					ID:       userID.String(),
					Username: "newname",
					Email:    "test@example.com",
					Data:     faData{Bio: "updated bio"},
				},
			})
		})

		p := testProvider(t, handler)
		user, err := p.UpdateUser(context.Background(), userID, &models.UpdateUserRequest{
			Username: &username,
			Bio:      &bio,
		})

		require.NoError(t, err)
		assert.Equal(t, "newname", user.Username)
		assert.NotNil(t, user.Bio)
		assert.Equal(t, "updated bio", *user.Bio)
	})
}

func TestFusionAuthProvider_DeleteUser(t *testing.T) {
	userID := uuid.New()

	t.Run("success", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/api/user/"+userID.String(), r.URL.Path)
			assert.Equal(t, "true", r.URL.Query().Get("hardDelete"))
			assert.Equal(t, http.MethodDelete, r.Method)
			w.WriteHeader(http.StatusOK)
		})

		p := testProvider(t, handler)
		err := p.DeleteUser(context.Background(), userID)
		assert.NoError(t, err)
	})
}

func TestFusionAuthProvider_GetAuthorizationURL(t *testing.T) {
	p, err := NewFusionAuthProvider(&FusionAuthConfig{
		Host:          "https://auth.example.com",
		APIKey:        "key",
		ApplicationID: "app-id",
		ClientID:      "client-id",
		RedirectURI:   "http://localhost/callback",
		Scopes:        []string{"openid", "profile", "email"},
	})
	require.NoError(t, err)

	url, err := p.GetAuthorizationURL(context.Background(), "state-123")
	require.NoError(t, err)

	assert.Contains(t, url, "https://auth.example.com/oauth2/authorize")
	assert.Contains(t, url, "client_id=client-id")
	assert.Contains(t, url, "response_type=code")
	assert.Contains(t, url, "state=state-123")
	assert.Contains(t, url, "redirect_uri=")
}

func TestFusionAuthProvider_HandleCallback(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/oauth2/token", r.URL.Path)
			r.ParseForm()
			assert.Equal(t, "authorization_code", r.FormValue("grant_type"))
			assert.Equal(t, "auth-code-123", r.FormValue("code"))

			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(faTokenResponse{
				AccessToken:  "callback-access-token",
				RefreshToken: "callback-refresh-token",
				ExpiresIn:    3600,
				TokenType:    "Bearer",
			})
		})

		p := testProvider(t, handler)
		result, err := p.HandleCallback(context.Background(), "auth-code-123", "state-123")

		require.NoError(t, err)
		assert.Equal(t, "callback-access-token", result.AccessToken)
		assert.Equal(t, "callback-refresh-token", result.RefreshToken)
		assert.Equal(t, 3600, result.ExpiresIn)
	})
}

func TestFusionAuthProvider_TenantHeader(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "tenant-abc", r.Header.Get("X-FusionAuth-TenantId"))
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{}`))
	})

	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)

	p, _ := NewFusionAuthProvider(&FusionAuthConfig{
		Host:          srv.URL,
		APIKey:        "key",
		ApplicationID: "app",
		TenantID:      "tenant-abc",
	})

	p.GetUser(context.Background(), uuid.New())
}

func TestMapFAUserToPublic(t *testing.T) {
	uid := uuid.New()

	t.Run("full user", func(t *testing.T) {
		faU := &faUser{
			ID:        uid.String(),
			Email:     "test@example.com",
			Username:  "testuser",
			FirstName: "Test",
			LastName:  "User",
			ImageURL:  "https://example.com/pic.png",
			Data: faData{
				DisplayName: "Display Name",
				Bio:         "my bio",
				Pronouns:    "she/her",
			},
		}

		pub := mapFAUserToPublic(faU)
		assert.Equal(t, uid, pub.ID)
		assert.Equal(t, "testuser", pub.Username)
		assert.Equal(t, "Display Name", *pub.DisplayName)
		assert.Equal(t, "https://example.com/pic.png", *pub.AvatarURL)
		assert.Equal(t, "my bio", *pub.Bio)
		assert.Equal(t, "she/her", *pub.Pronouns)
	})

	t.Run("minimal user", func(t *testing.T) {
		faU := &faUser{
			ID:       uid.String(),
			Username: "minimal",
		}

		pub := mapFAUserToPublic(faU)
		assert.Equal(t, "minimal", pub.Username)
		assert.Nil(t, pub.DisplayName)
		assert.Nil(t, pub.AvatarURL)
		assert.Nil(t, pub.Bio)
	})

	t.Run("name fallback", func(t *testing.T) {
		faU := &faUser{
			ID:        uid.String(),
			Username:  "user",
			FirstName: "John",
			LastName:  "Doe",
		}

		pub := mapFAUserToPublic(faU)
		assert.NotNil(t, pub.DisplayName)
		assert.Equal(t, "John Doe", *pub.DisplayName)
	})
}

func TestMapFAUserToModel(t *testing.T) {
	uid := uuid.New()

	faU := &faUser{
		ID:            uid.String(),
		Email:         "test@example.com",
		Username:      "testuser",
		Verified:      true,
		InsertInstant: 1700000000000,
		TwoFactor: faMFA{
			Methods: []faMFAMethod{{ID: "m1"}},
		},
	}

	user := mapFAUserToModel(faU)
	assert.Equal(t, uid, user.ID)
	assert.Equal(t, "test@example.com", user.Email)
	assert.True(t, user.MFAEnabled)
	assert.True(t, user.Verified)
}

func TestFusionAuthErrorParsing(t *testing.T) {
	t.Run("general error", func(t *testing.T) {
		resp := &http.Response{
			StatusCode: 400,
			Body: io.NopCloser(strings.NewReader(`{
				"generalErrors": [{"message": "Something went wrong"}]
			}`)),
		}
		err := fusionAuthError(resp)
		assert.Contains(t, err.Error(), "Something went wrong")
	})

	t.Run("field error", func(t *testing.T) {
		resp := &http.Response{
			StatusCode: 400,
			Body: io.NopCloser(strings.NewReader(`{
				"fieldErrors": {"user.email": [{"message": "required"}]}
			}`)),
		}
		err := fusionAuthError(resp)
		assert.Contains(t, err.Error(), "required")
	})

	t.Run("unknown error", func(t *testing.T) {
		resp := &http.Response{
			StatusCode: 500,
			Body:       io.NopCloser(strings.NewReader(`not json`)),
		}
		err := fusionAuthError(resp)
		assert.Contains(t, err.Error(), "status 500")
	})
}
