# http-tunnel-go
A Cross-Platform HTTP Tunnel written using Go

# Usage
```
  -auth string
    	Proxy Authentication [username:password] (optional)
  -payload string
    	Payload (required)
  -port int
    	Server Port (default 8888)
  -proxy string
    	Proxy Server [host:port] (required)
```

## Example:
Without proxy authentication:

```./http-tunnel-go -proxy 52.77.245.227:8080 -port 8888 -payload "CONNECT [host_port] HTTP/1.0[crlf][crlf]"```

With proxy authentication:

```./http-tunnel-go -proxy 52.77.245.227:8080 -port 8888 -auth leo:leo -payload "CONNECT [host_port] HTTP/1.0[crlf][crlf]"```

## License
Non-Profit Open Software License 3.0 (NPOSL-3.0)
