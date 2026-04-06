import SwiftUI

struct MessageComposerView: View {
    let channelId: String
    let onMessageSent: (Message) -> Void

    @State private var text = ""
    @State private var isSending = false

    private let apiClient: APIClient

    init(
        channelId: String,
        apiClient: APIClient = APIClient(baseURL: URL(string: "https://api.hearth.hndrx.co")!),
        onMessageSent: @escaping (Message) -> Void
    ) {
        self.channelId = channelId
        self.apiClient = apiClient
        self.onMessageSent = onMessageSent
    }

    private var canSend: Bool {
        !text.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty && !isSending
    }

    var body: some View {
        HStack(alignment: .bottom, spacing: 8) {
            // Attachment button placeholder
            Button {
                // TODO: Attachment picker
            } label: {
                Image(systemName: "plus.circle.fill")
                    .font(.title2)
                    .foregroundStyle(.secondary)
            }

            // Text input
            TextField("Message #...", text: $text, axis: .vertical)
                .textFieldStyle(.plain)
                .lineLimit(1...6)
                .padding(.horizontal, 12)
                .padding(.vertical, 8)
                .background(Color(.tertiarySystemFill), in: RoundedRectangle(cornerRadius: 20))

            // Send button
            Button {
                Task { await sendMessage() }
            } label: {
                Image(systemName: "arrow.up.circle.fill")
                    .font(.title2)
                    .foregroundStyle(canSend ? .orange : .secondary)
            }
            .disabled(!canSend)
        }
        .padding(.horizontal, 12)
        .padding(.vertical, 8)
        .background(Color(.systemBackground))
    }

    private func sendMessage() async {
        let content = text.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !content.isEmpty else { return }

        isSending = true
        let sentText = content
        text = ""

        do {
            let message = try await apiClient.sendMessage(channelId: channelId, content: sentText)
            onMessageSent(message)
        } catch {
            // Restore text on failure so user doesn't lose their message
            text = sentText
        }

        isSending = false
    }
}

#Preview {
    MessageComposerView(channelId: "preview") { _ in }
        .border(Color.gray)
}
