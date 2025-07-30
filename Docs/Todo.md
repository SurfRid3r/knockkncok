# knockknock-NG Todo List

---

## Milestones

- [x] **M0: Protocol & Client CLI**
  - [x] Implement SPA v2 protocol encoding and decoding (`proto.go`).
  - [x] Create the `kk` client command.
  - [x] Implement `kk init` for key generation.
  - [x] Implement `kk send` to send the knock packet.

- [x] **M1: Generic Sniffer**
  - [x] Implement the packet sniffer using `gopacket` (`sniffer.go`).
  - [ ] Ensure it works on all supported platforms (macOS, Linux, Windows).

- [ ] **M2: Linux Firewall & TTL Engine**
  - [ ] Implement the `Firewall` interface for Linux using `nftables` or `iptables` (`firewall.go`).
  - [ ] Implement the dynamic TTL engine (`ttl.go`).
  - [ ] Integrate the TTL engine with the firewall.
  - [ ] Implement `bbolt` persistence for the whitelist (`db.go`).

- [ ] **M3: Windows Firewall**
  - [ ] Implement the `Firewall` interface for Windows using `netsh` (`firewall_windows.go`).
  - [ ] Test Windows-specific sniffing requirements (Npcap).

- [ ] **M4: Android Client**
  - [ ] Create an Android `.aar` library for the knock functionality.
  - [ ] Develop a simple demo APK to test the `.aar`.

- [x] **General Tasks**
  - [x] Write `knockd` server main loop (`main.go`).
  - [ ] Implement logging for server events.
  - [x] Create documentation (`README.md`).