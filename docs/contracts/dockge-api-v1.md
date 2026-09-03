# Dockge Automation REST API v1

Base: `/api/v1/automation`

## Autenticação

Use `Authorization: Bearer <token>`.

O servidor **não armazena o segredo em texto claro**. Cada credencial mantém somente SHA-256, scopes, namespaces, expiração e metadados administrativos. O segredo completo é exibido uma única vez quando a credencial é criada ou rotacionada.

A gestão humana das credenciais é feita em **Settings → API Access** e exige sessão autenticada + senha atual. Criar, rotacionar e revogar não são operações expostas pela própria REST API máquina-a-máquina.

Scopes disponíveis:

- `server:read`
- `stacks:read`
- `stacks:write`
- `stacks:delete`
- `stacks:operate`
- `stacks:adopt`

Qualquer scope `stacks:*` exige pelo menos um prefixo/namespace de stack. Exemplo: uma credencial do PIGE360 pode receber somente `pige360-`; ela não poderá operar `mailcow-*` ou stacks externas.

## Endpoints

- `GET /health`
- `GET /info`
- `GET /stacks`
- `GET /stacks/{name}`
- `PUT /stacks/{name}`
- `DELETE /stacks/{name}` — não remove volumes por padrão
- `POST /stacks/{name}/actions/pull`
- `POST /stacks/{name}/actions/up`
- `POST /stacks/{name}/actions/down`
- `POST /stacks/{name}/actions/restart`
- `POST /stacks/{name}/actions/start`
- `POST /stacks/{name}/actions/stop`
- `GET /stacks/{name}/ps`
- `GET /stacks/{name}/logs?tail=200`

## Adoção e isolamento

Stacks preexistentes nunca são adotadas silenciosamente. Um `PUT` sobre stack externa exige `adopt=true` e scope `stacks:adopt`.

O endpoint não aceita shell arbitrário. Operações Docker Compose são uma allowlist fixa e cada chamada passa por autenticação, scope e validação de namespace.

## Auditoria

Operações da REST API e ciclo de vida das credenciais são registrados em JSONL no arquivo configurado por `DOCKGE_API_AUDIT_FILE` (padrão `/app/data/api-audit.jsonl` no Compose canônico). Segredos e Bearer tokens nunca devem ser escritos no audit log.

## Política comercial

Licenciamento, inadimplência, autorização de upgrade, janela de manutenção e obrigação de backup pertencem ao **Control Plane consumidor**. O Dockge recebe uma ação técnica já autorizada e não implementa regra comercial de produto.
