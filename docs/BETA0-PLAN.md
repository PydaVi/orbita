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

## Perguntas em aberto

Estas são as decisões que ainda precisam de conversa antes do Beta 0 virar código:

1. **Stack** — seguir o stack de referência do tutorial (Next.js/TypeScript, caminho oficial mais documentado) ou manter Go (consistente com `comum` e o princípio "stdlib first, dependência quando o problema já foi sentido")? Ir de Go aqui significa portar padrões do ecossistema JS na mão; ir de TS significa uma segunda linguagem entre os dois repositórios.

2. **O que é "obra" sem catalog-service** — o Lexicon referencia uma obra por slug, mas não há resolução de catálogo neste beta. Assumir slug como string livre (sem validação) parece o suficiente pro escopo mínimo — confirmar se isso é aceitável ou se já vale importar alguma checagem.

3. **De quem são os DIDs de teste** — usar contas reais da Bluesky (a própria, e talvez 1-2 pessoas que topem testar) ou subir um PDS de desenvolvimento local (mais perto do ambiente do tutorial, mas foge do caminho "usar a rede que já existe" que foi a decisão consciente de não hospedar PDS próprio agora).

4. **Licença** — AGPL-3.0 (mesma escolha do Mastodon; existe justamente pra impedir alguém pegar o código, fechar, e operar um serviço proprietário em cima) vs MIT (mais permissivo, mais fácil pra atrair contribuição casual, mas não impede um fork fechado) vs adiar a decisão. Nenhuma opção aqui é neutra — vale decidir com calma, não por padrão.

5. **Critério de "Beta 0 concluído"** — proposta mínima pra validar ou revisar: login via OAuth funcionando contra uma conta real, um registro `social.orbita.shelf.item` criado no PDS dessa conta, Tap sincronizando esse registro pra um banco local, e uma página simples listando o que foi sincronizado. Sem UI além disso, sem segundo Lexicon, sem afinidade.

## Fora de escopo neste beta (de propósito)

- PDS próprio — usamos o PDS que a conta já tem
- Mais de um tipo de registro (nota, afinidade, constelação ficam pra depois)
- Qualquer UI além do mínimo necessário pra provar que o dado sincronizou
- Federação entre múltiplas AppViews — aqui só existe a nossa, lendo uma rede que já existe
