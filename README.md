# pluggo

pluggo is a flexible TCP proxy and relay server written in Go. It listens for incoming TCP connections on configurable frontends and transparently forwards them to different types of backends. It is especially useful for waking up sleeping devices on demand and connecting to them automatically.

## What does it do?

- Listens for incoming TCP connections on specified addresses (frontends)
- Forwards each connection to a configured backend
- Supports three backend types:
  - **Echo backend:** Echoes all received data back to the client
  - **TCP forwarder backend:** Forwards the connection to another TCP server
  - **Wake-on-LAN (WOL) forwarder backend:** Sends a WOL magic packet to wake up a target machine, waits for it to become available, then forwards the connection

## Example Use Case

You want to connect to a device in your network that is usually suspended. pluggo can listen for connections, send a WOL packet to wake the target device, wait for it to boot, and then forward your connection to it automatically.

## Build it

Simply use Go:

```sh
go build -o pluggo *.go
```

or if you want to have a small, stripped binary without any debug information:

```sh
go build -ldflags '-s' -o pluggo *.go
```

## Configuration

pluggo is configured using a `config.toml` file. You define one or more TCP frontends and backends. Each frontend listens for incoming connections and forwards them to a specified backend. Below is an example configuration, followed by an explanation of each field:

```toml
[[frontends.tcp]]
name        = "Test Frontend"          # Unique name for this frontend
listenAddr  = "127.0.0.1:8080"         # Address and port to listen on (e.g., 0.0.0.0:1234)
target      = "WoL Forwarder"          # Name of the backend to forward connections to

[[backends.echo]]
name = "Echo Backend"                  # Unique name for this echo backend

[[backends.wolForwarder]]
name             = "WoL Forwarder"     # Unique name for this WOL forwarder backend
targetAddr       = "192.168.0.2:22"    # Address to forward the connection to after waking the device
wolMACAddr       = "12:34:56:ab:cd:ef" # MAC address of the device to wake up
wolBroadcastAddr = "192.168.0.255:9"   # Broadcast address and port for the WOL magic packet
```

You can define multiple frontends and backends as needed. Each frontend can point to any backend by name.

## Disclaimer

This project is just something I made for my own homeserver. You can use or fork it if you want, but don't expect me to add features for you. Use it at your own risk.
