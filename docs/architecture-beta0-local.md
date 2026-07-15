# Arquitetura do Beta 0 — ambiente local

> Documento didático: explica como as peças se encaixam no ambiente de desenvolvimento local, não é uma decisão nova (as decisões já estão em `docs/BETA0-PLAN.md`). Escrito depois de validar cada hop na mão, via `curl`, não só na teoria.

## Visão geral — do PDS local até o nosso appview

```
┌──────────────────────────────────────────┐
│  scripts/dev-pds/run.mjs (processo Node)  │
│                                            │
│   ┌──────────┐        ┌────────────────┐ │
│   │   PLC    │        │      PDS       │ │
│   │ :33195   │◄───────┤     :2583      │ │
│   │ (fake,   │  DID   │  (@atproto/pds │ │
│   │ em       │  ops   │   real, mesmo  │ │
│   │ memória) │        │   código do    │ │
│   └──────────┘        │   Bluesky)     │ │
│                        └───────┬────────┘ │
└────────────────────────────────┼──────────┘
                                  │ WebSocket cru
                                  │ com.atproto.sync.subscribeRepos
                                  ▼
                        ┌──────────────────┐
                        │    Tap (local)    │
                        │  aponta pra :2583  │
                        │  em vez do relay   │
                        │  de produção real  │
                        │                    │
                        │  filtra:           │
                        │  social.orbita.    │
                        │  shelf.item        │
                        └─────────┬──────────┘
                                  │ webhook (HTTP POST)
                                  ▼
                        ┌──────────────────┐
                        │  cmd/appview (Go)  │
                        │  handler /webhook  │
                        │  → banco local     │
                        └──────────────────┘
```

## Por que isso funciona sem relay nenhum

Um relay de verdade agrega o firehose de muitos PDSes e re-expõe isso como um stream único. O Tap não distingue relay de PDS individual — o código (`cmd/tap/firehose.go`) pega a URL configurada, troca o esquema por `ws`/`wss`, e gruda `xrpc/com.atproto.sync.subscribeRepos` nela, sempre. Como todo PDS já expõe esse mesmo path (é o firehose bruto dele, antes de qualquer agregação), apontar o Tap direto pro nosso PDS local funciona — é o caso degenerado onde "a rede inteira" e "uma fonte só" coincidem, porque só existe um PDS no nosso sandbox.

**Consequência importante:** o mesmo binário do Tap, sem nenhuma mudança de código, serve pros dois cenários — só muda a URL de configuração:

| Cenário | URL do Tap | O que ele vê |
|---|---|---|
| Dev local (este documento) | `http://localhost:2583` (nosso PDS) | só os registros que nós mesmos criamos no sandbox |
| Beta 0 "de verdade" (conta real da Bluesky) | `https://relay1.us-east.bsky.network` (padrão, sem configurar nada) | qualquer registro `social.orbita.shelf.item` escrito por qualquer conta real da rede |

## O que já validamos na mão (sem Go, sem Tap ainda)

Sequência real, executada via `curl` contra o PDS local:

1. `POST /xrpc/com.atproto.server.createAccount` → criou `did:plc:nuftb5ux5jsmfsitowhsu4ab`, com documento DID completo (`alsoKnownAs`, `verificationMethod`, `service` apontando pro `:2583`)
2. Token de acesso recebido tem header `{"typ":"at+jwt","alg":"HS256"}` — confirmando a domain-separation da spec de XRPC
3. `POST /xrpc/com.atproto.repo.createRecord` (`collection: social.orbita.shelf.item`) → devolveu `uri` (`at://did:plc:.../social.orbita.shelf.item/3mqgdrhodjk2i`) e `cid` — o par que vira strongRef quando outro registro precisar apontar pra este
4. `GET /xrpc/com.atproto.repo.getRecord` → leu o mesmo registro de volta, intacto

Detalhe registrado durante o teste: a resposta trouxe `"validationStatus": "unknown"` — o PDS aceita qualquer NSID sem validar contra o Lexicon, porque não tem de onde saber que o nosso schema existe. Validação de schema é responsabilidade do lado cliente, não do servidor.

## Pipeline validado de ponta a ponta

Rodamos o Tap de verdade (binário compilado via `go install github.com/bluesky-social/indigo/cmd/tap`), configurado assim:

```
TAP_PLC_URL=http://localhost:33195
TAP_RELAY_URL=http://localhost:2583        # nosso PDS local, não o relay real
TAP_SIGNAL_COLLECTION=social.orbita.shelf.item
TAP_COLLECTION_FILTERS=social.orbita.shelf.item
TAP_WEBHOOK_URL=http://localhost:8092/webhook
TAP_NO_REPLAY=true
```

Ao escrever um segundo registro (`workSlug: duna-parte-2`) com o Tap já conectado, o log mostrou o mecanismo de **backfill** de verdade: `"fetching repo from PDS"` → `"parsing repo CAR"` → `"iterating repo records"`. O Tap não entregou só o evento novo — foi buscar o repositório inteiro (exportado em CAR) porque essa era a primeira vez que ele via esse DID, e reprocessou tudo. Resultado: nosso `cmd/appview` recebeu **três** eventos no `/webhook`, não um:

