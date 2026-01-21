.PHONY: help install sync dev prod test clean logs docker-build docker-run check-uv check-env

# Check if uv is installed
check-uv:
	@command -v uv >/dev/null 2>&1 || { echo "Error: uv is not installed. Install it with: curl -LsSf https://astral.sh/uv/install.sh | sh"; exit 1; }

# Check virtual environment
check-env:
	@if [ -d ".venv" ]; then \
		echo "✅ Local virtual environment: .venv/"; \
		echo "Python: $$(uv run python --version 2>/dev/null || echo 'Not initialized')"; \
		echo "Location: $$(pwd)/.venv"; \
	else \
		echo "⚠️  Virtual environment not found. Run: make install"; \
	fi

help:
	@echo "geo_agent - OpenAI compatible agent service"
	@echo ""
	@echo "Available commands:"
	@echo "  make install       - Install uv and sync dependencies"
	@echo "  make sync          - Sync dependencies (uv sync)"
	@echo "  make check-env     - Check virtual environment status"
	@echo "  make dev           - Run development server (with hot reload)"
	@echo "  make prod          - Run production server"
	@echo "  make test          - Run tests"
	@echo "  make test-curl     - Test with curl"
	@echo "  make test-openai   - Test with OpenAI SDK"
	@echo "  make test-logging  - Test logging enhancements"
	@echo "  make logs          - Tail all logs"
	@echo "  make logs-qwen     - Tail qwen API call logs"
	@echo "  make logs-access   - Tail access logs"
	@echo "  make logs-error    - Tail error logs"
	@echo "  make stats         - Show token usage statistics"
	@echo "  make clean         - Clean logs and cache"
	@echo "  make docker-build  - Build Docker image"
	@echo "  make docker-run    - Run Docker container"
	@echo ""
	@echo "Using uv with local virtual environment (.venv/)"

install: check-uv
	@echo "Syncing dependencies with uv..."
	uv sync

sync: check-uv
	@echo "Syncing dependencies with uv..."
	uv sync

dev: check-uv
	@echo "Starting development server..."
	@echo "Server will run at http://localhost:8100"
	@echo "API docs at http://localhost:8100/docs"
	uv run python main.py

prod: check-uv
	@echo "Starting production server..."
	uv run uvicorn main:app --host 0.0.0.0 --port 8100 --workers 4

test: check-uv
	@echo "Running tests..."
	uv run pytest -v

test-curl:
	@echo "Testing with curl..."
	@curl -X POST http://localhost:8100/v1/chat/completions \
		-H "Content-Type: application/json" \
		-d '{"model":"qwen3-max","messages":[{"role":"user","content":"你好，请介绍一下你自己"}],"temperature":0.7}'

test-openai: check-uv
	@echo "Testing with OpenAI SDK..."
	@uv run python -c "from openai import OpenAI; \
		client = OpenAI(base_url='http://localhost:8100/v1', api_key='dummy'); \
		response = client.chat.completions.create(model='qwen3-max', messages=[{'role':'user','content':'你好'}]); \
		print(response.choices[0].message.content)"

test-logging: check-uv
	@echo "Testing logging enhancements..."
	@chmod +x test_logging.py
	@uv run python test_logging.py

logs:
	@echo "Tailing all logs..."
	tail -f logs/*.log

logs-qwen:
	@echo "Tailing qwen API call logs..."
	tail -f logs/qwen_calls.log | jq .

logs-access:
	@echo "Tailing access logs..."
	tail -f logs/access.log | jq .

logs-error:
	@echo "Tailing error logs..."
	tail -f logs/error.log | jq .

stats:
	@echo "Token usage statistics:"
	@echo "Total API calls:"
	@grep -c "qwen_api_call" logs/qwen_calls.log || echo "0"
	@echo "Total tokens used:"
	@grep "total_tokens" logs/qwen_calls.log | jq -r '.usage.total_tokens' | awk '{s+=$$1} END {print s}' || echo "0"
	@echo "Average latency (ms):"
	@grep "latency_ms" logs/qwen_calls.log | jq -r '.latency_ms' | awk '{s+=$$1; c++} END {if(c>0) print s/c; else print 0}'

clean:
	@echo "Cleaning logs and cache..."
	rm -f logs/*.log
	rm -rf __pycache__
	rm -rf app/__pycache__
	rm -rf app/**/__pycache__
	rm -rf .pytest_cache
	rm -rf .venv
	find . -type d -name "__pycache__" -exec rm -rf {} + 2>/dev/null || true

docker-build:
	@echo "Building Docker image..."
	docker build -t geo_agent:latest .

docker-run:
	@echo "Running Docker container..."
	docker run -d -p 8100:8100 --env-file .env --name geo_agent geo_agent:latest
