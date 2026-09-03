# Generic Infrastructure Control Architecture

```text
Control Plane PIGE360 ─┐
Control Plane ERP ─────┼── HTTPS/REST ──> Generic Infrastructure Agent
Control Plane Connect ─┘                        │
                                               │ localhost REST
                                               ▼
                                        Dockge API-first
                                               │
                                               ▼
                                      Docker Engine / Compose
```

## Separação de responsabilidades

### Generic Infrastructure Agent

- identidade persistente da máquina;
- enrollment em um ou mais Control Planes;
- inventário de host/Docker/Compose;
- heartbeat;
- polling de desired-state;
- execução via API local do Dockge;
- sem conhecimento de regras de negócio de qualquer plataforma.

### Dockge API-first

- executor Docker/Compose local;
- API REST autenticada;
- tokens com scopes;
- namespace/prefixo de stacks por Control Plane;
- coexistência com stacks externas;
- adoção explícita de stack legada;
- não exige exposição do Docker socket na rede.

### Control Plane

- conhece cliente/CNPJ/licenciamento;
- decide qual stack, versão, domínio e configuração devem existir;
- guarda estado desejado e auditoria;
- não precisa SSH após bootstrap.

## Coexistência com Dockge/provedor anterior

A API nova só opera stacks dentro dos prefixos concedidos ao token e só altera uma stack preexistente após adoção explícita com scope `stacks:adopt`. Assim WordPress, Mailcow e stacks de terceiros permanecem intactos.
