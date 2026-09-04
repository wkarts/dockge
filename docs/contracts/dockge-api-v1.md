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

## Semântica de atualização de `.env`

Em `PUT /stacks/{name}`:

- `compose_env` **omitido** numa stack existente preserva o `.env` atual;
- `compose_env: ""` é uma solicitação explícita para limpar o arquivo;
- numa stack nova, a ausência de `compose_env` cria `.env` vazio;
- `.env` e o marcador `.dockge-managed.json` são substituídos atomicamente e mantidos com permissão privada quando suportada pelo sistema operacional.

Isso evita que uma atualização somente do `compose.yaml` destrua credenciais ou parâmetros já provisionados.

## Idempotência de mutações

`PUT`, `DELETE` e `POST` de operação aceitam o header:

```http
Idempotency-Key: act-001
```

O Generic Infrastructure Agent 0.2+ envia o próprio `action.id` como chave. Clientes externos podem usar a mesma proteção.

Regras:

1. a chave é isolada por credencial/principal da API;
2. a primeira chamada reserva a execução em armazenamento persistente antes da mutação;
3. uma execução concluída é retornada como replay sem repetir Docker/Compose; a resposta inclui `X-Idempotency-Replayed: true`;
4. reutilizar a mesma chave com método, rota ou payload diferente retorna `409 idempotency_key_reused_with_different_request`;
5. se o processo cair depois da reserva e antes de persistir o resultado, uma nova chamada com a mesma chave retorna `409 idempotency_result_in_doubt` e **não** executa automaticamente a mutação novamente;
6. nesse estado indeterminado, o Control Plane deve reconciliar o estado atual e emitir um novo `action.id` somente se uma nova execução for realmente necessária.

O store é configurado por `DOCKGE_API_IDEMPOTENCY_FILE` e deve permanecer no volume persistente de dados. O formato não grava em texto claro o `Idempotency-Key` nem o identificador do principal; usa hashes para a chave de armazenamento e para o fingerprint da requisição.

## Auditoria

Operações da REST API e ciclo de vida das credenciais são registrados em JSONL no arquivo configurado por `DOCKGE_API_AUDIT_FILE` (padrão `/app/data/api-audit.jsonl` no Compose canônico). Segredos e Bearer tokens nunca devem ser escritos no audit log.

Arquivos persistentes recomendados no Compose API-first:

- `DOCKGE_API_TOKENS_FILE=/app/data/api-tokens.json`
- `DOCKGE_API_AUDIT_FILE=/app/data/api-audit.jsonl`
- `DOCKGE_API_IDEMPOTENCY_FILE=/app/data/api-idempotency.json`

## Política comercial

Licenciamento, inadimplência, autorização de upgrade, janela de manutenção e obrigação de backup pertencem ao **Control Plane consumidor**. O Dockge recebe uma ação técnica já autorizada e não implementa regra comercial de produto.
