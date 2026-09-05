# Dockge Ecosystem — Implementation Status

## Estado da entrega 1.0

A arquitetura **Dockge Core / Dockge Manager / Dockge Deploy / Native Agents** está materializada com toolchains, versionamento, pipelines e responsabilidades separados. Os componentes permanecem no mesmo repositório por conveniência operacional, mas não entram no runtime/imagem do Dockge Core e podem ser separados fisicamente no futuro sem alterar seus contratos.

### Dockge Core `1.6.1`

- continua sendo o orquestrador Docker/Compose e a fonte de verdade operacional;
- Automation API `/api/v1/automation` preservada;
- tokens/scopes/namespaces, auditoria e `Idempotency-Key` persistente preservados;
- Native Agents preservados e protegidos por gate de CI;
- nenhum binário do antigo Generic Infrastructure Agent é publicado nas releases do Core;
- mudanças exclusivas de Manager/Deploy não entram no contexto da imagem nem devem gerar SemVer do Core.

### Dockge Manager `1.0.1`

Implementado em `dockge-manager/`:

- FastAPI;
- SQLAlchemy 2 + PostgreSQL + Alembic;
- Vue 3 + TypeScript + PWA responsiva;
- autenticação JWT local;
- `Workspace`, `Environment`, `CredentialRef`, `DockgeTarget`, `Application`, `Deployment`, `DeploymentRevision`, `DeploymentSnapshot`, `Operation`, `AuditEvent` e `HealthSnapshot`;
- tokens Dockge e `.env` criptografados em repouso com Fernet;
- múltiplos Dockge Targets;
- health polling;
- stacks, logs e operações start/stop/restart/pull/up/down;
- apply de Compose exclusivamente via Automation API;
- uma `Idempotency-Key` por intenção mutável;
- retries de transporte/5xx reutilizando exatamente a mesma chave, permitindo replay seguro de resposta perdida;
- operação `IN_DOUBT` quando o Core mantém a reserva fail-closed ou o resultado remoto permanece incerto;
- snapshots do runtime real antes de deployments;
- revisão desejada separada da revisão realmente ativa;
- verificação de containers/health depois do deploy;
- rollback automático quando a execução realizou **ou pode ter realizado** mutação e falha depois;
- reconciliação de snapshot antes de restaurar/apagar, evitando deletar stack externa em estado ambíguo;
- rollback manual para revisão previamente ativa;
- auditoria de sucesso, falha, retry e resultado incerto;
- PWA com infraestrutura, deployments e atividade;
- imagem multiarch independente `ghcr.io/wkarts/dockge-manager:1.0.1` após publicação da main;
- release independente `dockge-manager-v1.0.1` após publicação da main.

O Manager não monta `docker.sock`, não acessa o banco interno do Dockge, não usa Socket.IO interno e não escreve em `/opt/stacks`.

### Dockge Deploy `1.0.2`

Implementado em `dockge-deploy/`:

- CLI Go multiplataforma;
- SSH com verificação de host key, fingerprint SHA-256 e TOFU somente por aceite explícito;
- chave privada, `ssh-agent`, senha/passphrase somente por ambiente;
- inventário/doctor;
- bootstrap idempotente do Docker Engine + Compose em famílias Debian/Ubuntu, RPM e Alpine;
- detecção dos quatro nomes Compose padrão;
- instalação nova em modo PLAN/APPLY;
- upgrade com detecção do bind real de `/app/data`, snapshot da persistência real, preservação da imagem atual e rollback automático;
- análise de migração read-only com inventário dos mounts reais de dados/stacks;
- migração in-place de Dockge legado para `ghcr.io/wkarts/dockge`, preservando o bind de dados e o caminho real das stacks;
- fail-closed para layouts automáticos não suportados, como `/app/data` em named volume ou mismatch do mount de stacks;
- rollback manual de upgrade/migração;
- criação/rotação de credencial Automation API dedicada ao Manager;
- cliente direto da Automation API;
- stack list/inspect/logs/apply/start/stop/restart/pull/up/down;
- mutações API plan-first, `--apply` explícito e `Idempotency-Key`;
- redirects bloqueados e HTTP remoto bloqueado por padrão;
- builds Linux, Windows e macOS em amd64/arm64;
- binários crus, pacotes `.tar.gz`/`.zip` e `SHA256SUMS.txt`;
- release independente `dockge-deploy-v1.0.2` publicada.

## Invariantes verificáveis

Nenhum dos componentes novos pode:

- modificar ou substituir os Native Agents;
- montar/expor Docker socket no Manager;
- executar `docker compose down -v` ou `docker system prune -a --volumes` automaticamente;
- remover `/opt/stacks`;
- declarar sucesso de deployment sem consultar o estado real no Dockge;
- transformar uma falha definitiva anterior à mutação em adoção/remoção automática de stack;
- presumir que uma resposta HTTP perdida significa que a mutação não aconteceu;
- apagar automaticamente uma stack sem marcador API-managed ao reconciliar um snapshot que antes representava ausência.

## Próximas evoluções sem bloquear a linha 1.0

A linha 1.0 é operacional. Melhorias posteriores podem incluir notificações/alertas avançados, tags de inventário, mais adapters de distribuição Linux, bastion/jump-host, SSO, políticas de aprovação e reconciliação assíncrona de operações antigas `IN_DOUBT`, sem alterar a divisão arquitetural estabelecida.
