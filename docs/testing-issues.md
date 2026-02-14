# Testing Issues Log

Log of problems, observations, and discussions from testing sessions.

---

## 2026-02-14: Cross-Platform Testing (Linux Host ↔ Windows Joiner)

**Environment**: Omarchy (Arch Linux) hosting, Windows client joining

### Observations

- Handshake completed successfully (SDP offer/answer exchanged)
- Linux side reported "connected"
- Windows side did not report "connected"

### Requirements Clarification

- **Host side**: Must have Minecraft server running on `localhost:42517` *before* WebRTC connection completes. The tunnel immediately attempts to connect to the Minecraft server when the DataChannel opens.
  - For single-player worlds: Use "Open to LAN" button in Minecraft pause menu (defaults to port 42517)
- **Joiner side**: Opens a listener on `localhost:42517`. User connects their Minecraft client to this address after the tunnel shows "connected".

### Changes Made

- Updated default port from `25565` (dedicated server default) to `42517` (single-player "Open to LAN" default)
- This aligns with the primary use case: two gamers playing together via single-player LAN

### Bugs Found

1. **Join side UI freezes on error** - If there's a connection error on the join side, the UI becomes unresponsive. User cannot retry or navigate back to try again. The app appears stuck with no way to recover without restarting.
   - **Fix applied**: Removed `set({ status: "error" })` from acceptOffer, show offer input on "error" status, added "Retry" button, enabled Back button on error state

2. **Join side is broken** - The join functionality is not working correctly. Further investigation needed to determine root cause (WebRTC connection state, DataChannel handling, or proxy logic).

3. **Answer token base64 decode error** - `Error: invalid answer token format: illegal base64 data at input byte 2000`
   - Occurred when host pasted answer token from joiner
   - Token was transferred via Discord: Joiner exported file → uploaded to Discord → Host downloaded → copy/paste
   - Likely cause: Discord may have modified the file, or copy/paste introduced whitespace/newlines
   - WebRTC SDP tokens are large (~4KB+), sensitive to any modifications
   - **Fix applied**: Host UI now stays on "waiting-for-answer" state when answer validation fails, allowing retry

4. **WebRTC connection fails after handshake** - Both sides show "Connection failed" after what appears to be successful token exchange:
   - **Host logs**: `Error: invalid answer token format...` → `Tunnel established!` → `Connection failed`
   - **Join logs**: `Answer generated - share this with host` → `Token exported to file` → `Connection failed`
   - This indicates: Token exchange completed (at least once), but WebRTC PeerConnectionState went to `Failed`
   - Possible causes: ICE negotiation failure (NAT traversal), corrupted tokens causing mismatched connections, or firewall/network issues
   - **Investigation needed**: Check ICE candidate gathering, STUN server reachability, and whether both sides are actually connecting to each other
