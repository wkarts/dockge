# Migração administrativa do repositório

`wkarts/dockge` é um projeto independente (`fork: false`).

## Estado canônico desejado

- Default branch: `main`
- Integração/homologação: `develop`
- Desenvolvimento: Pull Requests para `develop`
- Produção: promoção por PR `develop -> main`
- Registry: `ghcr.io/wkarts/dockge`
- `master`: legado congelado e destinado à exclusão

## Configurações do painel GitHub

As seguintes propriedades pertencem a Repository Settings e não são alteradas por commits Git:

1. trocar Default branch de `master` para `main`;
2. remover homepage herdada `https://dockge.kuma.pet` ou substituí-la por um endereço oficial próprio quando existir;
3. atualizar a descrição para refletir a plataforma independente API-first;
4. habilitar Issues/Discussions apenas se fizerem parte da governança desejada;
5. opcionalmente habilitar `Automatically delete head branches` após merge;
6. após validar `main`, excluir `master`.

## Transição segura do master

Enquanto `master` ainda for a default branch:

- nenhum release parte dele;
- nightly herdado permanece removido;
- README o identifica como legado;
- um gate mínimo pode existir somente para inicializar validações das novas branches.

Após a primeira promoção validada para `main`, `master` pode ser fast-forwardado uma última vez para o snapshot de `main`, congelado e então excluído após a troca da default branch.

## Histórico e licença

A existência de commits anteriores no grafo Git não representa vínculo operacional. O histórico é preservado para rastreabilidade e para respeitar a licença MIT e avisos de copyright do código-base de origem.
