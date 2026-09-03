# Dockge Automation REST API v1

Base: `/api/v1/automation`

Autenticação: `Authorization: Bearer <token>`. O servidor armazena apenas SHA-256 dos tokens. Tokens têm scopes, validade opcional e prefixos de stack.

Scopes: `server:read`, `stacks:read`, `stacks:write`, `stacks:delete`, `stacks:operate`, `stacks:adopt`.

Endpoints:

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

Stacks preexistentes nunca são adotadas silenciosamente. Um `PUT` sobre stack externa exige `adopt=true` e scope `stacks:adopt`.

O endpoint não aceita shell arbitrário. Operações são uma allowlist fixa de ações Docker Compose.
