package main

import (
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
	l, host, err := app.Listen()
	if err != nil {
		fatal(err)
	}
	srv := app.NewServer(version)
	go func() {
		if err := http.Serve(l, srv.Handler(host)); err != nil {
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
