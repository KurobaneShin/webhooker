package main

import (
	"io"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/gliderlabs/ssh"
	"github.com/teris-io/shortid"
	gossh "golang.org/x/crypto/ssh"
)

var clients sync.Map

type HTTPHandler struct{}

func (h *HTTPHandler) handleWebhook(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	ch, ok := clients.Load(id)

	if !ok {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("client id not found"))
		return
	}

	b, err := io.ReadAll(r.Body)
	if err != nil {
		log.Fatal(err)
	}

	defer r.Body.Close()

	ch.(chan string) <- string(b)
}

func startHttpServer() error {
	port := ":5000"
	router := http.NewServeMux()

	handle := &HTTPHandler{}

	router.HandleFunc("/{id}/*", handle.handleWebhook)

	return http.ListenAndServe(port, router)
}

func startSshServer() {
	sshPort := ":2222"

	respCh := make(chan string)

	go func() {
		time.Sleep(time.Second * 3)
		id, _ := shortid.Generate()
		respCh <- "http://webhooker.com/" + id

		time.Sleep(time.Second * 5)

		for {
			time.Sleep(time.Second * 2)
			respCh <- "webhook data received"
		}
	}()

	handler := newSshHandler()

	server := ssh.Server{
		Addr:    sshPort,
		Handler: handler.handleSSHSession,
		ServerConfigCallback: func(ctx ssh.Context) *gossh.ServerConfig {
			cfg := &gossh.ServerConfig{
				ServerVersion: "SSH-2.0-sendit",
			}

			cfg.Ciphers = []string{"chacha20-poly1305@openssh.com"}
			return cfg
		},
		PublicKeyHandler: func(ctx ssh.Context, key ssh.PublicKey) bool {
			return true
		},
	}

	b, err := os.ReadFile("keys/privatekey")
	if err != nil {
		log.Fatal(err)
	}

	privateKey, err := gossh.ParsePrivateKey(b)
	if err != nil {
		log.Fatal("failed to parse private key:", err)
	}

	server.AddHostKey(privateKey)

	log.Fatal(server.ListenAndServe())
}

func main() {
	go startSshServer()
	startHttpServer()
}

type SSHHandler struct {
	channels map[string]chan string
}

func newSshHandler() *SSHHandler {
	return &SSHHandler{
		channels: make(map[string]chan string),
	}
}

func (h *SSHHandler) handleSSHSession(session ssh.Session) {
	cmd := session.RawCommand()

	if cmd == "init" {

		id := shortid.MustGenerate()

		webhookUrl := "http://webhooker.com/" + id + "\n"
		session.Write([]byte(webhookUrl))
		respCh := make(chan string)
		h.channels[id] = respCh
		clients.Store(id, respCh)
		return
	}

	if len(cmd) > 0 {
		respCh, ok := h.channels[cmd]
		if !ok {
			session.Write([]byte("invalid webhook Id\n"))
		}

		for data := range respCh {
			session.Write([]byte(data + "\n"))
		}
	}
}
