# Cinema · vagas por sessão

App para Windows que consulta o site do Cinemark (sem login) e mostra quantas
poltronas restam em cada sessão de um filme, respeitando os seus filtros:
cidade, cinemas, datas, faixa de horário, tecnologia (2D, IMAX, XD…), áudio
(dublado/legendado), mínimo de assentos livres e **N assentos lado a lado**.

## Instalar

1. Baixe `cinema-windows-x64.zip` na [última release](../../releases/latest).
2. Extraia e rode `setup.exe`. Ele baixa a versão mais recente do `cinema.exe`,
   confere o checksum SHA-256, instala em `%LOCALAPPDATA%\Cinema` e cria atalhos.
   Se estiver sem internet, usa o `cinema.exe` que veio no zip.
3. Abra o **Cinema** pelo atalho. A janela abre no navegador; feche-a para encerrar o app.

O Windows SmartScreen pode avisar sobre "editor desconhecido", pois o executável não é assinado.

## Usar sem instalar (GitHub Codespaces)

Para um PC onde você não pode rodar programas:

1. No GitHub, abra o repositório → **Code** → aba **Codespaces** → **Create codespace on** a branch padrão.
2. Aguarde o ambiente subir (1 a 2 min na primeira vez). O app inicia sozinho na porta 8080.
3. Abra a aba **Portas** (Ports) do Codespaces, clique no ícone de globo da porta 8080 (**Cinema**) e use a página que abrir.
   A porta é privada por padrão: só você, logado no GitHub, acessa.
4. Se o app não estiver rodando, no terminal do Codespaces: `go run ./cmd/cinema -remote -addr 127.0.0.1:8080`.
5. Ao terminar, pare o Codespace (canto inferior esquerdo → *Stop*) para não gastar a cota gratuita.

Modo servidor (`-remote`): não abre navegador, não encerra sozinho e aceita qualquer Host
com a mesma origem. Ao escutar em endereço público (ex.: `0.0.0.0`), exige a senha na
variável `CINEMA_PASSWORD` (usuário livre, HTTP Basic).

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
