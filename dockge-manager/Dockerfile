FROM node:22-alpine AS web-build
WORKDIR /src/frontend
COPY frontend/package.json ./package.json
RUN npm install --no-audit --no-fund
COPY frontend/ ./
RUN npm run build

FROM python:3.13-slim AS runtime
ENV PYTHONDONTWRITEBYTECODE=1 \
    PYTHONUNBUFFERED=1 \
    DOCKGE_MANAGER_STATIC_DIR=/app/static
WORKDIR /app
RUN useradd --create-home --uid 10001 manager
COPY requirements.txt /app/requirements.txt
RUN pip install --no-cache-dir -r /app/requirements.txt
COPY backend /app/backend
COPY migrations /app/migrations
COPY alembic.ini VERSION /app/
COPY --from=web-build /src/frontend/dist /app/static
ENV PYTHONPATH=/app/backend
USER manager
EXPOSE 8000
CMD ["sh", "-c", "alembic upgrade head && uvicorn app.main:app --host 0.0.0.0 --port 8000"]
