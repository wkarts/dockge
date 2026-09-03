# Arquitetura

Control Plane (produto) -> REST/HTTPS desired state -> Agent -> REST loopback -> Dockge -> Docker/Compose.

Ações comerciais (adimplência, entitlement, autorização de atualização) existem somente no Control Plane. Agent e Dockge são deliberadamente neutros.

Cada Controller possui `allowed_deployment_prefixes`; o Agent recusa qualquer ação fora desses namespaces. Não existe endpoint genérico de execução de comandos.

## Bootstrap e operação permanente

O bootstrap pode ser realizado por pacote local (`.deb`, `.rpm`, Setup `.exe`, `.pkg`) ou por uma ferramenta administrativa que use SSH apenas na implantação inicial. Depois do enrollment, a operação normal é outbound-only por REST/HTTPS.

## Coexistência

O Agent não assume que a máquina está vazia. Docker, Compose, Dockge, Mailcow e stacks de terceiros podem preexistir. A API do Dockge só adota uma stack externa mediante ação explícita e credencial com escopo `stacks:adopt`.
