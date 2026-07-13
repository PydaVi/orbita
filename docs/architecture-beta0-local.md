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

Também apareceu um erro periódico (`"failed to enumerate network"`, HTTP 401) — é uma tentativa separada do Tap de enumerar repositórios pré-existentes por coleção, que exige auth que não configuramos; não afeta o firehose ao vivo, que conectou e entregou normalmente.

## O que ainda falta (não implementado)

- [x] Rodar o Tap de verdade, apontado pro `:2583` local, e confirmar que ele entrega webhook quando um novo `social.orbita.shelf.item` é escrito
- [x] `cmd/appview` ganhar um handler `/webhook` que recebe isso — só loga por enquanto, ainda não grava em banco
- [ ] Trocar o `curl` manual por código Go real (`atproto/auth/oauth` pro login, `atproto/lex`/`atproto/repo` pra montar e assinar o registro)
- [ ] Banco local (schema mínimo: `account` + a cópia indexada de `shelf.item`, como já descrito em `docs/BETA0-PLAN.md`) — os três eventos acima ainda só vão pro log, não pra uma tabela

Ver checklist completo e decisões em [`docs/BETA0-PLAN.md`](BETA0-PLAN.md).
