# Hearth — End-to-End Encryption (E2EE)

**Version:** 2.0  
**Last Updated:** 2026-02-11  
**Status:** Core Feature (v1.0)

---

## Philosophy

**E2EE is not optional in Hearth. It's the default.**

Every private conversation is encrypted end-to-end. The server **never** sees plaintext content. This isn't a premium feature or an afterthought — it's the foundation of how Hearth works.

> *"If we can read your messages, so can hackers, governments, and bad actors. We chose not to have that capability."*

---

## Scope

| Channel Type | E2EE | Notes |
|--------------|------|-------|
| Direct Messages | ✅ **Always On** | Cannot be disabled |
| Group DMs | ✅ **Always On** | All members have keys |
| Server Channels | ✅ **Default On** | Can be disabled per-channel by admin |
| Voice/Video | ✅ **Always On** | SRTP with E2EE key exchange |
| Voice (future) | ⚠️ Planned | SRTP with E2EE key exchange |

---

## Why E2EE by Default?

### The Problem with "Optional"

When E2EE is optional:
- Most users don't enable it (friction)
- Metadata reveals who uses encryption (targeting)
- Server still stores plaintext for non-E2EE users
- "Nothing to hide" mentality prevails
- One compromised account exposes all non-E2EE history

### The Hearth Approach

When E2EE is default:
- **Zero plaintext on server** — Nothing to steal
- **Zero trust architecture** — Server is just a relay
- **Uniform metadata** — Everyone looks the same
- **No compliance headaches** — We can't produce what we don't have
- **User trust** — Privacy isn't a feature, it's the product

---

## Protocol

### Signal Protocol (MLS for Groups)

Hearth implements the Signal Protocol for 1:1 messaging and MLS (Messaging Layer Security) for groups. These are the same battle-tested protocols used by Signal, WhatsApp, and others. They provide:

- **Perfect Forward Secrecy (PFS):** Compromising a key doesn't expose past messages
- **Future Secrecy:** Recovering from compromise without manual intervention
- **Deniability:** No cryptographic proof of authorship
- **Asynchronous:** Works when recipients are offline

### Key Components

```
┌─────────────────────────────────────────────────────────────────┐
│                    Signal Protocol Stack                         │
├─────────────────────────────────────────────────────────────────┤
│  X3DH (Extended Triple Diffie-Hellman) - Key Agreement          │
│  ├── Identity Key (IK) - Long-term, per-device                  │
│  ├── Signed Pre-Key (SPK) - Medium-term, rotated                │
│  └── One-Time Pre-Keys (OPK) - Single-use, replenished          │
├─────────────────────────────────────────────────────────────────┤
│  Double Ratchet - Message Encryption                             │
│  ├── Diffie-Hellman Ratchet - New keys per message exchange     │
│  ├── Symmetric Ratchet - Chain keys for each direction          │
│  └── Message Keys - Derived per message, never reused           │
├─────────────────────────────────────────────────────────────────┤
│  AEAD (AES-256-GCM) - Symmetric Encryption                       │
│  └── Authenticated encryption with associated data               │
└─────────────────────────────────────────────────────────────────┘
```

---

## Key Management

### Device Keys

Each device generates its own key set:

```
Device A (Phone):
├── Identity Key Pair (Ed25519)
├── Signed Pre-Key Pair (X25519), rotated weekly
└── One-Time Pre-Keys (X25519), 100 keys, replenished

Device B (Desktop):
├── Identity Key Pair (Ed25519)
├── Signed Pre-Key Pair (X25519)
└── One-Time Pre-Keys (X25519)
```

### Key Registration

On device setup:
1. Generate Identity Key
2. Generate Signed Pre-Key (sign with Identity Key)
3. Generate 100 One-Time Pre-Keys
4. Upload public keys to server

```json
// POST /api/v1/keys/upload
{
  "identity_key": "base64...",
  "signed_pre_key": {
    "id": 1,
    "public_key": "base64...",
    "signature": "base64..."
  },
  "one_time_pre_keys": [
    {"id": 1, "public_key": "base64..."},
    {"id": 2, "public_key": "base64..."}
  ]
}
```

### Key Rotation

| Key Type | Rotation | Trigger |
|----------|----------|---------|
| Identity Key | Never | Permanent per device |
| Signed Pre-Key | Weekly | Time-based |
| One-Time Pre-Keys | On use | Replenish when <20 remain |

