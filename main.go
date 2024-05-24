package main

import (
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"sync"

	"github.com/gliderlabs/ssh"
	gossh "golang.org/x/crypto/ssh"
	"golang.org/x/term"
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
	handler := newSshHandler()

	fwhandler := &ssh.ForwardedTCPHandler{}
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
		LocalPortForwardingCallback: ssh.LocalPortForwardingCallback(func(ctx ssh.Context, dhost string, dport uint32) bool {
			log.Println("Accepted forward", dhost, dport)
			return true
		}),
		ReversePortForwardingCallback: ssh.ReversePortForwardingCallback(func(ctx ssh.Context, host string, port uint32) bool {
			log.Println("attempt to bind", host, port, "granted")
			return true
		}),
		RequestHandlers: map[string]ssh.RequestHandler{
			"tcpip-forward":        fwhandler.HandleSSHRequest,
			"cancel-tcpip-forward": fwhandler.HandleSSHRequest,
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
	term := term.NewTerminal(session, "Welcome to webhooker\n\nPlease enter your webhook destination:\n")

	for {
		input, err := term.ReadLine()
		if err != nil {
			log.Fatal()
		}
		fmt.Println(input)

		if !strings.Contains(input, "-R") {
			generatedPort := randomPort()
			webhookUrlMessage := fmt.Sprintf("\nGeneratedWebhook: %d", generatedPort)
			command := fmt.Sprintf("\nCommand to copy:\nssh -R 127.0.0.1:%d:%s localhost -p 2222\n", generatedPort, input)
			term.Write([]byte(webhookUrlMessage + command))
			return
		}

	}
}

func randomPort() int {
	var (
		min = 49151
		max = 65535
	)

	return rand.Intn(max-min+1) + min
}
