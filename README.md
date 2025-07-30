# knockknock-NG

`knockknock-NG` is a single-packet authorization (SPA) framework designed for simplicity and security.

## Overview

- **Client (`kk`)**: A simple command-line tool for macOS, Linux, Windows, and Android.
- **Server (`knockd`)**: A daemon for Linux and Windows.

The primary use case is to run services like SOCKS5, RDP, or SSH on a server without a password. To gain access, a client must first send a specially crafted, encrypted TCP SYN packet (the "knock"). If the server validates the packet, it adds the client's IP address to a temporary whitelist, granting access to a predefined set of ports.

## Design Principles

- **Minimalist Client**: The client's only job is to send the knock. It doesn't need to know the ports or the TTL (time-to-live) for the firewall rule.
- **Centralized Server Control**: The server configuration defines the allowed ports and the TTL for the firewall rule.
- **Minimal Platform Differences**: The code is written in pure Go, with platform-specific implementations for the `Sniffer` and `Firewall` interfaces.
- **No Listening Ports**: The server uses a raw socket to sniff for SPA packets, which means it doesn't have any open ports to attack.
- **Lean Codebase**: The core daemon is around 350 lines of Go code.

## How It Works: The Invisible Doorman

`knockd` does **not** open any ports to listen for connections. This is the key to its security. Instead, it works like an invisible doorman watching traffic from a hidden camera.

1.  **Before the Knock**: Your server's firewall blocks all access to sensitive ports (like SSH on port 22). The SSH service is running, but is unreachable from the outside world.

2.  **The Knock**: You run `kk send`. This sends a single, encrypted packet to your server. This packet is like a secret passphrase.

3.  **The Doorman Sees**: `knockd`, which is passively sniffing all traffic on the network interface, sees this special packet. It decrypts and validates it using your shared secret key.

4.  **Opening the Door**: If the packet is valid, `knockd`'s only job is to tell the server's firewall: "Please open port 22, but *only* for the IP address that just sent me that secret knock."

5.  **Connecting**: The firewall now has a temporary rule. You can immediately connect using your normal SSH client (`ssh user@your_server`). Anyone else trying to connect is still blocked.

6.  **Closing the Door**: After a configured time (TTL), `knockd` tells the firewall to remove the temporary rule, closing the door again until the next successful knock.

This process means your server has virtually no attack surface, as the protected services are invisible until you authenticate.

## Getting Started

### Server (`knockd`)

1.  **Create a configuration file (`knockd.toml`)**:

    ```toml
    iface        = "eth0"            # (Optional) The interface to listen on. Auto-detected if not specified.
    allow_ports  = [1080, 22, 443]   # Ports to open upon successful knock
    base_ttl_min = 10                # Base TTL in minutes
    max_ttl_min  = 1440              # Maximum TTL in minutes
    db_file      = "whitelist.db"

    key = "..."                  # 256-bit master key (base64)
    ```

    If `key` is not specified, the server will generate a new one and print it to the console.

2.  **Run the server**:

    ```bash
    ./knockd
    ```

### Client (`kk`)

1.  **Initialize the client (one-time setup)**:

    ```bash
    ./kk init
    ```

    This will generate the master key for you to copy to your `knockd.toml` file.

2.  **Send a knock**:

    ```bash
    ./kk send -s <server_ip> -k <master_key>
    ```

## Compiling from Source

To compile `knockd` and `kk`, you need to have Go installed. You can cross-compile for different operating systems.

All binaries will be placed in the `output` directory.

### For Linux (amd64)

```bash
# Server
GOOS=linux GOARCH=amd64 go build -o output/knockd ./knockd

# Client
GOOS=linux GOARCH=amd64 go build -o output/kk ./kk
```

### For Windows (amd64)

```bash
# Server
GOOS=windows GOARCH=amd64 go build -o output/knockd.exe ./knockd

# Client
GOOS=windows GOARCH=amd64 go build -o output/kk.exe ./kk
```

### For macOS (amd64)

```bash
# Server
GOOS=darwin GOARCH=amd64 go build -o output/knockd ./knockd

# Client
GOOS=darwin GOARCH=amd64 go build -o output/kk ./kk
```

## Security

### Permissions

`knockd` needs sufficient permissions to modify firewall rules.

-   **On Linux**: You must run `knockd` as `root`, or grant it the necessary capabilities:
    ```bash
    sudo setcap cap_net_raw,cap_net_admin+eip ./output/knockd
    ```
-   **On Windows**: You must run `knockd.exe` with Administrator privileges.


- **Replay Attack Prevention**: The SPA packet includes a timestamp and a nonce to prevent replay attacks.
- **No Listening Ports**: The server does not listen on any ports, reducing the attack surface.
- **Least Privilege**: The server runs with the minimum required privileges.
- **Persistent Whitelist**: The server uses a `bbolt` database to store the whitelist, so it can be restored after a restart.
