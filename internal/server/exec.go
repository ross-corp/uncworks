package server

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"sync"

	"github.com/gorilla/websocket"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/remotecommand"
	runtimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// ExecHandler serves the interactive shell WebSocket endpoint.
type ExecHandler struct {
	k8sClient      runtimeclient.Client
	restConfig     *rest.Config
	namespace      string
	allowedOrigins []string
	wsUpgrader     websocket.Upgrader
}

// NewExecHandler creates a new ExecHandler with origin validation.
func NewExecHandler(k8sClient runtimeclient.Client, restConfig *rest.Config, namespace string, allowedOrigins []string) *ExecHandler {
	h := &ExecHandler{
		k8sClient:      k8sClient,
		restConfig:     restConfig,
		namespace:      namespace,
		allowedOrigins: allowedOrigins,
	}
	h.wsUpgrader = websocket.Upgrader{
		CheckOrigin: h.checkOrigin,
	}
	return h
}

// checkOrigin validates WebSocket upgrade requests against allowed origins.
func (e *ExecHandler) checkOrigin(r *http.Request) bool {
	origin := r.Header.Get("Origin")
	if origin == "" {
		return true // Non-browser clients (curl, etc.) don't send Origin.
	}
	for _, allowed := range e.allowedOrigins {
		if allowed == "*" || strings.EqualFold(allowed, origin) {
			return true
		}
	}
	return false
}

// RegisterExecHandlers registers the exec WebSocket endpoint on the given mux.
func (e *ExecHandler) RegisterExecHandlers(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/runs/{id}/exec", e.handleExec)
}

// resizeMessage is the JSON format for terminal resize messages from the client.
type resizeMessage struct {
	Type string `json:"type"`
	Cols uint16 `json:"cols"`
	Rows uint16 `json:"rows"`
}

// termSizeQueue implements remotecommand.TerminalSizeQueue for dynamic terminal resizing.
type termSizeQueue struct {
	sizes chan remotecommand.TerminalSize
}

func newTermSizeQueue() *termSizeQueue {
	return &termSizeQueue{
		sizes: make(chan remotecommand.TerminalSize, 4),
	}
}

func (q *termSizeQueue) Next() *remotecommand.TerminalSize {
	size, ok := <-q.sizes
	if !ok {
		return nil
	}
	return &size
}

// stdinPipe bridges WebSocket reads to the SPDY stdin stream.
type stdinPipe struct {
	data chan []byte
	buf  []byte
	done chan struct{}
}

func newStdinPipe() *stdinPipe {
	return &stdinPipe{
		data: make(chan []byte, 16),
		done: make(chan struct{}),
	}
}

func (p *stdinPipe) Read(dest []byte) (int, error) {
	// Drain remaining buffer first.
	if len(p.buf) > 0 {
		n := copy(dest, p.buf)
		p.buf = p.buf[n:]
		return n, nil
	}

	select {
	case chunk, ok := <-p.data:
		if !ok {
			return 0, io.EOF
		}
		n := copy(dest, chunk)
		if n < len(chunk) {
			p.buf = chunk[n:]
		}
		return n, nil
	case <-p.done:
		return 0, io.EOF
	}
}

func (p *stdinPipe) Close() {
	select {
	case <-p.done:
	default:
		close(p.done)
	}
}

