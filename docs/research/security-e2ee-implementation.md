# E2EE Implementation Research

**Topic:** End-to-End Encryption Implementation Patterns  
**Date:** 2026-02-12  
**Status:** Research Notes

---

## Executive Summary

Hearth's E2EE.md already covers the core architecture well (Signal Protocol, X3DH, Double Ratchet, MLS for groups). This document supplements with implementation pitfalls, recent developments, and lessons from production systems.

---

## 1. Protocol Comparison

### Signal Protocol (Recommended for DMs)

Used by: Signal, WhatsApp, Google Messages (RCS), Facebook Messenger (Secret Conversations)

**Components:**
- **X3DH:** Extended Triple Diffie-Hellman for initial key agreement
- **Double Ratchet:** Per-message key derivation with forward secrecy
- **Prekeys:** One-time ephemeral keys for async session establishment

**Cryptographic Primitives:**
- Curve25519 (key agreement)
- AES-256 (symmetric encryption)
- HMAC-SHA256 (authentication)
- Ed25519 (signatures)

**Key Properties:**
| Property | Signal Protocol |
|----------|-----------------|
| Forward Secrecy | ✅ Per-message |
| Post-Compromise Security | ✅ Self-healing |
| Async Support | ✅ Via prekeys |
| Deniability | ✅ No proof of authorship |
| Audit Status | ✅ Formally verified (2016) |

### Matrix Olm/Megolm (Alternative for Groups)

Used by: Element, Matrix ecosystem

**Key Insight:** Matrix separates 1:1 (Olm) from group (Megolm):

- **Olm:** Double Ratchet variant for pairwise sessions
- **Megolm:** Efficient group encryption with different trade-offs

**Megolm Design:**
```
Sender creates Megolm session:
├── 32-bit counter (i)
├── Ed25519 signing keypair (K)
└── Ratchet state: R[i,0], R[i,1], R[i,2], R[i,3]

Message encryption:
1. Derive AES-256 key + HMAC key + IV from ratchet state
2. Encrypt with AES-256-CBC + PKCS#7 padding
3. HMAC-SHA256 (truncated to 64 bits)
4. Sign with Ed25519
5. Advance ratchet
```

**Why Megolm over pairwise for groups:**
- O(1) encryption per message (vs O(n) for pairwise)
- Recipients can decrypt multiple times
- Ratchet can skip forward efficiently (max 1020 hashes)

**Trade-off:** Weaker forward secrecy — if session key leaks, all messages from that point forward can be decrypted until session rotation.

---

## 2. Signal Protocol Deep Dive

### X3DH Key Agreement

```
Alice (initiator)              Bob (responder)
─────────────────              ────────────────
Identity Key: IKa              Identity Key: IKb
Ephemeral Key: EKa             Signed Prekey: SPKb
                               One-Time Prekey: OPKb (optional)

Shared Secret = HKDF(
    DH(IKa, SPKb) ||           // Alice's identity, Bob's signed prekey
    DH(EKa, IKb) ||            // Alice's ephemeral, Bob's identity
    DH(EKa, SPKb) ||           // Alice's ephemeral, Bob's signed prekey
    DH(EKa, OPKb)              // Alice's ephemeral, Bob's one-time prekey
)
```

**Why 4 DH operations?**
- Mutual authentication (both identities involved)
- Forward secrecy (ephemeral keys)
- One-time prekey prevents replay attacks

### Double Ratchet Mechanics

Each party maintains:
- Root key (RK)
- Sending chain key (CKs)
- Receiving chain key (CKr)
- Sending ratchet key pair
- Receiving ratchet public key

**Symmetric Ratchet:** For each message:
```
CK(new), MK = KDF(CK(old), constant)
```

**DH Ratchet:** When receiving a new public key:
```
RK(new), CK(receiving) = KDF(RK, DH(my_private, their_public))
Generate new ratchet keypair
RK(new), CK(sending) = KDF(RK, DH(new_private, their_public))
```