---

## Message Flow

### Initial Message (X3DH)

```
Alice                          Server                          Bob
  │                              │                              │
  │  1. Request Bob's keys       │                              │
  │─────────────────────────────>│                              │
  │                              │                              │
  │  2. Bob's key bundle         │                              │
  │<─────────────────────────────│                              │
  │  (IK, SPK, OPK)              │                              │
  │                              │                              │
  │  3. X3DH key agreement       │                              │
  │  (compute shared secret)     │                              │
  │                              │                              │
  │  4. Encrypt message          │                              │
  │  (Double Ratchet init)       │                              │
  │                              │                              │
  │  5. Send encrypted message   │                              │
  │─────────────────────────────>│─────────────────────────────>│
  │  (+ Alice's ephemeral key)   │                              │
  │                              │                              │
  │                              │  6. Bob decrypts             │
  │                              │  (X3DH + Double Ratchet)     │
```

### Subsequent Messages (Double Ratchet)

```
Alice                                                          Bob
  │                                                              │
  │  Message with new DH ratchet key                             │
  │─────────────────────────────────────────────────────────────>│
  │                                                              │
  │                             Reply with new DH ratchet key    │
  │<─────────────────────────────────────────────────────────────│
  │                                                              │
  │  (Each message advances the ratchet)                         │
```

---

## Message Format

### Encrypted Message

```json
{
  "type": "encrypted",
  "sender_device_id": "device_abc123",
  "recipient_device_id": "device_xyz789",
  "ciphertext": "base64...",
  "header": {
    "dh_key": "base64...",
    "previous_chain_length": 0,
    "message_number": 42
  },
  "timestamp": "2026-02-11T04:00:00Z"
}
```

### Decrypted Content

```json
{
  "content": "Hello, this is a secret message!",
  "attachments": [
    {
      "id": "attachment_123",
      "key": "base64...",
      "digest": "sha256:..."
    }
  ]
}
```

---

## Multi-Device Support

### Per-Device Encryption

Messages are encrypted separately for each of the recipient's devices:

```json
{
  "messages": [
    {
      "device_id": "phone_123",
      "ciphertext": "base64..."
    },
    {
      "device_id": "desktop_456",
      "ciphertext": "base64..."
    }
  ]
}
```

### Device Verification

Users can verify devices via:
1. **Safety Numbers:** Compare numeric codes in person
2. **QR Codes:** Scan to verify
3. **Emoji Grid:** Match emoji patterns

```
Safety Number:
12345 67890 12345 67890
12345 67890 12345 67890
12345 67890 12345 67890
```

---

## Group E2EE (Sender Keys)

For group DMs, Hearth uses Sender Keys (like Signal groups):

1. Each sender creates a Sender Key
2. Sender Key distributed to all group members via pairwise channels
3. Messages encrypted once with Sender Key
4. All recipients decrypt with same key

### Trade-offs

| Approach | Pros | Cons |
|----------|------|------|
| Pairwise | Forward secrecy per pair | O(n) encryptions per message |
| Sender Keys | O(1) encryption | Weaker forward secrecy |

---

## Encrypted Attachments

Files are encrypted client-side before upload:

```
1. Generate random AES-256 key
2. Encrypt file with AES-256-GCM
3. Upload encrypted blob to server
4. Include key in encrypted message body
```

```json
{
  "attachment": {
    "url": "https://cdn.hearth.example.com/encrypted/abc123",
    "key": "base64...",
    "digest": "sha256:...",
    "size": 1048576,
    "mime_type": "image/jpeg"
  }
}
```

---

## Key Backup & Recovery

### Encrypted Backup

Users can backup their keys with a passphrase:

1. Derive key from passphrase (Argon2id)
2. Encrypt key bundle with derived key
3. Store encrypted backup on server

### Recovery Options

| Method | Security | Convenience |
|--------|----------|-------------|
| Passphrase backup | Medium | Easy |
| Recovery key (44 chars) | High | Medium |
| Multi-device sync | High | Easy |
| No backup | Highest | Lose history on new device |

---

## Security Considerations

### What E2EE Protects

✅ Message content (text, files)  
✅ Who you're messaging (with sealed sender, future)  
✅ Past messages if current keys leak  

### What E2EE Doesn't Protect