func (e *ExecHandler) handleExec(w http.ResponseWriter, r *http.Request) {
	runID := r.PathValue("id")

	// Look up pod name from CRD.
	podName, err := e.lookupPodName(r.Context(), runID)
	if err != nil {
		http.Error(w, fmt.Sprintf("agent run %q not found: %v", runID, err), http.StatusNotFound)
		return
	}
	if podName == "" {
		http.Error(w, "pod not available for this agent run", http.StatusNotFound)
		return
	}

	// Upgrade to WebSocket.
	wsConn, err := e.wsUpgrader.Upgrade(w, r, nil)
	if err != nil {
		slog.Error("websocket upgrade failed", "err", err, "path", r.URL.Path)
		return
	}
	defer func() { _ = wsConn.Close() }()

	// Create SPDY exec session.
	clientset, err := kubernetes.NewForConfig(e.restConfig)
	if err != nil {
		slog.Error("create clientset failed", "err", err)
		_ = wsConn.WriteMessage(websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.CloseInternalServerErr, "k8s client error"))
		return
	}

	req := clientset.CoreV1().RESTClient().Post().
		Resource("pods").
		Name(podName).
		Namespace(e.namespace).
		SubResource("exec").
		VersionedParams(&corev1.PodExecOptions{
			Container: "rpc-gateway",
			Command:   []string{"bash", "-l"},
			Stdin:     true,
			Stdout:    true,
			Stderr:    true,
			TTY:       true,
		}, scheme.ParameterCodec)

	spdyExec, err := remotecommand.NewSPDYExecutor(e.restConfig, "POST", req.URL())
	if err != nil {
		slog.Error("create SPDY executor failed", "err", err)
		_ = wsConn.WriteMessage(websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.CloseInternalServerErr, "exec error"))
		return
	}

	sizeQueue := newTermSizeQueue()
	stdinWriter := newStdinPipe()
	stdoutReader, stdoutWriter := io.Pipe()

	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()

	var wg sync.WaitGroup

	// Goroutine: SPDY stream (blocks until session ends).
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer func() { _ = stdoutWriter.Close() }()
		defer stdinWriter.Close()
		err := spdyExec.StreamWithContext(ctx, remotecommand.StreamOptions{
			Stdin:             stdinWriter,
			Stdout:            stdoutWriter,
			Stderr:            stdoutWriter, // TTY merges stderr into stdout.
			Tty:               true,
			TerminalSizeQueue: sizeQueue,
		})
		if err != nil {
			slog.Debug("SPDY stream ended", "err", err)
		}
		cancel()
	}()

	// Goroutine: SPDY stdout → WebSocket writes.
	wg.Add(1)
	go func() {
		defer wg.Done()
		buf := make([]byte, 4096)
		for {
			n, err := stdoutReader.Read(buf)
			if n > 0 {
				if writeErr := wsConn.WriteMessage(websocket.BinaryMessage, buf[:n]); writeErr != nil {
					slog.Warn("websocket write error", "err", writeErr)
					cancel()
					return
				}
			}
			if err != nil {
				if err != io.EOF {
					slog.Warn("stdout read error", "err", err)
				}
				cancel()
				return
			}
		}
	}()

	// Main goroutine: WebSocket reads → SPDY stdin (or resize).
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer cancel()
		defer stdinWriter.Close()
		defer close(sizeQueue.sizes)

		for {
			msgType, msg, err := wsConn.ReadMessage()
			if err != nil {
				if !websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
					slog.Warn("websocket read error", "err", err)
				}
				return
			}

			if msgType == websocket.TextMessage && len(msg) > 0 && msg[0] == '{' {
				// Check if this is a resize message.
				var rm resizeMessage
				if json.Unmarshal(msg, &rm) == nil && rm.Type == "resize" {
					select {
					case sizeQueue.sizes <- remotecommand.TerminalSize{
						Width:  rm.Cols,
						Height: rm.Rows,
					}:
					default:
						// Drop resize if queue is full.
					}
					continue
				}
			}

			// Regular input data → stdin pipe.
			select {
			case stdinWriter.data <- msg:
			case <-ctx.Done():
				return
			}
		}
	}()

	// Wait for context cancellation (either side closed).
	<-ctx.Done()

	// Close WebSocket gracefully.
	_ = wsConn.WriteMessage(websocket.CloseMessage,
		websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))

	// Close pipes to unblock goroutines.
	_ = stdoutReader.Close()
	stdinWriter.Close()

	wg.Wait()
}

// lookupPodName delegates to the shared lookupRunningPod function.
func (e *ExecHandler) lookupPodName(ctx context.Context, runID string) (string, error) {
	return lookupRunningPod(ctx, e.k8sClient, e.namespace, runID)
}