### Out-of-Order Message Handling

Signal caches skipped message keys to handle reordering:
```
MAX_SKIP = 2000 messages per chain
MAX_CACHED_KEYS = 5000 total

On receive(message_number):
    if message_number > current + MAX_SKIP:
        reject (too far ahead)
    if message_number < current:
        lookup cached key
    else:
        advance chain, cache skipped keys
```

**Security Note:** Cached keys weaken forward secrecy. Implementations should:
- Limit cache size and lifetime
- Clear keys after successful decryption
- Allow users to reduce MAX_SKIP for high-security use

---

## 3. Post-Quantum Considerations

### Harvest Now, Decrypt Later (HNDL)

Threat: Adversary records encrypted traffic today, decrypts with future quantum computer.

**Signal's Response (2024):** PQXDH + Triple Ratchet
- Adds Kyber-1024 (ML-KEM) alongside X25519
- Hybrid approach: classical + post-quantum
- "Triple Ratchet" = Double Ratchet + Sparse Post-Quantum Ratchet (SPQR)

**SimpleX (2024):** Added quantum resistance to their Double Ratchet implementation.

### Recommendation for Hearth

**Phase 1 (v1.0):** Implement classical Signal Protocol
- Well-understood, audited, battle-tested
- Adequate for most threat models

**Phase 2 (v1.x):** Add post-quantum hybrid
- Watch Signal's PQXDH standardization
- Consider NIST-approved ML-KEM (Kyber successor)
- Hybrid = defense in depth (if PQ algorithm breaks, classical still protects)

---

## 4. Implementation Pitfalls

### Pitfall 1: Prekey Exhaustion

**Problem:** One-time prekeys run out; sessions established with only signed prekey lose replay protection.

**Solution:**
```
// Client-side monitoring
async function monitorPrekeys() {
    const count = await getServerPrekeyCount();
    if (count < 20) {
        await uploadNewPrekeys(100);
    }
}

// Server-side: return prekey count in /sync response
// Client: replenish proactively, not reactively
```

### Pitfall 2: Key Rotation Timing

**Problem:** Signed prekeys not rotated → long-term key compromise exposes more sessions.

**Solution:**
- Rotate signed prekey every 7 days (Signal default)
- Keep old signed prekey for ~30 days (allow in-flight sessions)
- Never delete identity key (it IS the device identity)

### Pitfall 3: Session State Loss

**Problem:** Database corruption or app reinstall loses ratchet state → can't decrypt pending messages.

**Solution:**
```
// Atomic writes for session state
await db.transaction(async (tx) => {
    await tx.update('sessions', session);
    await tx.update('message_keys', messageKeys);
    // Both succeed or both fail
});

// Optional: Encrypted cloud backup of session keys
// (requires careful passphrase-derived key handling)
```

### Pitfall 4: Timing Side Channels

**Problem:** Decryption time varies based on success/failure → timing oracle.

**Solution:**
- Constant-time comparison for MACs
- Always complete full decryption path
- Use audited crypto libraries (libsignal, vodozemac)

### Pitfall 5: Device Verification UX

**Problem:** Users skip verification → MITM possible via malicious server.

**Solutions from Signal/WhatsApp:**
- Safety numbers shown prominently
- Alert on identity key change
- QR code scanning for in-person verification
- Optional: Require verification for sensitive channels

---

## 5. Platform Comparison

### What Discord Does (No E2EE)

Discord provides:
- TLS for transit encryption
- Server-side storage (plaintext to Discord)
- No E2EE for DMs or channels

**Implications:**
- Discord can read all messages
- Subpoenas can produce message content
- Server compromise exposes everything

### What Slack Does (No E2EE)

Similar to Discord:
- Enterprise Key Management (EKM) for enterprise tier
- Customer-controlled encryption keys at rest
- Still not E2EE — Slack infrastructure can decrypt

### What Signal Does (Gold Standard)

