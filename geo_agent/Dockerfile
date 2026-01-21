# Dockerfile for geo_agent

FROM python:3.12-slim

WORKDIR /app

# Install uv
COPY --from=ghcr.io/astral-sh/uv:latest /uv /usr/local/bin/uv

# Copy project files
COPY pyproject.toml .
COPY uv.lock* .
COPY .python-version* .

# Install dependencies
RUN uv sync --frozen --no-dev

# Copy application code
COPY . .

# Create logs directory
RUN mkdir -p logs

# Expose port
EXPOSE 8100

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
  CMD uv run python -c "import requests; requests.get('http://localhost:8100/health')"

# Run application
CMD ["uv", "run", "python", "main.py"]