```json
{"id":1,"type":"identity","identity":{"did":"did:plc:...","handle":"handle.invalid","is_active":true,"status":"active"}}
{"id":2,"type":"record","record":{"collection":"social.orbita.shelf.item","action":"create","record":{"workSlug":"matrix",...}}}
{"id":3,"type":"record","record":{"collection":"social.orbita.shelf.item","action":"create","record":{"workSlug":"duna-parte-2",...}}}
```

`id:2` é o registro `matrix`, escrito **antes** do Tap sequer existir — veio só por causa do backfill.

**`"handle": "handle.invalid"` não é bug.** Nosso handle de teste (`alice.test`) não é um domínio real, então a resolução bidirecional handle↔DID que estudamos (DNS TXT / `.well-known`, contra `alsoKnownAs` no documento DID) não tem como se confirmar — o Tap marca isso honestamente como inválido em vez de fingir que está tudo certo. É a mesma verificação de segurança da spec, funcionando.

## Schema do `work` mudou — repipeline confirmado

Depois da pesquisa de ecossistema (ver `docs/BETA0-PLAN.md`), `workSlug` (string livre) virou `work: {provider, id}` (referência externa mínima, ex.: `{"provider": "tmdb-movie", "id": "603"}`). Reescrevemos um registro novo com o schema atualizado e confirmamos o pipeline inteiro de novo, dessa vez com o Tap já rodando (não precisou de backfill):

```json
{"id":4,"type":"record","record":{"live":true,"collection":"social.orbita.shelf.item","action":"create","record":{"work":{"id":"603","provider":"tmdb-movie"},"createdAt":"2026-07-14T02:25:47.000Z"}}}
```

`"live": true` dessa vez — evento ao vivo de verdade, não backfill, confirmando que o pipeline reage a escrita nova em tempo real, não só na primeira descoberta do repositório. Os registros antigos (`workSlug: "matrix"`, `workSlug: "duna-parte-2"`) continuam no PDS local como dado órfão do schema anterior — sandbox descartável, sem necessidade de migração.

## OAuth real — por que o PDS local não serve pra isso, e a saga de rede pra testar com conta real

### O PDS local nunca ia funcionar aqui, por desenho

O `Resolver` do pacote `atproto/auth/oauth` (`resolver.go`) exige `https://` e proíbe porta explícita em três métodos (`ResolveAuthServerURL`, `ResolveAuthServerMetadata`, `ResolveClientMetadata`) — sem exceção configurável, é lógica fixa no código, não um campo trocável (o tipo é concreto, não interface). Isso não é sobre o `client_id` (que pode ser `http://localhost`, exceção de dev que já usamos) — é sobre o **servidor de autorização em si** nunca poder ser resolvido em HTTP puro com porta. Faz sentido: permitir isso geral abriria uma brecha real de SSRF. Conclusão: login OAuth só dá pra testar contra o **PDS real** — exatamente o papel que a decisão de identidades híbridas já previa pra essa situação.

### A saga pra alcançar o callback de volta (WSL2 + navegador)

Rodar o appview aqui (ambiente do assistente) não bastava — o `127.0.0.1:8092` daqui não é o `127.0.0.1:8092` que o navegador do autor enxerga, mesmo sendo a mesma máquina/WSL2 (confirmado por um teste: `bind: address already in use` provou que a rede *é* compartilhada nesse nível, então o problema estava adiante, entre o WSL2 e o navegador de fato).

Passo a passo do que aconteceu:
1. `http://127.0.0.1:8092/oauth/callback` como redirect_uri → `ERR_CONNECTION_REFUSED` no navegador, mesmo com o appview rodando no terminal do autor (não só aqui). Teste isolado (`http://127.0.0.1:8092/health` direto, sem OAuth) deu o mesmo erro — confirmando que o problema era puramente de rede, nada a ver com OAuth.
2. Achado empírico do autor: `http://localhost:8092/health` **funcionava**, `127.0.0.1` não — causa exata não identificada (hipótese: proxy/VPN local com regra de bypass só pro nome "localhost", não pro IP literal).
3. Trocamos o redirect_uri pra `http://localhost:8092/oauth/callback` → PAR foi **recusado pelo servidor real da Bluesky** (`HTTP 400 invalid_request`) — a spec só aceita as formas literais `127.0.0.1`/`[::1]`, "localhost" como texto não é uma delas, e o servidor валida isso de verdade.
4. Hipótese seguinte: se "localhost" resolve e "127.0.0.1" não, talvez o ambiente prefira IPv6 — testamos `http://[::1]:8092/oauth/callback`. **Funcionou nos dois lados**: PAR aceito pela Bluesky (forma literal válida) *e* alcançável pelo navegador do autor.

Login completo, ponta a ponta, contra `pydavi.bsky.social` de verdade: `did:plc:kpsswg4vfyzjvxp577wsqh3t` (confirmado batendo com `com.atproto.identity.resolveHandle` contra a API pública da Bluesky).