❌ Metadata (who messaged whom, when)  
❌ User profiles (public info)  
❌ Server channel messages  
❌ Against compromised devices  

### Threat Model

| Threat | Mitigation |
|--------|------------|
| Server compromise | E2EE - server never sees plaintext |
| Network eavesdropping | TLS + E2EE |
| Key compromise | Forward secrecy (Double Ratchet) |
| Device theft | Local encryption + biometrics |
| Malicious client | Open source, reproducible builds |

---

## API Reference

### Upload Keys
```
POST /api/v1/keys/upload
Authorization: Bearer <token>
```

### Get User's Keys
```
GET /api/v1/keys/{user_id}/devices
```

### Get Device's Pre-Key Bundle
```
GET /api/v1/keys/{user_id}/devices/{device_id}/bundle
```

### Register Device
```
POST /api/v1/devices/register
```

### Remove Device
```
DELETE /api/v1/devices/{device_id}
```

---

## Server Channel E2EE

### How It Works

Server channels use MLS (Messaging Layer Security) for group encryption:

1. **Channel Key Group** — Each E2EE channel has an MLS group
2. **Member Join** — Adding a member adds them to the MLS group
3. **Key Rotation** — Keys rotate on member changes
4. **Forward Secrecy** — Past messages stay encrypted even if keys leak

### Admin Controls

Server owners can choose per-channel:

```
┌─────────────────────────────────────────────────────────────┐
│  Channel Settings: #general                                  │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  🔒 End-to-End Encryption                                   │
│                                                             │
│  ● Encrypted (recommended)                                  │
│    Messages are E2EE. Server cannot read content.          │
│    ⚠️ Search only works on your device.                    │
│    ⚠️ New members cannot see history before joining.       │
│                                                             │
│  ○ Unencrypted                                              │
│    Messages stored on server. Full search available.        │
│    Use for public announcements or searchable archives.     │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

### Trade-offs (Encrypted Channels)

| Feature | Encrypted | Unencrypted |
|---------|-----------|-------------|
| Server can read | ❌ No | ✅ Yes |
| Server-side search | ❌ No | ✅ Yes |
| History for new members | ❌ No | ✅ Yes |
| Link previews | ✅ Client-side | ✅ Server-side |
| File storage | ✅ Encrypted | ✅ Plaintext |
| Compliance export | ❌ No | ✅ Yes |

### Default Behavior

| Channel Type | Default | Changeable |
|--------------|---------|------------|
| DMs | Encrypted | ❌ No |
| Group DMs | Encrypted | ❌ No |
| Text Channels | Encrypted | ✅ Yes (by admin) |
| Voice Channels | Encrypted | ❌ No |
| Announcement | Unencrypted | ✅ Yes |
| Forum | Encrypted | ✅ Yes |

---

## Voice/Video E2EE

All voice and video calls are end-to-end encrypted using:

1. **SRTP** — Secure Real-time Transport Protocol
2. **DTLS** — Key exchange for SRTP
3. **Orotund frames** — Additional E2EE layer on top

The SFU (Selective Forwarding Unit) only sees encrypted packets — it cannot decode audio/video content.

```
┌─────────┐         ┌─────────┐         ┌─────────┐
│ Client A│◄───────►│   SFU   │◄───────►│Client B │
│(encrypt)│ cipher  │(relay)  │ cipher  │(decrypt)│
└─────────┘         └─────────┘         └─────────┘
          ▲                              ▲
          └──────── E2EE keys ───────────┘
            (exchanged peer-to-peer)
```

---

## Implementation Status

| Component | Status | Priority |
|-----------|--------|----------|
| X3DH key agreement | 🔨 In Progress | P0 |
| Double Ratchet | 🔨 In Progress | P0 |
| Multi-device | 🔨 In Progress | P0 |
| MLS (groups) | 🔨 In Progress | P0 |
| Key backup | 📋 Planned | P1 |
| Device verification | 📋 Planned | P1 |
| Voice E2EE (SRTP) | 📋 Planned | P0 |
| Sealed sender | 📋 Planned | P2 |

---

## Libraries

Recommended implementations:
- **libsignal-protocol:** Reference implementation
- **olm/megolm:** Matrix's implementation
- **Go:** `github.com/aspect-build/go-signal-protocol`

---

*End of E2EE Documentation*
