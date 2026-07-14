# Órbita

> Toda rede social tem um centro. Na maioria, esse centro é você.
> Aqui, o centro é a obra: o filme, a série, o disco, o livro que você ama.

Este repositório é o começo de uma AppView da Órbita sobre o **AT Protocol** — o protocolo aberto por trás do Bluesky. Identidade portável via DID, dados no PDS que a própria pessoa já controla, tipos de registro definidos em Lexicon. Sem servidor dono do seu gosto cultural.

## De onde isso vem

A Órbita nasceu como laboratório de sistemas distribuídos, um produto real, construído para simular problemas de devops/SRE como resiliência de sistemas distribuídos, estado persistente, cache, observabilidade, escala horizontal etc.

No fim, construí pra esse lab um produto que me empolgou tanto que esse aqui é a continuação natural: migrar a mesma ideia de produto para uma arquitetura onde ninguém — nem a própria Órbita — é dona dos dados de quem usa. `orbita` nasce já pensado para além do autor: uma AppView pública, construída em aberto, dentro da comunidade do AT Protocol.

## O que diferencia a Órbita

- **O nó central é a obra, não a pessoa.** Capa, título e tipo da obra vêm antes de qualquer nome de usuário.
- **Sem engajamento algorítmico.** Feed cronológico. Sem "em alta", sem ranking por curtida.
- **Sem métrica pública de popularidade.** Contagem de seguidores existe só no seu próprio perfil, nunca como dado de status no perfil alheio.
- **Afinidade não é número, é forma.** A estante de cada pessoa desenha uma constelação; afinidade acontece quando duas constelações se parecem, sem exibir um placar de compatibilidade.
- **Não é ambiente para criador de conteúdo.** É espaço para comunidade se encontrar pelo que ama de verdade.

## Estado atual

**Beta 0 — em andamento.** Decisões da primeira rodada fechadas (stack Go, licença, identidades de teste híbridas, critério de conclusão) — ver [`docs/BETA0-PLAN.md`](docs/BETA0-PLAN.md), que continua vivo e é atualizado a cada passo real, não só na hora do planejamento.

O que já existe:
- [`lexicons/social/orbita/shelf/item.json`](lexicons/social/orbita/shelf/item.json) — o primeiro Lexicon, schema do gesto de adicionar uma obra à estante
- [`cmd/appview/main.go`](cmd/appview/main.go) + [`oauth.go`](cmd/appview/oauth.go) + [`shelf.go`](cmd/appview/shelf.go) — servidor Go com `/health`, `/webhook`, e **login OAuth real + escrita autenticada de verdade**
- **Primeiro dado real da Órbita no AT Protocol**: um `social.orbita.shelf.item` escrito via OAuth (PAR/PKCE/DPoP completos, sem atalho) na conta real do autor, confirmado no PDS de produção
- [`scripts/dev-pds/`](scripts/dev-pds/) — PDS + PLC locais e descartáveis, sem Postgres, sem TLS, pra estudar e testar sem depender de conta real
- Pipeline completo validado de ponta a ponta — PDS local → Tap → webhook, com backfill de registro pré-existente confirmado — arquitetura documentada em [`docs/architecture-beta0-local.md`](docs/architecture-beta0-local.md)

O que falta pro Beta 0 ser considerado concluído (ver critério em `docs/BETA0-PLAN.md`): OAuth de verdade contra uma conta real (todo esse ciclo hoje é feito via `curl`, não código Go), e indexar o que o webhook recebe num banco local.

Isso é um hobby virando ideia, documentado em público. Progresso e decisões saem também no perfil [@orbita.bsky.social](https://bsky.app/profile/orbita.bsky.social) *(em breve)*.

## Por que AT Protocol

Se o servidor da Órbita fechasse hoje, a estante cultural de alguém sumiria junto. O AT Protocol resolve exatamente isso:

- **DID** — identidade portável, independente de qualquer servidor específico
- **PDS** — os dados moram num repositório que a própria pessoa controla (o mesmo que já usa no Bluesky, ou um auto-hospedado)
- **Lexicon** — o formato dos registros (`social.orbita.shelf.item`, `social.orbita.note`, …) é um contrato público, não um detalhe interno de banco de dados
- **AppView** — a Órbita passa a ser uma lente sobre dados que vivem espalhados pela rede, não a dona deles

## Licença

[AGPL-3.0](LICENSE). Mesma escolha do Mastodon, pelo mesmo motivo: a cláusula de uso em rede fecha a brecha que o GPL comum deixa aberta — sem ela, alguém poderia pegar o código, modificar, e operar como serviço hospedado sem nunca precisar devolver as modificações à comunidade, já que quem usa só interage pela rede, nunca recebe uma cópia do software. Aberto pra estudar, usar e contribuir; protegido contra virar produto fechado de terceiro.

## Uso de IA no desenvolvimento

Este projeto é desenvolvido com uso ativo de assistentes de IA, como parceiro de pesquisa, implementação e documentação, sob minha direção e revisão em cada decisão. Nenhum princípio de produto, decisão de arquitetura ou linha de código entra aqui sem eu entender e validar o porquê primeiro; é esse o próprio motivo de manter tudo documentado (`docs/BETA0-PLAN.md`, os diagramas de arquitetura) tão de perto — inclusive coisas erradas que assumi e corrigi ao longo do caminho ficam registradas, não escondidas.

Divulgo isso abertamente porque transparência já é um princípio não-negociável da própria Órbita como produto, seria incoerente pedir isso da rede social e esconder isso do processo que a constrói.

## Contribuindo

Ainda não há processo formal, este é literalmente o estágio de desenhar o primeiro passo. Se a ideia ressoa com você, abra uma issue com pergunta, crítica ou interesse em ajudar. O objetivo do Beta 0 é justamente descobrir, em público, se e como isso vira trabalho de mais gente além de uma pessoa só.
