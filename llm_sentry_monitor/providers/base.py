import logging
from abc import ABC, abstractmethod
from typing import List, Dict, Any

# 配置日志
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s'
)

class BaseProvider(ABC):
    def __init__(self, headless: bool = False, timeout: int = 30000):
        self.headless = headless
        self.timeout = timeout
        self.logger = logging.getLogger(self.__class__.__name__)

    @abstractmethod
    def search(self, keyword: str, prompt: str) -> Dict[str, Any]:
        """
        执行搜索并返回结果
        返回格式: {
            "full_text": str,
            "citations": List[Dict[str, str]] # [{"url": "...", "title": "..."}]
        }
        """
        pass
