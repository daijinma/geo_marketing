"""
Encoding utilities - Handle text encoding issues
"""


def ensure_utf8_string(text, logger=None):
    """
    确保文本是 UTF-8 编码的字符串
    处理可能的编码问题，防止乱码
    特别处理 UTF-8 被当作 Latin-1 读取的情况（如：ä¸"ä¸š 应该是中文）
    
    Args:
        text: 要处理的文本
        logger: 可选的日志记录器，用于记录编码修复过程
    """
    if text is None:
        return ""
    
    original_text = text
    
    if isinstance(text, bytes):
        try:
            return text.decode('utf-8')
        except UnicodeDecodeError:
            # 尝试其他常见编码
            for encoding in ['utf-8', 'gbk', 'gb2312', 'latin-1']:
                try:
                    decoded = text.decode(encoding)
                    # 如果解码成功，尝试重新编码为 UTF-8 以确保一致性
                    result = decoded.encode('utf-8').decode('utf-8')
                    if logger and encoding != 'utf-8':
                        logger.debug(f"编码修复: 从 {encoding} 解码字节数据 (长度: {len(text)})")
                    return result
                except (UnicodeDecodeError, UnicodeEncodeError):
                    continue
            # 如果所有编码都失败，使用 replace 模式
            if logger:
                logger.warning(f"编码修复失败: 无法解码字节数据 (长度: {len(text)})，使用 replace 模式")
            return text.decode('utf-8', errors='replace')
    elif not isinstance(text, str):
        # 如果不是字符串也不是字节，转换为字符串
        return str(text)
    else:
        # 已经是字符串，需要检测并修复编码问题
        # 情况1: UTF-8 被当作 Latin-1 读取（常见乱码情况）
        # 检测特征：包含 Latin-1 范围内的字节值（128-255），但实际应该是 UTF-8 多字节字符
        if any(ord(c) > 127 for c in text):
            try:
                # 尝试将字符串重新编码为 Latin-1（无损），再解码为 UTF-8
                # 这可以修复 UTF-8 被当作 Latin-1 读取的情况
                fixed = text.encode('latin-1').decode('utf-8')
                # 验证修复后的字符串是否包含有效的中文字符
                # 如果修复成功，应该包含中文字符或至少不是乱码模式
                if fixed and len(fixed) > 0:
                    # 检查是否包含常见的中文字符范围
                    has_chinese = any('\u4e00' <= c <= '\u9fff' for c in fixed)
                    # 或者检查是否不再包含明显的乱码模式（连续的 Latin-1 高字节字符）
                    has_garbled_pattern = any(
                        ord(c) > 127 and ord(c) < 160 
                        for c in text[:min(100, len(text))]
                    )
                    if has_chinese or not has_garbled_pattern:
                        if logger:
                            logger.info(f"编码修复: 修复 UTF-8 被当作 Latin-1 读取的乱码")
                            logger.debug(f"  原始: {text[:100]}...")
                            logger.debug(f"  修复: {fixed[:100]}...")
                        return fixed
            except (UnicodeEncodeError, UnicodeDecodeError) as e:
                if logger:
                    logger.debug(f"编码修复尝试失败: {e}")
                pass
        
        # 情况2: 双重编码问题（UTF-8 被编码了两次）
        try:
            # 尝试检测双重编码：如果字符串可以编码为 Latin-1 再解码为 UTF-8，可能是双重编码
            double_encoded = text.encode('latin-1', errors='ignore').decode('utf-8', errors='ignore')
            if double_encoded and double_encoded != text:
                # 检查修复后的结果是否更合理
                if any('\u4e00' <= c <= '\u9fff' for c in double_encoded):
                    if logger:
                        logger.info(f"编码修复: 修复双重编码问题")
                        logger.debug(f"  原始: {text[:100]}...")
                        logger.debug(f"  修复: {double_encoded[:100]}...")
                    return double_encoded
        except Exception as e:
            if logger:
                logger.debug(f"双重编码检测失败: {e}")
            pass
        
        # 情况3: 正常的 UTF-8 字符串，验证有效性
        try:
            # 尝试编码再解码，确保是有效的 UTF-8
            text.encode('utf-8').decode('utf-8')
            return text
        except UnicodeEncodeError:
            # 如果编码失败，说明字符串可能包含无效字符
            fixed = text.encode('utf-8', errors='replace').decode('utf-8')
            if logger and fixed != text:
                logger.warning(f"编码修复: 使用 replace 模式处理无效字符")
            return fixed
        except UnicodeDecodeError:
            # 如果解码失败，说明字符串可能已经是错误的编码
            # 尝试其他修复方法
            try:
                fixed = text.encode('latin-1').decode('utf-8')
                if logger:
                    logger.info(f"编码修复: 通过 Latin-1 重新编码修复")
                return fixed
            except:
                # 最后尝试：使用 replace 模式
                try:
                    fixed = text.encode('latin-1', errors='replace').decode('utf-8', errors='replace')
                    if logger:
                        logger.warning(f"编码修复: 使用 replace 模式作为最后手段")
                    return fixed
                except:
                    return text
