# Beta 0 — rascunho de planejamento

**Status:** decisões da primeira rodada fechadas (stack, licença, identidades de teste, critério de conclusão, identificação de obra). Continua sendo um documento vivo — decisão fechada aqui significa "o suficiente pra começar a escrever código", não "impossível de revisitar".

## Objetivo

No mesmo espírito do Beta 0 de `comum` ("produto antes de infraestrutura"), o objetivo aqui é provar a menor fatia possível da Órbita rodando de ponta a ponta sobre AT Protocol real — antes de qualquer ambição maior (PDS próprio, relay, firehose, múltiplos tipos de registro, federação de verdade entre AppViews).

Sentir o problema mínimo primeiro: autenticar contra uma identidade que não é nossa, escrever um registro num repositório que não controlamos, e ler esse dado de volta.

## Referência de estudo

Tutorial oficial **Statusphere** (`atproto.com/guides/statusphere-tutorial`) é a referência mais próxima de um "Beta 0" que a própria documentação do AT Protocol oferece. Arquitetura confirmada (verificada em duas fontes — página do tutorial e repositório de exemplo):

- **OAuth** contra o PDS que a pessoa já tem (não hospedamos PDS próprio neste beta) — escopo de permissão restrito ao Lexicon customizado
- **Lexicon customizado** — schema do registro, versionado, com codegen de tipos
- **Sincronização em tempo real** via **Tap** (`github.com/bluesky-social/indigo/cmd/tap`) — ferramenta que assiste ao stream da rede filtrando por coleção e entrega eventos via webhook; é o que substitui consumir Jetstream/firehose na mão neste estágio
- **Banco local** (SQLite via Kysely, no tutorial de referência) — a AppView só indexa o que já foi sincronizado, nunca consulta o PDS ao vivo numa request de leitura
- **Frontend mínimo** — só o suficiente pra provar que o dado voltou

## O que muda em relação ao Statusphere

Statusphere usa um Lexicon só (`xyz.statusphere.status`, um emoji como status). Pra Órbita, o gesto equivalente mais próximo do "gesto fundamental" do produto — o próprio `CLAUDE.md` de `comum` chama a estante de "a ação mais importante da Órbita" — seria um primeiro Lexicon próprio: `social.orbita.shelf.item`.

Rascunho de schema mínimo (a decidir):
```
social.orbita.shelf.item
  workSlug: string   # sem resolver contra catálogo nenhum ainda neste beta
  createdAt: datetime
```

Note: sem nota, sem afinidade, sem constelação, sem tipo de obra — só o gesto de adicionar algo à estante.

## Decisões já fechadas

1. **Stack: Go.** Confirmado que não há impedimento técnico — `github.com/bluesky-social/indigo` é o monorepo Go oficial do Bluesky/AT Protocol (o mesmo de onde vem o Tap) e cobre exatamente o que este beta precisa: `atproto/auth/oauth` (cliente OAuth), `atproto/identity` (resolução de DID/handle), `atproto/lexicon` (validação de schema), `atproto/repo` (estrutura de repositório), `atproto/atcrypto` (assinatura/criptografia). Não é um workaround — é a implementação de referência, a mesma que roda a infraestrutura real do Bluesky. Mantém as duas bases (`comum` e `orbita`) na mesma linguagem.
   - **Risco aceito e documentado, não escondido:** o próprio Indigo se declara em desenvolvimento ativo — "features and software interfaces have not stabilized and may break or be removed". Ou seja: esperar alguma quebra de API ao atualizar dependências, e pinar versão explicitamente desde o primeiro `go.mod`. Mesmo espírito de risco já aceito conscientemente em `comum` (rate-gate do MusicBrainz, cache fail-open) — nomeado aqui, não descoberto depois.

3. **Identidades de teste: híbrido.** Dois ambientes, propósitos diferentes:
   - **PDS de desenvolvimento local** (via `indigo/cmd/pds` rodando localmente) para o loop do dia a dia — rápido, descartável, sem rate limit, sem sujar a rede real com registro de teste. Mesmo papel que Postgres/Redis local já cumprem em `comum`.
   - **Conta(s) reais da Bluesky** para validação periódica de interoperabilidade de verdade — confirmar que um registro `social.orbita.shelf.item` escrito por este código sobrevive num PDS de produção real, não só no ambiente controlado. Tecnicamente possível sem fricção: o protocolo não exige aprovação da Bluesky pra escrever um NSID customizado no repositório de alguém — é exatamente esse o ponto do AT Protocol.
   - Critério prático: o Beta 0 só conta como validado (item 5) quando passar nos dois ambientes, não só no local.

4. **Licença: AGPL-3.0.** Mesma escolha do Mastodon, e pelo mesmo motivo específico: a cláusula de uso em rede fecha a brecha que o GPL comum deixa aberto — sem ela, alguém poderia pegar o código, modificar, e operar como serviço hospedado sem nunca precisar distribuir as modificações (usuário só interage pela rede, nunca recebe uma cópia do software). AGPL obriga a disponibilizar o código modificado a quem usa o serviço pela rede, não só a quem recebe uma cópia binária. É a proteção certa contra "alguém fecha isso e vende" sem impedir uso/estudo/fork legítimo.

5. **Critério de "Beta 0 concluído"** — confirmado: login via OAuth funcionando contra uma conta real, um registro `social.orbita.shelf.item` criado no PDS dessa conta, Tap sincronizando esse registro pra um banco local, e uma página simples listando o que foi sincronizado. Sem UI além disso, sem segundo Lexicon, sem afinidade.

2. **Identificação da obra: string livre, sem validação, neste beta.** Confirmado: `workSlug` é só texto (`"matrix"`), digitado ou fixado no código do teste, sem checar existência, sem impedir que outra pessoa grave `"the-matrix"` pro mesmo filme. Aceito de propósito — ver nota abaixo sobre quando isso deixa de ser aceitável.

## Anotado para depois (não é problema do Beta 0)

Ponto real, levantado depois de fechar a decisão acima: obra é recurso compartilhado, não possuído por ninguém (o próprio roadmap de `comum` já nomeava isso antes de qualquer código de AT Protocol existir — seção "O catálogo de obras"). String livre resolve o Beta 0 porque só existe uma pessoa testando; quebra assim que duas pessoas divergem no texto pro mesmo filme, ou quando afinidade precisar comparar estantes de verdade.

A direção mais provável, quando esse beta chegar, **não é reinventar um catalog-service central** — é usar o padrão idiomático do próprio AT Protocol pra referência entre registros: um **strongRef** (par URI+CID, o mesmo mecanismo que um "like" do Bluesky usa pra apontar pro post curtido) em vez de uma string solta. Na prática:

- Um record type novo, `social.orbita.work`, publicado não pelo usuário mas pela própria conta de serviço da Órbita (ou futuramente por qualquer AppView que queira publicar catálogo) — um registro por obra, com identidade estável.
- A resolução externa que `comum` já validou (TMDB/MusicBrainz/Open Library — [ADR-009](https://github.com/PydaVi/comum/blob/main/docs/decisions/ADR-009-resolucao-externa-catalogo.md)) é reaproveitável quase sem mudança: mesma lógica de busca/normalização, só que o resultado vira um record `social.orbita.work` publicado, em vez de uma linha na tabela `works`.
- `social.orbita.shelf.item.work` passa a ser um strongRef pra esse registro, não mais uma string — duas pessoas adicionando "Matrix" à estante apontam pro mesmo registro, não pra dois textos parecidos.

Não decidir isso agora — só não deixar essa observação se perder antes do beta em que ela importa.
