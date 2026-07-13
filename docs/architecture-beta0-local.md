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

## O que ainda falta (não implementado)

- [ ] Rodar o Tap de verdade, apontado pro `:2583` local, e confirmar que ele entrega webhook quando um novo `social.orbita.shelf.item` é escrito
- [ ] `cmd/appview` ganhar um handler `/webhook` que recebe isso e grava no banco local
- [ ] Trocar o `curl` manual por código Go real (`atproto/auth/oauth` pro login, `atproto/lex`/`atproto/repo` pra montar e assinar o registro)
- [ ] Banco local (schema mínimo: `account` + a cópia indexada de `shelf.item`, como já descrito em `docs/BETA0-PLAN.md`)

Ver checklist completo e decisões em [`docs/BETA0-PLAN.md`](BETA0-PLAN.md).
