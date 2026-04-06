import SwiftUI

struct LoginView: View {
    @Environment(AuthManager.self) private var authManager
    @State private var email = ""
    @State private var password = ""
    @State private var mfaCode = ""
    @State private var showMFA = false
    @State private var showRegister = false
    @State private var isLoading = false
    @State private var errorMessage: String?

    var body: some View {
        NavigationStack {
            ScrollView {
                VStack(spacing: 24) {
                    // Logo / Header
                    VStack(spacing: 8) {
                        Image(systemName: "flame.fill")
                            .font(.system(size: 64))
                            .foregroundStyle(.orange)

                        Text("Hearth")
                            .font(.largeTitle.bold())

                        Text("Welcome back!")
                            .font(.subheadline)
                            .foregroundStyle(.secondary)
                    }
                    .padding(.top, 40)

                    // Error Banner
                    if let errorMessage {
                        HStack {
                            Image(systemName: "exclamationmark.triangle.fill")
                            Text(errorMessage)
                        }
                        .font(.callout)
                        .foregroundStyle(.white)
                        .padding()
                        .frame(maxWidth: .infinity)
                        .background(Color.red.opacity(0.85), in: RoundedRectangle(cornerRadius: 10))
                    }

                    // Form Fields
                    VStack(spacing: 16) {
                        VStack(alignment: .leading, spacing: 6) {
                            Text("Email")
                                .font(.subheadline.weight(.medium))
                            TextField("you@example.com", text: $email)
                                .textFieldStyle(.roundedBorder)
                                .textContentType(.emailAddress)
                                .keyboardType(.emailAddress)
                                .autocorrectionDisabled()
                                .textInputAutocapitalization(.never)
                        }

                        VStack(alignment: .leading, spacing: 6) {
                            Text("Password")
                                .font(.subheadline.weight(.medium))
                            SecureField("Password", text: $password)
                                .textFieldStyle(.roundedBorder)
                                .textContentType(.password)
                        }

                        if showMFA {
                            VStack(alignment: .leading, spacing: 6) {
                                Text("Two-Factor Code")
                                    .font(.subheadline.weight(.medium))
                                TextField("123456", text: $mfaCode)
                                    .textFieldStyle(.roundedBorder)
                                    .keyboardType(.numberPad)
                                    .textContentType(.oneTimeCode)
                            }
                            .transition(.move(edge: .top).combined(with: .opacity))
                        }
                    }
                    .padding(.horizontal)

                    // Login Button
                    Button {
                        Task { await performLogin() }
                    } label: {
                        Group {
                            if isLoading {
                                ProgressView()
                                    .tint(.white)
                            } else {
                                Text("Log In")
                                    .fontWeight(.semibold)
                            }
                        }
                        .frame(maxWidth: .infinity)
                        .frame(height: 44)
                    }
                    .buttonStyle(.borderedProminent)
                    .tint(.orange)
                    .disabled(email.isEmpty || password.isEmpty || isLoading)
                    .padding(.horizontal)

                    // Divider
                    HStack {
                        Rectangle().frame(height: 1).foregroundStyle(.quaternary)
                        Text("or")
                            .font(.caption)
                            .foregroundStyle(.secondary)
                        Rectangle().frame(height: 1).foregroundStyle(.quaternary)
                    }
                    .padding(.horizontal)

                    // OAuth Buttons
                    VStack(spacing: 12) {
                        OAuthButton(provider: "GitHub", iconName: "chevron.left.forwardslash.chevron.right") {
                            // TODO: Implement GitHub OAuth
                        }
                        OAuthButton(provider: "Google", iconName: "globe") {
                            // TODO: Implement Google OAuth
                        }
                        OAuthButton(provider: "Discord", iconName: "ellipsis.message.fill") {
                            // TODO: Implement Discord OAuth
                        }
                    }
                    .padding(.horizontal)

                    // Register Link
                    Button {
                        showRegister = true
                    } label: {
                        HStack(spacing: 4) {
                            Text("Don't have an account?")
                                .foregroundStyle(.secondary)
                            Text("Sign up")
                                .fontWeight(.semibold)
                        }
                        .font(.subheadline)
                    }
                    .padding(.top, 8)
                }
                .padding(.bottom, 32)
            }
            .navigationDestination(isPresented: $showRegister) {
                RegisterView()
            }
        }
    }

    private func performLogin() async {
        isLoading = true
        errorMessage = nil

        do {
            _ = try await authManager.login(email: email, password: password)
        } catch {
            errorMessage = error.localizedDescription
        }

        isLoading = false
    }
}

private struct OAuthButton: View {
    let provider: String
    let iconName: String
    let action: () -> Void

    var body: some View {
        Button(action: action) {
            HStack {
                Image(systemName: iconName)
                    .frame(width: 20)
                Text("Continue with \(provider)")
                    .fontWeight(.medium)
            }
            .frame(maxWidth: .infinity)
            .frame(height: 44)
        }
        .buttonStyle(.bordered)
    }
}

#Preview {
    LoginView()
        .environment(AuthManager())
}
