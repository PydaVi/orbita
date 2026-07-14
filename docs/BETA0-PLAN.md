# Beta 0 — rascunho de planejamento

**Status:** decisões da primeira rodada fechadas (stack, licença, identidades de teste, critério de conclusão, identificação de obra). Continua sendo um documento vivo — decisão fechada aqui significa "o suficiente pra começar a escrever código", não "impossível de revisitar". Desenvolvido com uso ativo de IA sob revisão direta — ver "Uso de IA no desenvolvimento" no [`README.md`](../README.md).

## Objetivo

No mesmo espírito de um Beta 0 clássico ("produto antes de infraestrutura"), o objetivo aqui é provar a menor fatia possível da Órbita rodando de ponta a ponta sobre AT Protocol real — antes de qualquer ambição maior (PDS próprio, relay, firehose, múltiplos tipos de registro, federação de verdade entre AppViews).

Sentir o problema mínimo primeiro: autenticar contra uma identidade que não é nossa, escrever um registro num repositório que não controlamos, e ler esse dado de volta.

## Progresso

- [x] Lexicon `social.orbita.shelf.item` — [`lexicons/social/orbita/shelf/item.json`](../lexicons/social/orbita/shelf/item.json)
- [x] Esqueleto do módulo Go (módulo único, `go 1.25.0`) — [`cmd/appview/main.go`](../cmd/appview/main.go), só `/health` por enquanto
- [ ] OAuth contra uma conta real (`atproto/auth/oauth`)
- [ ] Escrita do registro no PDS via sessão autenticada
- [x] PDS de desenvolvimento local — [`scripts/dev-pds/run.mjs`](../scripts/dev-pds/run.mjs), via `@atproto/dev-env` (`TestNetworkNoAppView`: só PLC + PDS, sem Bsky AppView/Ozone/Postgres)
- [x] Ciclo manual completo validado via `curl` (criar conta → escrever `shelf.item` → ler de volta) — ver [`docs/architecture-beta0-local.md`](architecture-beta0-local.md)
- [x] Webhook + consumo do Tap, filtrado pra `social.orbita.shelf.item` — rodado de verdade, backfill confirmado, ver [`docs/architecture-beta0-local.md`](architecture-beta0-local.md)
- [ ] Banco local (indexação da cópia sincronizada — ver "Onde cada dado mora" abaixo)
- [ ] Página simples listando o que foi sincronizado

## Onde cada dado mora

Distinção que vale manter fixada em código, não só em conversa: **PDS é a fonte da verdade** (o registro que a pessoa autorou, no repositório dela); **AppView é view derivada, descartável e reconstruível** (nossa cópia indexada, nunca autoritativa — mesmo papel que um cache Redis já cumpre em qualquer backend tradicional, só que agora a "fonte da verdade" também saiu do nosso controle). Mesmo a própria escrita do usuário logado passa pelo Tap antes de aparecer no nosso banco — não existe atalho de gravação direta local, nem pros próprios dados de quem está usando.

## Referência de estudo

Tutorial oficial **Statusphere** (`atproto.com/guides/statusphere-tutorial`) é a referência mais próxima de um "Beta 0" que a própria documentação do AT Protocol oferece. Arquitetura confirmada (verificada em duas fontes — página do tutorial e repositório de exemplo):

- **OAuth** contra o PDS que a pessoa já tem (não hospedamos PDS próprio neste beta) — escopo de permissão restrito ao Lexicon customizado
- **Lexicon customizado** — schema do registro, versionado, com codegen de tipos
- **Sincronização em tempo real** via **Tap** (`github.com/bluesky-social/indigo/cmd/tap`) — ferramenta que assiste ao stream da rede filtrando por coleção e entrega eventos via webhook; é o que substitui consumir Jetstream/firehose na mão neste estágio
- **Banco local** (SQLite via Kysely, no tutorial de referência) — a AppView só indexa o que já foi sincronizado, nunca consulta o PDS ao vivo numa request de leitura
- **Frontend mínimo** — só o suficiente pra provar que o dado voltou

## O que muda em relação ao Statusphere

