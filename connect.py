#!/usr/bin/env python3
"""
Simple telnet client that strips IAC codes and connects to Dark Pawns.
"""
import socket
import sys
import time
import select

def strip_iac(data: bytes) -> bytes:
    """Remove telnet IAC negotiation bytes."""
    i = 0
    out = bytearray()
    while i < len(data):
        if data[i] == 255:  # IAC
            i += 1
            if i < len(data):
                cmd = data[i]
                i += 1
                if cmd == 251:  # WILL
                    i += 1
                elif cmd == 252:  # WONT
                    i += 1
                elif cmd == 253:  # DO
                    i += 1
                elif cmd == 254:  # DONT
                    i += 1
                elif cmd == 250:  # SB
                    # subnegotiation — skip until SE
                    while i < len(data) and not (data[i] == 255 and i+1 < len(data) and data[i+1] == 240):
                        i += 1
                    i += 2
                else:
                    # other IAC command, skip
                    pass
        else:
            out.append(data[i])
            i += 1
    return bytes(out)

def main():
    host = "192.168.1.106"
    port = 4350

    sock = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
    sock.settimeout(5)
    try:
        sock.connect((host, port))
    except Exception as e:
        print(f"Connection failed: {e}")
        sys.exit(1)

    sock.setblocking(False)
    print(f"Connected to {host}:{port}")
    print("---")

    # initial read
    time.sleep(0.5)
    buf = b""
    while True:
        r, _, _ = select.select([sock], [], [], 0.1)
        if r:
            chunk = sock.recv(4096)
            if not chunk:
                break
            buf += chunk
        else:
            break

    if buf:
        cleaned = strip_iac(buf).decode('latin-1', errors='replace')
        sys.stdout.write(cleaned)
        sys.stdout.flush()

    # interactive loop
    sock.setblocking(True)
    while True:
        r, _, _ = select.select([sock, sys.stdin], [], [])
        if sock in r:
            data = sock.recv(4096)
            if not data:
                print("\nConnection closed.")
                break
            cleaned = strip_iac(data).decode('latin-1', errors='replace')
            sys.stdout.write(cleaned)
            sys.stdout.flush()
        if sys.stdin in r:
            line = sys.stdin.readline()
            if not line:
                break
            sock.send(line.encode())

if __name__ == "__main__":
    main()