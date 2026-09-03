# Translations

Guia para tradução do Dockge independente mantido em `wkarts/dockge`.

## Atualizar uma tradução existente

1. Crie uma branch a partir de `develop`.
2. Altere apenas os arquivos necessários em `frontend/src/lang/`.
3. Preserve as mesmas chaves existentes em `en.json`.
4. Execute as validações do projeto.
5. Abra Pull Request para `develop`.

## Adicionar um novo idioma

1. Crie o arquivo de idioma em `frontend/src/lang/` usando `en.json` como referência.
2. Adicione o código/descrição do idioma em `languageList` de `frontend/src/i18n.ts`.
3. Garanta que nenhuma chave obrigatória ficou ausente.
4. Abra PR para `develop`.

## Governança

Este projeto não depende do serviço de tradução ou repositório do projeto de origem. Ferramentas externas de localização poderão ser integradas futuramente apenas quando configuradas para `wkarts/dockge`.
