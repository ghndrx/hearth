import SwiftUI

struct MessageListView: View {
    let channel: Channel

    @State private var messages: [Message] = []
    @State private var isLoading = false
    @State private var isLoadingMore = false
    @State private var hasMoreMessages = true
    @State private var errorMessage: String?
    @State private var scrollToBottom = false

    private let apiClient: APIClient

    init(
        channel: Channel,
        apiClient: APIClient = APIClient(baseURL: URL(string: "https://api.hearth.hndrx.co")!)
    ) {
        self.channel = channel
        self.apiClient = apiClient
    }

    var body: some View {
        VStack(spacing: 0) {
            // Messages
            ScrollViewReader { proxy in
                ScrollView {
                    LazyVStack(spacing: 0) {
                        // Load more indicator
                        if hasMoreMessages && !messages.isEmpty {
                            Button {
                                Task { await loadOlderMessages() }
                            } label: {
                                if isLoadingMore {
                                    ProgressView()
                                        .padding()
                                } else {
                                    Text("Load older messages")
                                        .font(.caption)
                                        .foregroundStyle(.secondary)
                                        .padding()
                                }
                            }
                            .disabled(isLoadingMore)
                        }

                        // Message list
                        ForEach(Array(messages.enumerated()), id: \.element.id) { index, message in
                            let showHeader = shouldShowHeader(for: index)
                            MessageBubbleView(message: message, showHeader: showHeader)
                                .id(message.id)
                        }
                    }
                    .padding(.vertical, 8)
                }
                .onChange(of: messages.count) {
                    if scrollToBottom, let lastId = messages.last?.id {
                        withAnimation(.easeOut(duration: 0.2)) {
                            proxy.scrollTo(lastId, anchor: .bottom)
                        }
                        scrollToBottom = false
                    }
                }
            }

            Divider()

            // Composer
            MessageComposerView(channelId: channel.id) { newMessage in
                messages.append(newMessage)
                scrollToBottom = true
            }
        }
        .navigationTitle("#\(channel.name)")
        .navigationBarTitleDisplayMode(.inline)
        .overlay {
            if isLoading && messages.isEmpty {
                ProgressView("Loading messages...")
            }
        }
        .task {
            await loadMessages()
        }
    }

    /// Determine whether to show author header (suppress for consecutive messages by same user within 5 min).
    private func shouldShowHeader(for index: Int) -> Bool {
        guard index > 0 else { return true }
        let prev = messages[index - 1]
        let curr = messages[index]

        if prev.authorId != curr.authorId { return true }

        if let prevDate = prev.createdAt, let currDate = curr.createdAt {
            return currDate.timeIntervalSince(prevDate) > 300
        }

        return true
    }

    private func loadMessages() async {
        isLoading = true
        defer { isLoading = false }

        do {
            let fetched = try await apiClient.fetchMessages(channelId: channel.id)
            messages = fetched.reversed() // API returns newest first; display oldest first
            scrollToBottom = true
            hasMoreMessages = fetched.count >= 50
            errorMessage = nil
        } catch {
            errorMessage = error.localizedDescription
        }
    }

    private func loadOlderMessages() async {
        guard !isLoadingMore, let oldestId = messages.first?.id else { return }
        isLoadingMore = true
        defer { isLoadingMore = false }

        do {
            let older = try await apiClient.fetchMessages(
                channelId: channel.id, before: oldestId, limit: 50
            )
            messages.insert(contentsOf: older.reversed(), at: 0)
            hasMoreMessages = older.count >= 50
        } catch {
            errorMessage = error.localizedDescription
        }
    }
}

// MARK: - Message Bubble

struct MessageBubbleView: View {
    let message: Message
    let showHeader: Bool

    var body: some View {
        HStack(alignment: .top, spacing: 10) {
            if showHeader {
                // Avatar
                AsyncImage(url: message.author?.avatarUrl.flatMap { URL(string: $0) }) { image in
                    image.resizable().scaledToFill()
                } placeholder: {
                    Circle()
                        .fill(Color.orange.opacity(0.3))
                        .overlay {
                            Text(String(message.author?.username.prefix(1) ?? "?").uppercased())
                                .font(.caption.bold())
                                .foregroundStyle(.orange)
                        }
                }
                .frame(width: 36, height: 36)
                .clipShape(Circle())
            } else {
                Spacer()
                    .frame(width: 36)
            }

            VStack(alignment: .leading, spacing: 2) {
                if showHeader {
                    HStack(spacing: 6) {
                        Text(message.author?.displayName ?? message.author?.username ?? "Unknown")
                            .font(.subheadline.weight(.semibold))

                        if let createdAt = message.createdAt {
                            Text(createdAt, style: .relative)
                                .font(.caption2)
                                .foregroundStyle(.secondary)
                        }
                    }
                }

                Text(message.content)
                    .font(.body)
                    .textSelection(.enabled)

                // Attachments
                if let attachments = message.attachments, !attachments.isEmpty {
                    ForEach(attachments) { attachment in
                        AttachmentView(attachment: attachment)
                    }
                }

                // Reactions
                if let reactions = message.reactions, !reactions.isEmpty {
                    HStack(spacing: 4) {
                        ForEach(reactions, id: \.emoji) { reaction in
                            HStack(spacing: 2) {
                                Text(reaction.emoji)
                                Text("\(reaction.count)")
                                    .font(.caption2)
                            }
                            .padding(.horizontal, 6)
                            .padding(.vertical, 2)
                            .background(
                                reaction.me ? Color.orange.opacity(0.2) : Color(.tertiarySystemFill),
                                in: Capsule()
                            )
                        }
                    }
                    .padding(.top, 2)
                }
            }

            Spacer(minLength: 0)
        }
        .padding(.horizontal, 12)
        .padding(.vertical, showHeader ? 6 : 1)
    }
}

private struct AttachmentView: View {
    let attachment: Attachment

    var body: some View {
        if let contentType = attachment.contentType, contentType.hasPrefix("image/") {
            AsyncImage(url: URL(string: attachment.url)) { image in
                image
                    .resizable()
                    .scaledToFit()
                    .frame(maxWidth: 300, maxHeight: 300)
                    .clipShape(RoundedRectangle(cornerRadius: 8))
            } placeholder: {
                RoundedRectangle(cornerRadius: 8)
                    .fill(Color(.tertiarySystemFill))
                    .frame(width: 200, height: 150)
                    .overlay { ProgressView() }
            }
        } else {
            HStack {
                Image(systemName: "doc.fill")
                VStack(alignment: .leading) {
                    Text(attachment.filename)
                        .font(.caption.weight(.medium))
                    Text(ByteCountFormatter.string(fromByteCount: Int64(attachment.size), countStyle: .file))
                        .font(.caption2)
                        .foregroundStyle(.secondary)
                }
            }
            .padding(8)
            .background(Color(.tertiarySystemFill), in: RoundedRectangle(cornerRadius: 8))
        }
    }
}
