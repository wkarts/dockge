# Security Policy

## Relato de vulnerabilidades

Vulnerabilidades de segurança deste projeto devem ser reportadas exclusivamente ao repositório independente `wkarts/dockge` por GitHub Security Advisory:

https://github.com/wkarts/dockge/security/advisories/new

Não publique detalhes sensíveis em Pull Requests, Issues ou Discussions públicos antes da correção coordenada.

## Escopo

A política cobre:

- backend e frontend Dockge;
- REST API de automação;
- autenticação, tokens e scopes;
- integração com Docker/Compose;
- Generic Infrastructure Agent localizado em `infrastructure-agent/` enquanto estiver neste repositório;
- workflows oficiais de build/release e imagens `ghcr.io/wkarts/dockge`.

## Princípios de segurança da automação

- a API não deve oferecer shell remoto arbitrário;
- ações remotas devem ser tipadas e auditáveis;
- tokens de automação devem ter mínimo privilégio, expiração quando aplicável e escopo por namespace/deployment;
- stacks externas não podem ser adotadas implicitamente;
- segredos não devem ser persistidos em repositório, imagem ou log;
- Control Planes consumidores são responsáveis por políticas comerciais e autorização de negócio; Dockge e Agent executam apenas ações técnicas autorizadas.

## Versões suportadas

Correções de segurança são priorizadas na linha estável mais recente (`main`) e na linha de desenvolvimento (`develop`). Versões antigas podem exigir atualização antes de receber correção.

## Projeto independente

Não envie vulnerabilidades específicas desta continuação para repositórios de terceiros. `wkarts/dockge` possui governança e ciclo de release próprios.