**Lição pra quem repetir isso em outra máquina:** se `127.0.0.1` não alcançar o callback, tente `[::1]` antes de mexer em configuração de rede do WSL2/Windows (`.wslconfig`, `netsh portproxy`) — pode ser só isso.

Também apareceu um erro periódico (`"failed to enumerate network"`, HTTP 401) — é uma tentativa separada do Tap de enumerar repositórios pré-existentes por coleção, que exige auth que não configuramos; não afeta o firehose ao vivo, que conectou e entregou normalmente.

## Escrita real via OAuth — confirmada na rede de produção

`cmd/appview/oauth.go` + `cmd/appview/shelf.go` substituem o `curl` manual: login real (`StartAuthFlow`/`ProcessCallback`, PAR+PKCE+DPoP por dentro da lib) e escrita autenticada (`oauthSess.APIClient().Post(ctx, "com.atproto.repo.createRecord", ...)`). Testado contra a conta real do autor (`pydavi.bsky.social`, não o PDS local — motivo na seção acima), e **confirmado na rede**, não só pela tela de sucesso:

```
GET .../xrpc/com.atproto.repo.listRecords?repo=did:plc:kpsswg4vfyzjvxp577wsqh3t&collection=social.orbita.shelf.item
→ at://did:plc:kpsswg4vfyzjvxp577wsqh3t/social.orbita.shelf.item/3mqlbnf4e7m2e
```

Primeiro dado real da Órbita no AT Protocol — não sandbox, não backfill, escrito pelo nosso próprio código Go.

## Banco local e listagem — pipeline completo confirmado no sandbox

`cmd/appview/db.go` (SQLite puro-Go, `modernc.org/sqlite`, tabela `shelf_items`) + `webhook.go` reescrito pra parsear o evento real do Tap (`type: "record"`, `action: "create"`, coleção `social.orbita.shelf.item`) e indexar, em vez de só logar. `list.go` expõe `GET /shelf`, lendo direto do banco.

Testado de ponta a ponta contra o PDS local (registro novo, TMDB tv id `1396` — Breaking Bad): escrita → Tap (já rastreando o DID de teste) → webhook → `INSERT` no SQLite → `GET /shelf` mostrando o item, tudo no mesmo ciclo, sem intervenção manual em nenhum hop intermediário.

## Tap contra o relay real — o último item, confirmado

Subimos uma segunda instância do Tap, **sem configurar `TAP_RELAY_URL` nem `TAP_PLC_URL`** — os defaults já são o relay de produção (`https://relay1.us-east.bsky.network`) e o `plc.directory` real. Só `TAP_SIGNAL_COLLECTION`, `TAP_COLLECTION_FILTERS` e `TAP_WEBHOOK_URL` (apontado pro mesmo `:8092`), mais `TAP_BIND` diferente pra não colidir com a instância local.

O log mostrou algo que a instância local não conseguiu (lá, a "enumeração" falhava com 401 — nosso PLC fake não implementa esse endpoint): contra a rede real, a enumeração **funcionou**:

```
"enumerated repos by collection batch" collection=social.orbita.shelf.item count=1
"finished enumerating network, sleeping for 1 day"
"starting resync" did=did:plc:kpsswg4vfyzjvxp577wsqh3t
"fetching repo from PDS" pds=https://agaric.us-west.host.bsky.network
```

Achou a conta real do autor (a única com registro na nossa coleção, contagem 1), buscou o repositório de verdade no PDS de produção, fez backfill — sem precisar de nenhuma escrita nova. `GET /shelf` passou a mostrar os dois registros lado a lado, sandbox e produção:

```
tmdb-tv/1396    — did:plc:nuftb5ux5jsmfsitowhsu4ab   (sandbox local)
tmdb-movie/603  — did:plc:kpsswg4vfyzjvxp577wsqh3t   (rede real, pydavi.bsky.social)
```

**Mesmo binário, mesmo código Go, zero mudança de lógica** — só a URL de configuração diferenciou "brincar no sandbox" de "funcionar de verdade contra a rede inteira do Bluesky". É a prova mais concreta que temos de que o desenho (AppView como view derivada, reconstruível, nunca dona do dado) funciona igual nos dois mundos.

## Beta 0 — concluído

- [x] Rodar o Tap de verdade, apontado pro `:2583` local, e confirmar que ele entrega webhook quando um novo `social.orbita.shelf.item` é escrito
- [x] `cmd/appview` ganhar um handler `/webhook` que recebe isso e indexa num banco local
- [x] Trocar o `curl` manual por código Go real — OAuth completo, escrita confirmada na rede real
- [x] Banco local — SQLite, tabela `shelf_items`, indexação automática via webhook
- [x] Página simples listando o que foi sincronizado — `GET /shelf`
- [x] Tap apontado pro relay real — confirmado, enumeração + backfill funcionando contra a rede de produção

Ver checklist completo e decisões em [`docs/BETA0-PLAN.md`](BETA0-PLAN.md).
