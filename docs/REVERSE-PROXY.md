# Reverse Proxy

O Dockge usa HTTP e Socket.IO/WebSocket no mesmo serviço. O reverse proxy precisa preservar `Host`, encaminhar os cabeçalhos de upgrade e usar timeouts adequados para conexões persistentes.

## Recomendação de exposição

O Compose canônico publica o Dockge em loopback por padrão:

```text
127.0.0.1:5001
```

Isso permite colocar Nginx, Traefik, Caddy, HAProxy ou CloudPanel na frente sem expor a porta administrativa diretamente à Internet.

A REST API de automação também passa pelo mesmo serviço, em:

```text
/api/v1/automation
```

Para Agent local no mesmo host, prefira sempre `http://127.0.0.1:5001` e não exponha essa API externamente sem necessidade.

## Nginx

```nginx
server {
    listen 443 ssl http2;
    server_name dockge.example.com;

    location / {
        proxy_pass http://127.0.0.1:5001;
        proxy_http_version 1.1;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_read_timeout 3600s;
        proxy_send_timeout 3600s;
    }
}
```

## Traefik

Exemplo conceitual de labels:

```yaml
labels:
  - traefik.enable=true
  - traefik.http.routers.dockge.rule=Host(`dockge.example.com`)
  - traefik.http.routers.dockge.entrypoints=websecure
  - traefik.http.routers.dockge.tls=true
  - traefik.http.services.dockge.loadbalancer.server.port=5001
```

## Caddy

```caddyfile
dockge.example.com {
    reverse_proxy 127.0.0.1:5001
}
```

## Trust proxy

Ative a opção de trust proxy no Dockge somente quando ele realmente estiver atrás de um proxy conhecido. Não confie em `X-Forwarded-For` vindo diretamente da Internet.

## Segurança

- TLS deve terminar em proxy confiável ou no próprio Dockge.
- Não publique `/var/run/docker.sock` fora do host.
- Não exponha a API de automação com tokens amplos; use scopes e prefixos mínimos.
- Prefira Agent → Dockge em loopback e Agent → Control Plane via HTTPS outbound.
- Se a UI for restrita à rede/VPN, mantenha a porta 5001 sem bind público.
