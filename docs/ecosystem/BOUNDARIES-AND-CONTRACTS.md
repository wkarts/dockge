# Fronteiras, contratos e segurança

## Automation REST API preservada

Base atual: `/api/v1/automation`.

Baseline atual do Dockge 1.6.0:

```text
GET    /health
GET    /info
GET    /stacks
GET    /stacks/{name}
PUT    /stacks/{name}
DELETE /stacks/{name}
POST   /stacks/{name}/actions/pull
POST   /stacks/{name}/actions/up
POST   /stacks/{name}/actions/down
POST   /stacks/{name}/actions/restart
POST   /stacks/{name}/actions/start
POST   /stacks/{name}/actions/stop
GET    /stacks/{name}/ps
GET    /stacks/{name}/logs?tail=200
```

A API existente preserva Bearer tokens, scopes, namespaces, adoção explícita, auditoria e `Idempotency-Key` persistente.

## Regra de evolução

Se Manager ou Deploy precisarem de novas funções, elas devem nascer como capabilities públicas da Automation API. Não acessar banco interno, classes TypeScript internas ou Socket.IO interno como contrato externo.

Exemplos de extensões candidatas futuras, ainda não declaradas como existentes:

- métricas agregadas de host/container;
- progresso estruturado de operações longas;
- capabilities detalhadas;
- health summary de stack;
- snapshot/export de configuração.

## Native Agents protegidos

Não modificar para atender Manager/Deploy:

```text
backend/agent-manager.ts
backend/agent-socket-handler.ts
backend/agent-socket-handlers/
backend/models/agent.ts
backend/socket-handlers/agent-proxy-socket-handler.ts
backend/socket-handlers/manage-agent-socket-handler.ts
common/agent-socket.ts
```

## Segurança SSH

- validar host key;
- exibir fingerprint no primeiro vínculo;
- bloquear mudança inesperada de host key;
- suportar senha, chave e ssh-agent;
- não persistir passphrase em configuração plana;
- futuro bastion/jump host será um transporte, não uma exceção espalhada.

## Segredos

No Manager, segredos ficam criptografados em repouso e não retornam integralmente à UI após criação. No Deploy local, preferir Windows Credential Manager, macOS Keychain e Linux Secret Service/keyring.

## Operações proibidas automaticamente

```text
docker system prune -a --volumes
docker compose down -v
rm -rf /opt/stacks
rm -rf data persistente do Dockge
```

Nunca expor `docker.sock` pela rede.
