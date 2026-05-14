# Research: HTTP Efficiency in Go

## 1. Connection Pooling (Keep-Alive)
- **Problem:** Creating a new `http.Client` for every request forces a new TCP connection and TLS handshake.
- **Solution:** Reuse a single `*http.Client` or `*http.Transport`.
- **Details:** `http.Client` is thread-safe. Reusing it allows the underlying `Transport` to maintain a pool of idle connections.
- **Optimization:** Tune `MaxIdleConns` and `MaxIdleConnsPerHost` in `http.Transport`. For high-concurrency scanning, `MaxIdleConnsPerHost` should be increased (default is only 2).

## 2. Timeouts
- **Best Practice:** Always set `Timeout` on `http.Client`.
- **Granularity:** Use `context.WithTimeout` for per-request control if needed, but the client-level timeout is a good safety net.

## 3. Response Body Management
- **Rule:** Always close `resp.Body` to release the connection back to the pool.
- **Efficiency:** If the body isn't needed, use a `HEAD` request or discard the body with `io.Copy(io.Discard, resp.Body)` before closing. However, HyperWapp *needs* the body for technology detection.

## 4. Custom Transport for Mass Scanning
- **Settings:**
    - `DialContext`: Set aggressive dial timeouts.
    - `TLSHandshakeTimeout`: Limit time spent on handshakes.
    - `ExpectContinueTimeout`: Usually small or default.
    - `DisableKeepAlives`: Set to `false` (default) to enable reuse.
