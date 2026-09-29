package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"

	"github.com/luan313/cinema/internal/app"
)

// version é preenchida no build: -ldflags "-X main.version=v1.0.0".
var version = "dev"

func main() {
	addr := flag.String("addr", "127.0.0.1:0", "endereço para escutar (127.0.0.1:0 = porta local aleatória)")
	remote := flag.Bool("remote", false, "modo servidor (Codespaces/hospedagem): não abre o navegador, não encerra sozinho e aceita qualquer Host")
	flag.Parse()

	// A senha vem do ambiente para não aparecer na lista de processos.
	password := os.Getenv("CINEMA_PASSWORD")
	if *remote && !app.IsLoopback(*addr) && password == "" {
		fatal(fmt.Errorf("em modo servidor com endereço público (%s) defina a senha em CINEMA_PASSWORD", *addr))
	}

	l, host, err := app.Listen(*addr)
	if err != nil {
		fatal(err)
	}
	srv := app.NewServer(version)
	if *remote {
		cfg := app.Config{Password: password}
		fmt.Println("Cinema (modo servidor) escutando em", host)
		if err := http.Serve(l, srv.Handler(cfg)); err != nil {
			fatal(err)
		}
		return
	}

	cfg := app.Config{Host: host}
	go func() {
		if err := http.Serve(l, srv.Handler(cfg)); err != nil {
			fatal(err)
		}
	}()
	openBrowser("http://" + host + "/")
	fmt.Println("Cinema rodando em http://" + host + "/ — feche a janela do navegador para encerrar.")
	srv.WaitIdle(150 * time.Second) // abas em segundo plano reduzem a frequência do ping
}

func openBrowser(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	case "darwin":
		cmd = exec.Command("open", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	_ = cmd.Start()
}

// Com -H=windowsgui não há console; registra o erro num arquivo.
func fatal(err error) {
	_ = os.WriteFile(filepath.Join(os.TempDir(), "cinema-erro.log"), []byte(err.Error()+"\n"), 0o644)
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
