# Cinema · vagas por sessão

App para Windows que consulta o site do Cinemark (sem login) e mostra quantas
poltronas restam em cada sessão de um filme nos cinemas **Flamboyant** e
**Passeio das Águas** (Goiânia), respeitando os seus filtros:
cinemas, datas, faixa de horário, tecnologia (2D, IMAX, XD…), áudio
(dublado/legendado), mínimo de assentos livres, **N assentos juntos** e intervalo de fileiras por número
(1 = a mais perto da tela, contando para o fundo em cada sala).

## Instalar

1. Baixe `cinema-windows-x64.zip` na [última release](../../releases/latest).
2. Extraia e rode `setup.exe`. Ele baixa a versão mais recente do `cinema.exe`,
   confere o checksum SHA-256, instala em `%LOCALAPPDATA%\Cinema` e cria atalhos.
   Se estiver sem internet, usa o `cinema.exe` que veio no zip.
3. Abra o **Cinema** pelo atalho. A janela abre no navegador; feche-a para encerrar o app.

O Windows SmartScreen pode avisar sobre "editor desconhecido", pois o executável não é assinado.

## Como funciona

`cinema.exe` é um único binário Go. Ele sobe um servidor apenas em `127.0.0.1`
(porta aleatória), abre a interface no navegador e consulta a API pública do
Cinemark (veja [docs/API.md](docs/API.md)). Nada é enviado a terceiros.

"Vagas" = assentos livres da sala. "Grupos de N" = quantos grupos de N poltronas
livres lado a lado, na mesma fileira, cabem na sessão. Assentos de acessibilidade
ficam de fora, a menos que você marque a opção.

## Desenvolver

```sh
go test ./...
go run ./cmd/cinema          # abre a interface local
```

## Publicar uma release

```sh
git tag v0.1.0 && git push origin v0.1.0
```

O workflow `.github/workflows/release.yml` roda os testes, compila `cinema.exe` e
`setup.exe` para Windows e publica `cinema-windows-x64.zip` (setup + executável),
`cinema.exe` (baixado pelo setup) e `SHA256SUMS.txt`.
Também dá para disparar manualmente em Actions → Release → Run workflow, informando a tag.

## Aviso

Projeto pessoal, sem relação com o Cinemark. Usa endpoints não documentados que
podem mudar a qualquer momento. Faça buscas com moderação.
