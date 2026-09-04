# Generic Infrastructure Agent ↔ Control Plane REST Contract v1

O contrato é intencionalmente independente de produto. PIGE360, Connect API, ERP, Scheduler ou qualquer outro Control Plane podem implementá-lo sem introduzir regras específicas no Agent.

## Modelo de vínculo

Um mesmo host pode estar vinculado simultaneamente a vários Control Planes. Cada vínculo possui:

- URL própria;
- `agent_id` próprio;
- credencial de acesso própria;
- credencial de enrollment bootstrap própria;
- credencial Dockge própria quando o vínculo pode executar deployments;
- prefixos/namespaces de deployments autorizados próprios.

A configuração rejeita reutilização do mesmo arquivo sensível por vínculos diferentes. `enrollment_file`, `credential_file`, `agent_identity_file` e `dockge_credential_file` formam namespaces locais isolados por Control Plane.

A indisponibilidade de um Control Plane não deve interromper heartbeat/reconciliação dos demais. Enrollment pendente é tentado novamente em ciclos posteriores sem reiniciar o Agent.

## Transporte

Control Planes devem usar HTTPS. HTTP é aceito somente quando `allow_insecure_http=true` estiver explicitamente configurado para laboratório.

O endpoint local do Dockge aceita apenas esquemas HTTP/HTTPS e, por padrão, deve estar em loopback (`127.0.0.1`, `localhost` ou `::1`). Acesso não-loopback exige opt-in explícito.

## Enrollment

`POST /api/v1/infrastructure/agents/enroll`

Entrada mínima:

```json
{
  "agent_version": "0.2.0",
  "inventory": {},
  "nonce": "random-nonce"
}
```

A requisição é autenticada pela credencial bootstrap configurada para aquele Control Plane.

Saída:

```json
{
  "agent_id": "agt_123",
  "access_token": "one-time-returned-access-credential",
  "poll_seconds": 30,
  "token_expires_at": null
}
```

Depois de uma troca bem-sucedida, o Agent persiste a credencial operacional separadamente do JSON de configuração e remove localmente a credencial bootstrap. O Control Plane também deve considerar a credencial bootstrap consumida/revogada.

## Heartbeat

`POST /api/v1/infrastructure/agents/{agent_id}/heartbeat`

Envia inventário, versão do Agent, timestamp e saúde/detecção do Docker, Compose e da API local do Dockge.

O heartbeat é informativo. Ele não deve, por si só, executar mudança de infraestrutura.

## Desired state / fila de ações tipadas

`GET /api/v1/infrastructure/agents/{agent_id}/desired-state`

Exemplo:

```json
{
  "revision": "42",
  "actions": [
    {
      "id": "act-001",
      "deployment": "pige360-colegio-navegantes",
      "type": "dockge.stack.pull",
      "payload": {},
      "expires_at": "2026-09-03T23:00:00Z"
    },
    {
      "id": "act-002",
      "deployment": "pige360-colegio-navegantes",
      "type": "dockge.stack.up",
      "payload": {}
    }
  ]
}
```

### Regra de idempotência

`action.id` é a identidade imutável da execução.

Para um mesmo vínculo de Control Plane:

1. cada intenção de execução recebe um `action.id` único e não vazio;
2. se o Control Plane reenviar o **mesmo `action.id`**, o Agent **não executa a operação novamente** quando já existe resultado no journal;
3. o Agent lê o resultado persistido no journal local e apenas o reporta novamente;
4. se o Control Plane deseja uma nova tentativa real da operação, deve emitir **novo `action.id`**;
5. ações expiradas, negadas e que falharam também são resultados finais daquele `action.id` e são journaladas;
6. IDs iguais emitidos por Control Planes diferentes não colidem, pois o journal usa `controller + action_id` como chave.

A partir do Agent 0.2, existe uma segunda camada: toda mutação enviada ao Dockge carrega `Idempotency-Key: <action.id>`. O Dockge reserva a chave persistentemente antes de executar a alteração e guarda a conclusão. Isso cobre a janela em que Docker/Compose concluiu a operação, mas o processo do Agent caiu antes de conseguir gravar o journal local.

Se o Dockge encontrar uma reserva sem resultado final após uma queda, ele responde `409 idempotency_result_in_doubt` e não repete automaticamente a operação. O Agent reportará a falha ao Control Plane; o Control Plane deve reconciliar o estado atual e emitir um **novo** `action.id` apenas quando decidir por uma nova execução.

O journal é persistido no `data_dir` do Agent e sobrevive a restart/upgrade. Em conjunto com o store de idempotência do Dockge, uma falha de rede/processo não deve transformar um `pull`, `up`, `restart` ou `delete` em duplicação cega.

## Resultado

`POST /api/v1/infrastructure/agents/{agent_id}/actions/{action_id}/result`

Exemplo:

```json
{
  "action_id": "act-001",
  "status": "succeeded",
  "started_at": "2026-09-03T20:00:01Z",
  "finished_at": "2026-09-03T20:00:09Z",
  "message": ""
}
```

Estados previstos atualmente:

- `succeeded`;
- `failed`;
- `denied`;
- `expired`.

O Control Plane deve tornar o processamento do resultado igualmente idempotente.

## Ações suportadas pelo contrato atual

- `dockge.stack.apply`
- `dockge.stack.delete`
- `dockge.stack.pull`
- `dockge.stack.up`
- `dockge.stack.down`
- `dockge.stack.restart`
- `dockge.stack.start`
- `dockge.stack.stop`
- `noop`

Não existe ação genérica `shell`, `exec arbitrary` ou equivalente.

## Separação de responsabilidade

Políticas comerciais, adimplência, direito a upgrade, versão autorizada, janela de manutenção, aprovação humana e obrigação de backup pertencem exclusivamente ao **Control Plane da plataforma**.

O Agent e o Dockge não conhecem CNPJ, contrato, cobrança ou plano comercial. Eles executam somente ações técnicas tipadas que chegaram de um Control Plane autenticado e cujo deployment pertence ao namespace autorizado daquele vínculo.

Uma atualização de instalação de cliente deve ser modelada pelo Control Plane, por exemplo:

```text
solicitação de upgrade
        ↓
validar entitlement/adimplência
        ↓
obter autorização humana quando exigida
        ↓
executar/validar backup segundo estratégia do produto
        ↓
emitir ações técnicas com IDs únicos
        ↓
Agent → Dockge → Docker/Compose
```

## Segurança

- HTTPS obrigatório para Control Planes, salvo opt-in explícito de laboratório.
- Credencial individual por Agent/Control Plane; rotação e revogação devem ser suportadas pelo Control Plane.
- Arquivos sensíveis não podem ser reutilizados entre vínculos de Control Plane.
- mTLS por instalação é evolução compatível com este contrato.
- Nunca expor o Docker socket ao Control Plane.
- Dockge REST deve preferencialmente escutar somente em loopback/rede privada.
- Cada Control Plane recebe apenas os prefixos/namespaces de deployments que lhe pertencem.
- O Agent não oferece shell remoto arbitrário.
- Credenciais ficam em arquivos separados, com permissões restritas, e não dentro de `agent.json`.
- O journal não deve armazenar bearer tokens, enrollment tokens ou segredos de aplicação.
- O store de idempotência Dockge persiste somente hashes de principal/chave e fingerprint da requisição; não persiste o `action.id` em texto claro.
