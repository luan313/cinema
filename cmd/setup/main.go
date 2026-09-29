// setup baixa a versão mais recente do cinema.exe (GitHub Releases), confere o
// checksum, instala em %LOCALAPPDATA%\Cinema e cria atalhos.
package main

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

const (
	repo      = "luan313/cinema"
	appExe    = "cinema.exe"
	sumsAsset = "SHA256SUMS.txt"
)

type release struct {
	Tag    string  `json:"tag_name"`
	Assets []asset `json:"assets"`
}

type asset struct {
	Name string `json:"name"`
	URL  string `json:"browser_download_url"`
	Size int64  `json:"size"`
}

func (r release) find(name string) (asset, bool) {
	for _, a := range r.Assets {
		if a.Name == name {
			return a, true
		}
	}
	return asset{}, false
}

// parseSums lê linhas "<sha256>  <arquivo>" (formato do sha256sum).
func parseSums(text, file string) string {
	for _, line := range strings.Split(text, "\n") {
		f := strings.Fields(line)
		if len(f) == 2 && strings.TrimPrefix(f[1], "*") == file {
			return strings.ToLower(f[0])
		}
	}
	return ""
}

func main() {
	err := run()
	if err != nil {
		fmt.Println("\nERRO:", err)
	}
	fmt.Print("\nPressione Enter para fechar...")
	bufio.NewReader(os.Stdin).ReadString('\n')
	if err != nil {
		os.Exit(1)
	}
}

func run() error {
	fmt.Println("Instalador do Cinema")
	dir := installDir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	dest := filepath.Join(dir, appExe)
	tmp := dest + ".download"

	tag, err := download(tmp)
	if err != nil {
		fmt.Println("Download indisponível:", err)
		local, lerr := localCopy()
		if lerr != nil {
			return fmt.Errorf("não foi possível baixar e não há %s ao lado do setup: %w", appExe, lerr)
		}
		fmt.Println("Usando o", appExe, "que veio junto no zip.")
		if err := copyFile(local, tmp); err != nil {
			return err
		}
		tag = "local"
	}
	if err := os.Rename(tmp, dest); err != nil {
		os.Remove(dest)
		if err := os.Rename(tmp, dest); err != nil {
			return fmt.Errorf("instalar em %s (o app está aberto?): %w", dest, err)
		}
	}
	fmt.Printf("Instalado (%s): %s\n", tag, dest)
	if runtime.GOOS == "windows" {
		if err := shortcuts(dest); err != nil {
			fmt.Println("Aviso: não foi possível criar atalhos:", err)
		} else {
			fmt.Println("Atalhos criados na Área de Trabalho e no Menu Iniciar.")
		}
	}
	fmt.Print("Abrir o Cinema agora? [S/n] ")
	ans, _ := bufio.NewReader(os.Stdin).ReadString('\n')
	if a := strings.ToLower(strings.TrimSpace(ans)); a == "" || a == "s" || a == "y" {
		return exec.Command(dest).Start()
	}
	return nil
}

func installDir() string {
	if base := os.Getenv("LOCALAPPDATA"); base != "" {
		return filepath.Join(base, "Cinema")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".cinema")
}

func localCopy() (string, error) {
	self, err := os.Executable()
	if err != nil {
		return "", err
	}
	p := filepath.Join(filepath.Dir(self), appExe)
	if _, err := os.Stat(p); err != nil {
		return "", err
	}
	return p, nil
}

func getJSON(url string, out any) error {
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "cinema-setup")
	c := &http.Client{Timeout: 30 * time.Second}
	resp, err := c.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("GitHub respondeu HTTP %d", resp.StatusCode)
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

func fetchText(url string) (string, error) {
	c := &http.Client{Timeout: 30 * time.Second}
	resp, err := c.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	return string(b), err
}

// download baixa a última release para path e devolve a tag.
func download(path string) (string, error) {
	fmt.Println("Procurando a última versão…")
	var rel release
	if err := getJSON("https://api.github.com/repos/"+repo+"/releases/latest", &rel); err != nil {
		return "", err
	}
	a, ok := rel.find(appExe)
	if !ok {
		return "", fmt.Errorf("a release %s não tem %s", rel.Tag, appExe)
	}
	want := ""
	if s, ok := rel.find(sumsAsset); ok {
		txt, err := fetchText(s.URL)
		if err != nil {
			return "", err
		}
		want = parseSums(txt, appExe)
	}
	fmt.Printf("Baixando %s %s (%.1f MB)…\n", appExe, rel.Tag, float64(a.Size)/1e6)
	resp, err := (&http.Client{Timeout: 10 * time.Minute}).Get(a.URL)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("download HTTP %d", resp.StatusCode)
	}
	f, err := os.Create(path)
	if err != nil {
		return "", err
	}
	h := sha256.New()
	_, err = io.Copy(io.MultiWriter(f, h), resp.Body)
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		os.Remove(path)
		return "", err
	}
	if want != "" && hex.EncodeToString(h.Sum(nil)) != want {
		os.Remove(path)
		return "", errors.New("checksum SHA-256 não confere; download descartado")
	}
	return rel.Tag, nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return err
	}
	return out.Close()
}

func shortcuts(target string) error {
	script := fmt.Sprintf(`$s=New-Object -ComObject WScript.Shell;
foreach($d in @([Environment]::GetFolderPath('Desktop'),[Environment]::GetFolderPath('Programs'))){
 $l=$s.CreateShortcut((Join-Path $d 'Cinema.lnk')); $l.TargetPath='%s'; $l.WorkingDirectory='%s'; $l.Save() }`,
		strings.ReplaceAll(target, "'", "''"), strings.ReplaceAll(filepath.Dir(target), "'", "''"))
	out, err := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", script).CombinedOutput()
	if err != nil {
		return fmt.Errorf("%v: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}
