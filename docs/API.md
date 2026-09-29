# API do Cinemark (endpoints descobertos)

O site www.cinemark.com.br é um app Next.js que consome um BFF público, **sem login**:

```
https://br-www-frontend-ext-prod.cinemark.com.br/bff-api
```

Ele fica atrás do Cloudflare e só responde a requisições que parecem vir do site.
O cliente envia `Origin: https://www.cinemark.com.br`, `Referer: https://www.cinemark.com.br/`
e um `User-Agent` de navegador. Sem esses headers vem a página "Attention Required".

Todas as respostas são `{ "success": bool, "messageError": string, "dataResult": ... }`.

| Endpoint | Parâmetros | Retorno |
|---|---|---|
| `GET /v1/states` | — | `[{id, name, code}]` |
| `GET /v1/cities` | `stateId` | `[{id, name}]` |
| `GET /v1/theaters` | `cityId` | `[{code, name, city, state, ...}]` (`code` = id do cinema) |
| `GET /v1/movies/onDisplayByCity` | `cityId`, `pageNumber`, `pageSize` | `[{id, slug, name, ...}]` |
| `GET /v1/movies/preSaleByCity` | idem | filmes em pré-venda |
| `GET /v1/movies/detailBySlug` | `slug` | detalhes do filme |
| `GET /v1/sessions/movieAndCity` | `movieId`, `cityId`, `pageNumber`, `pageSize` | sessões, agrupadas por cinema e data |
| `GET /v1/seatmaps` | `theaterId`, `sessionId` (GUID) | mapa de assentos da sessão |

## Sessões

```json
{ "theaterId": 2120, "theaterName": "Mogi das Cruzes", "date": "2026-09-29T01:00:00",
  "rooms": [ { "number": 2, "features": [7], "audio": 20,
     "sessions": [ { "id": "<guid>", "date": "2026-09-29T15:30:00", "expired": false, "hybrid": false } ] } ] }
```

- `features` (tecnologia): 1 D-BOX, 2 XD, 3 IMAX, 4 Ingresso Azul, 5 Prime, 6 3D, 7 2D, 9 Cine Materna, 10 Infinity Vision
- `audio`: 10 Original, 20 Dublado, 30 Legendado
- `date` da sessão é horário local, sem fuso.

## Mapa de assentos

`dataResult.elements[]` com `{row, col, name, status, type, typeName, selectable}`.

- `status`: 1 livre, 2 indisponível, 3 ocupado
- `type`: 7 tela e 9 texto livre não são assentos; 5, 6, 8, 10, 11, 18, 21 são especiais
  (cadeirante, obeso, acompanhante, mobilidade reduzida); o restante (1 normal, 4 VIP,
  13/14 namoradeira esquerda/direita…) é assento comum.
- Uma sequência de assentos lado a lado = mesma `row` com `col` consecutivos.

## Cuidados

Use poucas requisições em paralelo (o app usa 6) e não faça polling agressivo.
Esses endpoints não são documentados oficialmente e podem mudar; os testes em
`testdata/` ajudam a perceber quando isso acontece.
