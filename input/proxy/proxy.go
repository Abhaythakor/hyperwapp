package proxy
import (
	"bytes"
	"encoding/pem"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/Abhaythakor/hyperwapp/model"
	"github.com/Abhaythakor/hyperwapp/util"
	"github.com/elazarl/goproxy"
)

// StartProxy starts a capture proxy server on the specified address.
// It intercepts HTTP responses and sends them to the output channel for analysis.
func StartProxy(addr string, outputCh chan<- model.OfflineInput) error {
	proxy := goproxy.NewProxyHttpServer()
	proxy.Verbose = false

	// Silence the internal goproxy logger to prevent terminal clutter
	proxy.Logger = log.New(io.Discard, "", 0)

	// Save the CA certificate (PEM format) to a file so the user can trust it in their browser
	caCertPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: goproxy.GoproxyCa.Certificate[0],
	})
	if err := os.WriteFile("hyperwapp-ca.crt", caCertPEM, 0644); err == nil {
		util.Info("Security Certificate (PEM) saved to: hyperwapp-ca.crt")
		util.Info("IMPORT this file into your browser (Authorities/Trusted Root) to fix SSL/Private errors.")
	}

	// Enable MITM for all HTTPS traffic to intercept headers/body
	proxy.OnRequest().HandleConnect(goproxy.AlwaysMitm)

	// Intercept responses
	proxy.OnResponse().DoFunc(func(resp *http.Response, ctx *goproxy.ProxyCtx) *http.Response {
		if resp == nil {
			return nil
		}

		// Clone body to read it without consuming it for the client
		var bodyBytes []byte
		if resp.Body != nil {
			bodyBytes, _ = io.ReadAll(resp.Body)
			resp.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
		}

		// Convert http.Header to map[string][]string
		headers := make(map[string][]string)
		for k, v := range resp.Header {
			headers[k] = v
		}

		input := model.OfflineInput{
			Domain:  resp.Request.URL.Hostname(),
			URL:     resp.Request.URL.String(),
			Headers: headers,
			Body:    bodyBytes,
			Path:    "proxy-stream", // Virtual path
		}

		outputCh <- input
		util.Debug("Proxy captured: %s", input.URL)

		return resp
	})

	util.Info("Starting Proxy Server on %s", addr)
	util.Info("Configure your browser to use this proxy to scan browsed pages.")
	
	server := &http.Server{Addr: addr, Handler: proxy}
	
	// Start server in background
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			util.Fatal("Proxy server failed: %v", err)
		}
	}()

	return nil
}