Statusphere usa um Lexicon só (`xyz.statusphere.status`, um emoji como status). Pra Órbita, o gesto equivalente mais próximo do "gesto fundamental" do produto — a estante sempre foi descrita como a ação mais importante da Órbita — é um primeiro Lexicon próprio: `social.orbita.shelf.item`, já escrito e validado contra exemplos reais (`xyz.statusphere.status`, `app.bsky.feed.like`) em [`lexicons/social/orbita/shelf/item.json`](../lexicons/social/orbita/shelf/item.json).

Sem nota, sem afinidade, sem constelação, sem tipo de obra — só o gesto de adicionar algo à estante.

## Decisões já fechadas

1. **Stack: Go.** Confirmado que não há impedimento técnico — `github.com/bluesky-social/indigo` é o monorepo Go oficial do Bluesky/AT Protocol (o mesmo de onde vem o Tap) e cobre exatamente o que este beta precisa: `atproto/auth/oauth` (cliente OAuth), `atproto/identity` (resolução de DID/handle), `atproto/lexicon` (validação de schema), `atproto/repo` (estrutura de repositório), `atproto/atcrypto` (assinatura/criptografia). Não é um workaround — é a implementação de referência, a mesma que roda a infraestrutura real do Bluesky.
   - **Risco aceito e documentado, não escondido:** o próprio Indigo se declara em desenvolvimento ativo — "features and software interfaces have not stabilized and may break or be removed". Ou seja: esperar alguma quebra de API ao atualizar dependências, e pinar versão explicitamente desde o primeiro `go.mod`. Mesmo espírito de risco já aceito conscientemente em decisões de dependência anteriores (ex.: rate-gate de API externa, cache fail-open) — nomeado aqui, não descoberto depois.

3. **Identidades de teste: híbrido.** Dois ambientes, propósitos diferentes:
   - **PDS de desenvolvimento local — já rodando.** `indigo/cmd/pds` não existe (correção: não é uma ferramenta Go, é suposição errada que eu tinha feito). A ferramenta real é `@atproto/dev-env` (pacote npm do próprio time do Bluesky) — mas o binário publicado (`bin.js`) sobe a rede de teste inteira deles (PDS + Bsky AppView + Ozone + Bsync, exigindo Postgres pro schema do AppView), pesado demais pro que precisamos. Usamos em vez disso a classe `TestNetworkNoAppView`, que sobe só PLC + PDS, sem Postgres — script próprio em [`scripts/dev-pds/run.mjs`](../scripts/dev-pds/run.mjs). Rápido, descartável, sem rate limit, sem sujar a rede real com registro de teste. Mesmo papel que um Postgres/Redis local cumprem em qualquer backend tradicional.
   - **Conta(s) reais da Bluesky** para validação periódica de interoperabilidade de verdade — confirmar que um registro `social.orbita.shelf.item` escrito por este código sobrevive num PDS de produção real, não só no ambiente controlado. Tecnicamente possível sem fricção: o protocolo não exige aprovação da Bluesky pra escrever um NSID customizado no repositório de alguém — é exatamente esse o ponto do AT Protocol.
   - Critério prático: o Beta 0 só conta como validado (item 5) quando passar nos dois ambientes, não só no local.

4. **Licença: AGPL-3.0.** Mesma escolha do Mastodon, e pelo mesmo motivo específico: a cláusula de uso em rede fecha a brecha que o GPL comum deixa aberto — sem ela, alguém poderia pegar o código, modificar, e operar como serviço hospedado sem nunca precisar distribuir as modificações (usuário só interage pela rede, nunca recebe uma cópia do software). AGPL obriga a disponibilizar o código modificado a quem usa o serviço pela rede, não só a quem recebe uma cópia binária. É a proteção certa contra "alguém fecha isso e vende" sem impedir uso/estudo/fork legítimo.

5. **Critério de "Beta 0 concluído"** — confirmado: login via OAuth funcionando contra uma conta real, um registro `social.orbita.shelf.item` criado no PDS dessa conta, Tap sincronizando esse registro pra um banco local, e uma página simples listando o que foi sincronizado. Sem UI além disso, sem segundo Lexicon, sem afinidade.

