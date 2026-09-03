# Generic Infrastructure Agent ↔ Control Plane REST Contract v1

O contrato é intencionalmente independente de produto. PIGE360, Connect API, ERP, Scheduler ou qualquer outro Control Plane podem implementá-lo.

## Enrollment

`POST /api/v1/infrastructure/agents/enroll`

Entrada: identidade do host, hostname, SO, arquitetura, versão do Agent, inventário e bootstrap token de uso inicial.

Saída: `agent_id` + `access_token`. O token de bootstrap deve ser invalidado/rotacionado após o enrollment.

## Heartbeat

`POST /api/v1/infrastructure/agents/{agent_id}/heartbeat`

Envia inventário, versão do Agent e saúde da API local do Dockge.

## Desired state

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
      "payload": {}
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

## Resultado

`POST /api/v1/infrastructure/agents/{agent_id}/actions/{action_id}/result`

O Agent sempre reporta sucesso/erro para permitir auditoria e idempotência no Control Plane.

## Separação de responsabilidade

Políticas comerciais, adimplência, direito a upgrade, janela de manutenção e autorização humana pertencem exclusivamente ao Control Plane da plataforma. O Agent e o Dockge não conhecem CNPJ, contrato ou cobrança; apenas executam ações tipadas já autorizadas pelo Control Plane.

## Segurança

- HTTPS obrigatório fora de localhost.
- Token individual por Agent; mTLS recomendado como evolução.
- Nunca expor Docker socket ao Control Plane.
- Dockge REST deve preferencialmente escutar somente em loopback/rede privada.
- Cada Control Plane recebe apenas os prefixes/namespaces de deployments que lhe pertencem.
- O Agent não oferece shell remoto arbitrário.
