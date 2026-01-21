"""
Structured logging system with multiple outputs
"""
import os
import sys
import json
import logging
from pathlib import Path
from datetime import datetime
from typing import Any, Dict, Optional
import structlog
from structlog.types import EventDict, Processor

# Create logs directory
LOGS_DIR = Path(__file__).parent.parent.parent / "logs"
LOGS_DIR.mkdir(exist_ok=True)

# Log files
ACCESS_LOG = LOGS_DIR / "access.log"
ERROR_LOG = LOGS_DIR / "error.log"
QWEN_CALLS_LOG = LOGS_DIR / "qwen_calls.log"

def add_timestamp(logger: Any, method_name: str, event_dict: EventDict) -> EventDict:
    """Add ISO timestamp to log"""
    event_dict["timestamp"] = datetime.utcnow().isoformat() + "Z"
    return event_dict

def add_log_level(logger: Any, method_name: str, event_dict: EventDict) -> EventDict:
    """Add log level"""
    if method_name == "warn":
        method_name = "warning"
    event_dict["level"] = method_name.upper()
    return event_dict

def setup_logging(log_level: str = "INFO"):
    """
    Setup structured logging with multiple outputs
    
    Outputs:
    - Console: Human-readable format (dev mode)
    - access.log: HTTP access logs
    - error.log: Error and warning logs
    - qwen_calls.log: Qwen API call logs
    """
    
    # Configure structlog
    processors: list[Processor] = [
        structlog.contextvars.merge_contextvars,
        structlog.processors.add_log_level,
        add_timestamp,
        structlog.processors.StackInfoRenderer(),
        structlog.processors.format_exc_info,
    ]
    
    # Add JSON renderer for file output
    structlog.configure(
        processors=processors + [
            structlog.processors.JSONRenderer()
        ],
        wrapper_class=structlog.make_filtering_bound_logger(logging.getLevelName(log_level)),
        context_class=dict,
        logger_factory=structlog.PrintLoggerFactory(),
        cache_logger_on_first_use=True,
    )
    
    # Setup Python logging for file handlers
    logging.basicConfig(
        format="%(message)s",
        level=logging.getLevelName(log_level),
        handlers=[
            logging.StreamHandler(sys.stdout)
        ]
    )

def get_logger(name: str = __name__):
    """Get a structured logger"""
    return structlog.get_logger(name)

