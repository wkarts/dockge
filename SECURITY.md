# Security Policy

## Relato de vulnerabilidades

Vulnerabilidades de segurança deste projeto devem ser reportadas exclusivamente ao repositório independente `wkarts/dockge` por GitHub Security Advisory:

https://github.com/wkarts/dockge/security/advisories/new

Não publique detalhes sensíveis em Pull Requests, Issues ou Discussions públicos antes da correção coordenada.

## Escopo

A política cobre:

- backend e frontend Dockge;
- autenticação humana, sessões e 2FA/TOTP;
- REST API de automação;
- credenciais API, scopes e namespaces;
- integração com Docker/Compose;
- Generic Infrastructure Agent localizado em `infrastructure-agent/` enquanto estiver neste repositório;
- workflows oficiais de build/release e imagens `ghcr.io/wkarts/dockge`.

## Autenticação humana

- 2FA usa TOTP compatível com aplicativos autenticadores comuns;
- o segredo TOTP é cifrado em repouso com AES-256-GCM usando chave derivada do segredo interno da instância;
- instalações históricas com segredo TOTP em plaintext são migradas automaticamente após autenticação válida;
- senha e mudanças de 2FA incrementam a revisão de autenticação e invalidam sessões anteriores;
- JWTs web têm validade limitada;
- códigos TOTP reutilizados no login são rejeitados;
- tentativas de login/2FA passam por rate limit;
- `disableAuth=true` é bloqueado por padrão e somente pode ser habilitado por política explícita de implantação (`DOCKGE_ALLOW_DISABLE_AUTH=true`).

### Recuperação local de 2FA

A perda do autenticador não cria endpoint remoto de bypass. O administrador com acesso ao host/container pode executar:

```bash
npm run reset-2fa
```

O comando exige confirmação interativa do usuário local, limpa a configuração TOTP e invalida as sessões existentes.

## Credenciais da REST API

- segredos novos usam aleatoriedade criptográfica e são exibidos uma única vez;
- somente SHA-256 é persistido para autenticação Bearer;
- criação, rotação e revogação são realizadas por sessão humana autenticada e exigem senha atual;
- scopes `stacks:*` exigem namespace/prefixo de stack;
- credenciais legadas permanecem administráveis e recebem UUID estável quando rotacionadas/revogadas;
- ciclo de vida das credenciais é auditado sem registrar o segredo;
- o arquivo de tokens é gravado de forma atômica e com permissões restritas quando suportado pelo sistema operacional.

## Princípios de segurança da automação

- a API não oferece shell remoto arbitrário;
- ações remotas são tipadas e auditáveis;
- tokens de automação devem ter mínimo privilégio, expiração quando aplicável e escopo por namespace/deployment;
- stacks externas não podem ser adotadas implicitamente;
- segredos não devem ser persistidos em repositório, imagem ou log;
- Control Planes consumidores são responsáveis por políticas comerciais e autorização de negócio; Dockge e Agent executam apenas ações técnicas autorizadas.

## Generic Infrastructure Agent

- conexão permanente é outbound para Control Planes por HTTPS;
- um host pode possuir múltiplos vínculos independentes;
- cada Control Plane recebe apenas os namespaces atribuídos ao seu vínculo;
- ações repetidas usam journal persistente por `controller + action_id` e não são executadas novamente;
- configuração e journal usam gravação/substituição atômica multiplataforma;
- o Agent distingue Dockge detectado de Dockge API-first compatível e nunca adota/substitui uma instalação preexistente automaticamente;
- Docker socket permanece local e não é exposto diretamente aos Control Planes.

## Versões suportadas

Correções de segurança são priorizadas na linha estável mais recente (`main`) e na linha de desenvolvimento (`develop`). Versões antigas podem exigir atualização antes de receber correção.

## Projeto independente

Não envie vulnerabilidades específicas desta continuação para repositórios de terceiros. `wkarts/dockge` possui governança e ciclo de release próprios.
