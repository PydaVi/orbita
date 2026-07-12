# Órbita

> Toda rede social tem um centro. Na maioria, esse centro é você.
> Aqui, o centro é a obra: o filme, a série, o disco, o livro que você ama.

Este repositório é o começo de uma AppView da Órbita sobre o **AT Protocol** — o protocolo aberto por trás do Bluesky. Identidade portável via DID, dados no PDS que a própria pessoa já controla, tipos de registro definidos em Lexicon. Sem servidor dono do seu gosto cultural.

## De onde isso vem

A Órbita nasceu em [`comum`](https://github.com/PydaVi/comum) como laboratório de sistemas distribuídos — um produto real, construído beta a beta, sentindo cada problema (estado persistente, cache, afinidade O(n²), observabilidade, escala horizontal) antes de usar a ferramenta que o resolve. Esse repositório continua vivo lá, como referência de estudo.

Este aqui é a continuação natural: migrar a mesma ideia de produto para uma arquitetura onde ninguém — nem a própria Órbita — é dona dos dados de quem usa. `orbita` nasce já pensado para além do autor: uma AppView pública, construída em aberto, dentro da comunidade do AT Protocol.

## O que diferencia a Órbita

- **O nó central é a obra, não a pessoa.** Capa, título e tipo da obra vêm antes de qualquer nome de usuário.
- **Sem engajamento algorítmico.** Feed cronológico. Sem "em alta", sem ranking por curtida.
- **Sem métrica pública de popularidade.** Contagem de seguidores existe só no seu próprio perfil, nunca como dado de status no perfil alheio.
- **Afinidade não é número, é forma.** A estante de cada pessoa desenha uma constelação; afinidade acontece quando duas constelações se parecem, sem exibir um placar de compatibilidade.
- **Não é ambiente para criador de conteúdo.** É espaço para comunidade se encontrar pelo que ama de verdade.

Os princípios completos — o porquê de cada decisão de produto — vivem no [`CLAUDE.md` de `comum`](https://github.com/PydaVi/comum/blob/main/CLAUDE.md). Este repositório parte deles; uma versão própria, adaptada ao contexto de AppView federada, é um dos primeiros itens em aberto do Beta 0.

## Estado atual

**Beta 0 — em andamento.** Decisões da primeira rodada fechadas (stack Go, licença, identidades de teste híbridas, critério de conclusão) — ver [`docs/BETA0-PLAN.md`](docs/BETA0-PLAN.md), que continua vivo e é atualizado a cada passo real, não só na hora do planejamento.

O que já existe:
- [`lexicons/social/orbita/shelf/item.json`](lexicons/social/orbita/shelf/item.json) — o primeiro Lexicon, schema do gesto de adicionar uma obra à estante
- [`cmd/appview/main.go`](cmd/appview/main.go) — esqueleto do servidor Go, só um `/health` por enquanto

O que falta pro Beta 0 ser considerado concluído (ver critério em `docs/BETA0-PLAN.md`): OAuth de verdade contra uma conta real, escrita do registro no PDS, sincronização via Tap, e uma página simples listando o que sincronizou. Nada disso está implementado ainda.

Isso é um hobby virando ideia, documentado em público. Progresso e decisões saem também no perfil [@orbita.bsky.social](https://bsky.app/profile/orbita.bsky.social) *(em breve)*.

## Por que AT Protocol

Se o servidor da Órbita fechasse hoje, a estante cultural de alguém sumiria junto. O AT Protocol resolve exatamente isso:

- **DID** — identidade portável, independente de qualquer servidor específico
- **PDS** — os dados moram num repositório que a própria pessoa controla (o mesmo que já usa no Bluesky, ou um auto-hospedado)
- **Lexicon** — o formato dos registros (`social.orbita.shelf.item`, `social.orbita.note`, …) é um contrato público, não um detalhe interno de banco de dados
- **AppView** — a Órbita passa a ser uma lente sobre dados que vivem espalhados pela rede, não a dona deles

## Licença

[AGPL-3.0](LICENSE). Mesma escolha do Mastodon, pelo mesmo motivo: a cláusula de uso em rede fecha a brecha que o GPL comum deixa aberta — sem ela, alguém poderia pegar o código, modificar, e operar como serviço hospedado sem nunca precisar devolver as modificações à comunidade, já que quem usa só interage pela rede, nunca recebe uma cópia do software. Aberto pra estudar, usar e contribuir; protegido contra virar produto fechado de terceiro.

## Contribuindo

Ainda não há processo formal — este é literalmente o estágio de desenhar o primeiro passo. Se a ideia ressoa com você, abra uma issue com pergunta, crítica ou interesse em ajudar. O objetivo do Beta 0 é justamente descobrir, em público, se e como isso vira trabalho de mais gente além de uma pessoa só.
