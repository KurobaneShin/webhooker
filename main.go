package main

import (
	"log"
	"os"
	"time"

	"github.com/gliderlabs/ssh"
	"github.com/teris-io/shortid"
	gossh "golang.org/x/crypto/ssh"
)

func main() {
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

	handler := SSHHandler{
		respCh: respCh,
	}

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

type SSHHandler struct {
	respCh chan string
}

func (h *SSHHandler) handleSSHSession(session ssh.Session) {
	forwardURL := session.RawCommand()
	_ = forwardURL
	resp := <-h.respCh

	session.Write([]byte(resp + "\n"))

	for data := range h.respCh {
		session.Write([]byte(data + "\n"))
	}
}
