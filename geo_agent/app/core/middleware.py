"""
FastAPI middleware
"""
import time
from typing import Callable
from fastapi import Request, Response
from fastapi.responses import JSONResponse
from starlette.middleware.base import BaseHTTPMiddleware
from starlette.middleware.cors import CORSMiddleware

from app.core.logger import logger_service, get_logger
from app.utils.helpers import generate_request_id

logger = get_logger(__name__)

class LoggingMiddleware(BaseHTTPMiddleware):
    """
    Middleware to log all HTTP requests
    """
    
    async def dispatch(self, request: Request, call_next: Callable) -> Response:
        # Generate request ID
        request_id = generate_request_id()
        request.state.request_id = request_id
        
        # Get client IP
        client_ip = request.client.host if request.client else "unknown"
        
        # Start timer
        start_time = time.time()
        
        # Log request
        logger.info(
            "request_started",
            request_id=request_id,
            method=request.method,
            path=request.url.path,
            client_ip=client_ip
        )
        
        # Print detailed HTTP request info for API endpoints
        if "/v1/" in request.url.path:
            print(f"\n{'🔵'*40}")
            print(f"🌐 HTTP请求 - {request.method} {request.url.path}")
            print(f"{'🔵'*40}")
            print(f"Request ID: {request_id}")
            print(f"客户端IP: {client_ip}")
            print(f"完整URL: {request.url}")
            if request.headers:
                print(f"请求头:")
                for key, value in request.headers.items():
                    if key.lower() not in ['authorization', 'cookie']:  # Don't log sensitive headers
                        print(f"  {key}: {value}")
            print(f"{'🔵'*40}\n")
        
        # Process request
        try:
            response = await call_next(request)
            
            # Calculate latency
            latency_ms = (time.time() - start_time) * 1000
            
            # Log access
            logger_service.log_access(
                method=request.method,
                path=request.url.path,
                status_code=response.status_code,
                latency_ms=latency_ms,
                client_ip=client_ip,
                request_id=request_id
            )
            
            # Print HTTP response info for API endpoints
            if "/v1/" in request.url.path:
                print(f"\n{'🟢'*40}")
                print(f"✅ HTTP响应 - {response.status_code}")
                print(f"{'🟢'*40}")
                print(f"Request ID: {request_id}")
                print(f"状态码: {response.status_code}")
                print(f"耗时: {latency_ms:.2f}ms")
                print(f"{'🟢'*40}\n")
            
            # Add request ID to response headers
            response.headers["X-Request-ID"] = request_id
            
            return response
            
        except Exception as e:
            latency_ms = (time.time() - start_time) * 1000
            
            logger.error(
                "request_failed",
                request_id=request_id,
                error=str(e),
                latency_ms=latency_ms
            )
            
            # Log error
            logger_service.log_error(
                error_type=type(e).__name__,
                error_message=str(e),
                request_id=request_id,
                method=request.method,
                path=request.url.path,
                client_ip=client_ip
            )
            
            # Log access with error status
            logger_service.log_access(
                method=request.method,
                path=request.url.path,
                status_code=500,
                latency_ms=latency_ms,
                client_ip=client_ip,
                request_id=request_id,
                error=str(e)
            )
            
            # Return error response
            return JSONResponse(
                status_code=500,
                content={
                    "error": {
                        "message": "Internal server error",
                        "type": "internal_error",
                        "request_id": request_id
                    }
                },
                headers={"X-Request-ID": request_id}
            )

def setup_cors(app):
    """
    Setup CORS middleware
    """
    app.add_middleware(
        CORSMiddleware,
        allow_origins=["*"],  # Configure based on your needs
        allow_credentials=True,
        allow_methods=["*"],
        allow_headers=["*"],
    )