class LoggerService:
    """
    Logger service for different log types
    """
    
    def __init__(self):
        self.logger = get_logger()
        self._setup_file_handlers()
    
    def _setup_file_handlers(self):
        """Setup file handlers for different log types"""
        self.access_handler = open(ACCESS_LOG, 'a', encoding='utf-8')
        self.error_handler = open(ERROR_LOG, 'a', encoding='utf-8')
        self.qwen_handler = open(QWEN_CALLS_LOG, 'a', encoding='utf-8')
    
    def _write_to_file(self, handler, data: Dict[str, Any]):
        """Write JSON log to file"""
        handler.write(json.dumps(data, ensure_ascii=False) + '\n')
        handler.flush()
    
    def log_access(
        self,
        method: str,
        path: str,
        status_code: int,
        latency_ms: float,
        client_ip: str,
        request_id: str,
        **kwargs
    ):
        """Log HTTP access"""
        log_data = {
            "timestamp": datetime.utcnow().isoformat() + "Z",
            "level": "INFO",
            "event": "http_access",
            "method": method,
            "path": path,
            "status_code": status_code,
            "latency_ms": round(latency_ms, 2),
            "client_ip": client_ip,
            "request_id": request_id,
            **kwargs
        }
        
        self._write_to_file(self.access_handler, log_data)
        
        # Also log to console
        self.logger.info(
            "http_access",
            method=method,
            path=path,
            status_code=status_code,
            latency_ms=round(latency_ms, 2)
        )
    
    def log_error(
        self,
        error_type: str,
        error_message: str,
        request_id: str,
        **kwargs
    ):
        """Log errors"""
        log_data = {
            "timestamp": datetime.utcnow().isoformat() + "Z",
            "level": "ERROR",
            "event": "error",
            "error_type": error_type,
            "error_message": error_message,
            "request_id": request_id,
            **kwargs
        }
        
        self._write_to_file(self.error_handler, log_data)
        
        # Also log to console
        self.logger.error(
            "error",
            error_type=error_type,
            error_message=error_message,
            request_id=request_id
        )
    
    def log_qwen_call(
        self,
        request_id: str,
        request: Dict[str, Any],
        response: Optional[Dict[str, Any]] = None,
        usage: Optional[Dict[str, int]] = None,
        latency_ms: Optional[float] = None,
        error: Optional[str] = None,
        client_ip: str = "",
        **kwargs
    ):
        """Log Qwen API calls with detailed request/response info"""
        log_data = {
            "timestamp": datetime.utcnow().isoformat() + "Z",
            "request_id": request_id,
            "level": "INFO" if not error else "ERROR",
            "event": "qwen_api_call",
            "request": request,
            "client_ip": client_ip,
            **kwargs
        }
        
        if response:
            log_data["response"] = response
        
        if usage:
            log_data["usage"] = usage
        
        if latency_ms is not None:
            log_data["latency_ms"] = round(latency_ms, 2)
        
        if error:
            log_data["error"] = error
        
        self._write_to_file(self.qwen_handler, log_data)
        
        # Detailed console logging - print full request and response
        console_msg = "qwen_api_call"
        console_ctx = {
            "request_id": request_id,
            "model": request.get("model"),
        }
        
        if usage:
            console_ctx["tokens"] = usage.get("total_tokens")
        
        if latency_ms:
            console_ctx["latency_ms"] = round(latency_ms, 2)
        
        if error:
            self.logger.error(console_msg, error=error, **console_ctx)
            # Print detailed error info
            print(f"\n{'='*80}")
            print(f"❌ API调用错误 - Request ID: {request_id}")
            print(f"{'='*80}")
            print(f"错误: {error}")
            print(f"{'='*80}\n")
        else:
            self.logger.info(console_msg, **console_ctx)
            # Print detailed request and response
            print(f"\n{'='*80}")
            print(f"📤 中转请求 - Request ID: {request_id}")
            print(f"{'='*80}")
            print(f"模型: {request.get('model', 'N/A')}")
            print(f"消息数: {len(request.get('messages', []))}")
            if request.get('temperature'):
                print(f"温度: {request.get('temperature')}")
            if request.get('max_tokens'):
                print(f"最大tokens: {request.get('max_tokens')}")
            print(f"\n消息内容:")
            for i, msg in enumerate(request.get('messages', []), 1):
                role = msg.get('role', 'unknown')
                content = msg.get('content', '')
                print(f"  [{i}] {role}: {content[:200]}{'...' if len(content) > 200 else ''}")
            
            if response:
                print(f"\n{'='*80}")
                print(f"📥 中转响应 - Request ID: {request_id}")
                print(f"{'='*80}")
                print(f"响应内容: {response.get('content', '')[:500]}{'...' if len(response.get('content', '')) > 500 else ''}")
                print(f"完成原因: {response.get('finish_reason', 'N/A')}")
                if usage:
                    print(f"\nToken使用:")
                    print(f"  输入tokens: {usage.get('prompt_tokens', usage.get('input_tokens', 0))}")
                    print(f"  输出tokens: {usage.get('completion_tokens', usage.get('output_tokens', 0))}")
                    print(f"  总tokens: {usage.get('total_tokens', 0)}")
                if latency_ms:
                    print(f"耗时: {latency_ms:.2f}ms")
            print(f"{'='*80}\n")
    
    def close(self):
        """Close all file handlers"""
        self.access_handler.close()
        self.error_handler.close()
        self.qwen_handler.close()

# Global logger service instance
logger_service = LoggerService()