2. **Identificação da obra: migrada pro formato `{provider, id}`.** Substituiu a string livre (`workSlug`) — decisão original aceita como intermediária, revista depois da pesquisa de ecossistema abaixo. Já implementado e validado de ponta a ponta contra o PDS local (`work: {provider: "tmdb-movie", id: "603"}` — o ID real do Matrix na TMDB), incluindo entrega via Tap/webhook com `"live": true`. Ver schema completo em [`lexicons/social/orbita/shelf/item.json`](../lexicons/social/orbita/shelf/item.json).

## Pesquisa de ecossistema: o que Popfeed e Skylights ensinam

Antes de fechar a direção pra identificação de obra, pesquisamos dois apps reais de mídia no AT Protocol e consultamos a rede de verdade (não só documentação secundária) — `com.atproto.identity.resolveHandle`, `plc.directory`, `com.atproto.repo.describeRepo` e `listRecords` contra a conta real `popfeed.social`, mais os lexicons públicos da Skylights (`github.com/Gregoor/skylights`).

**Três jeitos diferentes de referenciar uma obra, encontrados na prática:**

1. **Popfeed duplica tudo** — cada `social.popfeed.feed.listItem` carrega título, gêneros, poster, data de lançamento inteiros, correlacionados só por um `identifiers.tmdbId`/`igdbId` solto. Sem registro canônico.
2. **Skylights usa referência externa mínima** — `{"ref": "tmdb:m", "value": "603"}` dentro do próprio registro do usuário. Sem duplicar metadado, sem inventar um segundo record type.
3. **O que a gente tinha planejado** (strongRef pra um `social.orbita.work` publicado por conta de serviço própria) — nenhuma das duas apps reais faz isso.

**Decisão: seguimos o padrão da Skylights**, não o nosso plano antigo. Ele já é quase idêntico à resolução externa que a Órbita original valida (TMDB/MusicBrainz/Open Library, busca → normaliza → identificador estável) — só muda o formato de exposição, virando um par `{provider, id}` dentro do próprio `shelf.item`, em vez de string livre solta ou de um record novo. **Elimina de vez a necessidade de conta de serviço, self-host de PDS próprio, e o record type `social.orbita.work`** — nenhuma dessas três coisas é necessária. Ver detalhe do novo schema proposto abaixo.

**Achado extra que valida os princípios do produto, não só a arquitetura:** uma crítica de UX externa ao Popfeed (não afiliada) aponta que o app fica "entre duas cadeiras" — tracker (tipo Trakt) e rede social (tipo Letterboxd) ao mesmo tempo, herdando scroll infinito e lógica de feed do Bluesky de um jeito que atrapalha quem só quer registrar consumo. É exatamente o problema que os princípios 2 e 4 do produto original (sem engajamento algorítmico, sem scroll infinito, hierarquia obra > pessoa) evitam por desenho — validação concreta, não hipotética.

**O que existe no ecossistema e não deveríamos copiar:** `social.popfeed.challenge.*` é gamificação (desafios/metas de consumo) — contraria diretamente o princípio 4 ("sem design viciante"). Também achamos que **nem a própria Popfeed self-hosta a conta de serviço dela** (PDS deles está em `*.host.bsky.network`, infra da própria Bluesky) — reforça que self-hostar não era necessário mesmo antes de descartarmos essa ideia por outro motivo.

## Identificação de obra — schema revisado (substitui o plano de strongRef), já implementado

Migrado imediatamente, não adiado — não exigia infra nova, só formato de campo diferente. Seguindo o padrão idiomático confirmado na Skylights (`{"type": "ref", "ref": "#work"}` apontando pra um def local, não objeto inline — verificado contra `rel.json` deles antes de escrever, já que a spec do Lexicon não deixa claro se aninhamento inline sem `ref` é válido):

```json
"work": { "type": "ref", "ref": "#work" }
// ...
"work": {
  "type": "object",
  "required": ["provider", "id"],
  "properties": {
    "provider": { "type": "string", "knownValues": ["tmdb-movie", "tmdb-tv", "musicbrainz", "open-library"] },
    "id": { "type": "string", "minLength": 1, "maxLength": 200 }
  }
}
```

Os registros antigos no PDS local (`workSlug: "matrix"`, `workSlug: "duna-parte-2"`) ficam pra trás como dado órfão do schema anterior — sandbox é descartável de propósito, sem necessidade de migração.