- E2EE for everything by default
- Sealed sender (hide sender from server)
- No metadata stored (minimal logs)
- Open source, audited

### What WhatsApp Does (Signal Protocol + Metadata)

- E2EE using Signal Protocol
- BUT: Collects extensive metadata
- Backups can be unencrypted (user choice)
- Closed source client

### Hearth's Position

Based on E2EE.md, Hearth aims for:
- **DMs:** Always E2EE (like Signal)
- **Channels:** Admin choice (pragmatic)
- **Voice/Video:** E2EE via SRTP/DTLS

**Recommendation:** This is a good middle ground. Consider adding:
- Clear UX indicator for E2EE status
- Warning when disabling channel E2EE
- Audit log for E2EE setting changes

---

## 6. MLS for Group Channels

Messaging Layer Security (RFC 9420, finalized 2023) is the IETF standard for group E2EE.

### Why MLS over Sender Keys

| Aspect | Sender Keys | MLS |
|--------|-------------|-----|
| Forward Secrecy | Per-session | Per-epoch |
| Key Rotation | Manual | On membership change |
| Scalability | O(n) key distribution | O(log n) tree structure |
| Standardization | Proprietary | IETF RFC 9420 |

### MLS Key Concepts

- **Group:** Set of members with shared state
- **Epoch:** Version of group state (incremented on changes)
- **Ratchet Tree:** Binary tree for efficient key distribution
- **Commit:** Message that advances epoch (add/remove member, update keys)

### Implementation Libraries

- **OpenMLS (Rust):** Most mature, used in production
- **mls-rs (Rust):** Newer, lighter weight
- **mlspp (C++):** Cisco's implementation

### Recommendation for Hearth

**v1.0:** Start with Sender Keys for group DMs (simpler, Signal does this)  
**v1.x:** Migrate to MLS for server channels (better scaling, standard)

---

## 7. Gaps Identified in E2EE.md

### TODO 1: Prekey Monitoring
Add client-side logic to monitor and replenish one-time prekeys.

### TODO 2: Session Recovery
Document behavior when session state is lost. Options:
- Request session reset from peer
- Lose message history (acceptable for DMs)

### TODO 3: Identity Key Change Handling
Specify what happens when a contact's identity key changes:
- Block messages until verified?
- Show warning but allow (Signal default)?
- Admin-configurable per server?

### TODO 4: Sealed Sender Priority
Currently P2 — consider elevating. Metadata protection is valuable.

### TODO 5: Post-Quantum Roadmap
Add explicit section on PQ migration path.

### TODO 6: Library Selection
E2EE.md lists libraries but doesn't recommend. For Go backend + TypeScript client:
- **Server:** Key storage only, no crypto
- **Client (TypeScript):** `@aspect-build/libsignal-client` or `olm` via WebAssembly

---

## 8. Security Audit Checklist

Before shipping E2EE:

- [ ] Use audited crypto library (never roll your own)
- [ ] Constant-time operations for sensitive comparisons
- [ ] Secure key storage (keychain/keyring, not localStorage)
- [ ] Prekey replenishment monitoring
- [ ] Identity key change alerts
- [ ] Session state backup/recovery documented
- [ ] Rate limiting on key fetch APIs
- [ ] Audit logging for key operations
- [ ] Third-party security audit (before v1.0 GA)

---

## References

1. Signal Protocol Specification: https://signal.org/docs/
2. Double Ratchet Algorithm: https://signal.org/docs/specifications/doubleratchet/
3. Matrix E2EE Guide: https://matrix.org/docs/matrix-concepts/end-to-end-encryption/
4. MLS RFC 9420: https://www.rfc-editor.org/rfc/rfc9420.html
5. Signal Post-Quantum: https://signal.org/blog/pqxdh/
6. Formal Analysis of Signal Protocol (2016): https://eprint.iacr.org/2016/1013.pdf

---

*Next Research Topic: JWT/Auth Security (refresh tokens, session management)*
