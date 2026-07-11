# Beta 0 — rascunho de planejamento

**Status:** em aberto. Isto é ponto de partida pra discussão, não um roadmap fechado — nada aqui deve ser lido como decidido só porque está escrito.

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

## Pergunta em aberto — reformulada

2. **O que vai dentro do campo que identifica a obra, já que não existe catalog-service aqui?**

   Em `comum`, uma obra tem um slug estável, resolvido e validado contra TMDB/MusicBrainz/Open Library (Beta 4). Aqui, no Beta 0, não existe nenhum serviço de catálogo — então quando alguém adicionar "Matrix" à estante, o que vai no registro `social.orbita.shelf.item.workSlug`?

   A proposta mínima é: **uma string livre, digitada ou fixada no código do teste, sem validação nenhuma** — acontece de "matrix" ser só um texto qualquer neste beta, sem checar se existe, sem impedir que outra pessoa grave "the-matrix" pro mesmo filme. Aceitar essa inconsistência de propósito agora (ela só vira problema real quando houver mais de uma pessoa testando e afinidade entrar em cena, o que é beta seguinte) é mais simples do que importar qualquer noção de catálogo cedo demais.

   Pergunta: essa simplificação (string livre, zero validação, inconsistência aceita) serve pro Beta 0, ou você já quer algum tipo de lista fixa/travada de slugs válidos desde já?

## Fora de escopo neste beta (de propósito)

- PDS próprio — usamos o PDS que a conta já tem
- Mais de um tipo de registro (nota, afinidade, constelação ficam pra depois)
- Qualquer UI além do mínimo necessário pra provar que o dado sincronizou
- Federação entre múltiplas AppViews — aqui só existe a nossa, lendo uma rede que já existe
