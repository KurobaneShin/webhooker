package main

import (
	"fmt"
	"log"
	"os"

	"github.com/gliderlabs/ssh"
	gossh "golang.org/x/crypto/ssh"
)

func main() {
	sshPort := ":2222"

	server := ssh.Server{
		Addr:    sshPort,
		Handler: handleSSHSession,
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

func handleSSHSession(session ssh.Session) {
	cmd := session.RawCommand()
	fmt.Println("handling conn in ssh session ->", cmd)
}
