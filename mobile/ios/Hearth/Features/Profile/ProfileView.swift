import SwiftUI

struct ProfileView: View {
    @Environment(AuthManager.self) private var authManager
    @State private var showLogoutConfirmation = false

    var body: some View {
        NavigationStack {
            List {
                // User info section
                Section {
                    if let user = authManager.currentUser {
                        HStack(spacing: 14) {
                            // Avatar
                            AsyncImage(url: user.avatarUrl.flatMap { URL(string: $0) }) { image in
                                image.resizable().scaledToFill()
                            } placeholder: {
                                Circle()
                                    .fill(Color.orange.opacity(0.2))
                                    .overlay {
                                        Text(String(user.username.prefix(1)).uppercased())
                                            .font(.title2.bold())
                                            .foregroundStyle(.orange)
                                    }
                            }
                            .frame(width: 64, height: 64)
                            .clipShape(Circle())

                            VStack(alignment: .leading, spacing: 4) {
                                Text(user.displayName ?? user.username)
                                    .font(.title3.weight(.semibold))

                                Text("@\(user.username)")
                                    .font(.subheadline)
                                    .foregroundStyle(.secondary)

                                if let status = user.status {
                                    HStack(spacing: 4) {
                                        Circle()
                                            .fill(statusColor(status))
                                            .frame(width: 8, height: 8)
                                        Text(statusLabel(status))
                                            .font(.caption)
                                            .foregroundStyle(.secondary)
                                    }
                                }
                            }
                        }
                        .padding(.vertical, 4)
                    } else {
                        HStack {
                            ProgressView()
                            Text("Loading profile...")
                                .foregroundStyle(.secondary)
                        }
                    }
                }

                // Account details
                Section("Account") {
                    if let user = authManager.currentUser {
                        LabeledContent("Email", value: user.email ?? "Not set")
                        if let createdAt = user.createdAt {
                            LabeledContent("Joined") {
                                Text(createdAt, style: .date)
                            }
                        }
                    }
                }

                // Settings placeholders
                Section("Settings") {
                    NavigationLink {
                        Text("Appearance settings coming soon")
                            .foregroundStyle(.secondary)
                    } label: {
                        Label("Appearance", systemImage: "paintbrush.fill")
                    }

                    NavigationLink {
                        Text("Notification settings coming soon")
                            .foregroundStyle(.secondary)
                    } label: {
                        Label("Notifications", systemImage: "bell.fill")
                    }

                    NavigationLink {
                        Text("Privacy settings coming soon")
                            .foregroundStyle(.secondary)
                    } label: {
                        Label("Privacy & Safety", systemImage: "lock.fill")
                    }
                }

                // Logout
                Section {
                    Button(role: .destructive) {
                        showLogoutConfirmation = true
                    } label: {
                        Label("Log Out", systemImage: "rectangle.portrait.and.arrow.right")
                    }
                }

                // App info
                Section {
                    LabeledContent("Version", value: "1.0.0")
                    LabeledContent("Build", value: "1")
                } footer: {
                    Text("Hearth for iOS")
                        .frame(maxWidth: .infinity)
                        .multilineTextAlignment(.center)
                        .padding(.top, 8)
                }
            }
            .navigationTitle("Profile")
            .confirmationDialog(
                "Log Out",
                isPresented: $showLogoutConfirmation,
                titleVisibility: .visible
            ) {
                Button("Log Out", role: .destructive) {
                    authManager.logout()
                }
                Button("Cancel", role: .cancel) {}
            } message: {
                Text("Are you sure you want to log out?")
            }
        }
    }

    private func statusColor(_ status: User.UserStatus) -> Color {
        switch status {
        case .online: return .green
        case .idle: return .yellow
        case .dnd: return .red
        case .offline, .invisible: return .gray
        }
    }

    private func statusLabel(_ status: User.UserStatus) -> String {
        switch status {
        case .online: return "Online"
        case .idle: return "Idle"
        case .dnd: return "Do Not Disturb"
        case .offline: return "Offline"
        case .invisible: return "Invisible"
        }
    }
}

#Preview {
    ProfileView()
        .environment(AuthManager())
}
