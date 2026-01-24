import sys
import os

path = "backend/provider/deepseek.go"
with open(path, "r") as f:
    content = f.read()

# 1. Fix the Hijack Script
start_marker = "\thijackScript := `() => {"
end_marker = "\t}`"

start_idx = content.find(start_marker)
if start_idx == -1:
    print(f"Error: Start marker not found")
    # Debug print around likely area if not found
    print("Content around line 84:")
    lines = content.split('\n')
    if len(lines) > 80:
        for i in range(80, 90):
            print(f"{i}: {repr(lines[i])}")
    sys.exit(1)

end_idx = content.find(end_marker, start_idx)
if end_idx == -1:
    print("Error: End marker not found")
    sys.exit(1)

end_pos = end_idx + len(end_marker)

new_script = "\thijackScript := `(() => {\n"
new_script += "\t\tconsole.log('__DS_DEBUG__: Hijack script injected');\n"
new_script += "\t\tconst originalFetch = window.fetch;\n"
new_script += "\t\twindow.fetch = async (...args) => {\n"
new_script += "\t\t\tconst response = await originalFetch(...args);\n"
new_script += "\t\t\t\n"
new_script += "\t\t\ttry {\n"
new_script += "\t\t\t\tconst urlStr = args[0] instanceof Request ? args[0].url : String(args[0]);\n"
new_script += "\t\t\t\tconsole.log('__DS_REQ__:' + urlStr);\n"
new_script += "\t\t\t\t\n"
new_script += "\t\t\t\t// 宽松匹配 chat 接口\n"
new_script += "\t\t\t\tif (urlStr.includes('chat/completions') || urlStr.includes('chat/completion') || urlStr.includes('api/v0/chat')) {\n"
new_script += "\t\t\t\t\tconst clone = response.clone();\n"
new_script += "\t\t\t\t\tconst reader = clone.body.getReader();\n"
new_script += "\t\t\t\t\tconst decoder = new TextDecoder();\n"
new_script += "\t\t\t\t\t\n"
new_script += "\t\t\t\t\t// 异步读取流，不阻塞原始请求\n"
new_script += "\t\t\t\t\t(async () => {\n"
new_script += "\t\t\t\t\t\ttry {\n"
new_script += "\t\t\t\t\t\t\twhile (true) {\n"
new_script += "\t\t\t\t\t\t\t\tconst { done, value } = await reader.read();\n"
new_script += "\t\t\t\t\t\t\t\tif (done) break;\n"
new_script += "\t\t\t\t\t\t\t\tconst chunk = decoder.decode(value, { stream: true });\n"
new_script += "\t\t\t\t\t\t\t\t// 使用特殊前缀标记数据\n"
new_script += "\t\t\t\t\t\t\t\tconsole.log('__DS_CHUNK__:' + chunk);\n"
new_script += "\t\t\t\t\t\t\t}\n"
new_script += "\t\t\t\t\t\t} catch (e) {\n"
new_script += "\t\t\t\t\t\t\tconsole.error('__DS_ERROR__:', e);\n"
new_script += "\t\t\t\t\t\t}\n"
new_script += "\t\t\t\t\t})();\n"
new_script += "\t\t\t\t}\n"
new_script += "\t\t\t} catch (err) {\n"
new_script += "\t\t\t\t// 忽略非关键错误\n"
new_script += "\t\t\t}\n"
new_script += "\t\t\t\n"
new_script += "\t\t\treturn response;\n"
new_script += "\t\t};\n"
new_script += "\t})()`"

content = content[:start_idx] + new_script + content[end_pos:]

# 2. Add log handling
log_handler_marker = 'if strings.HasPrefix(logMsg, "__DS_CHUNK__:") {'

if log_handler_marker not in content:
    print("Error: Log handler marker not found")
    sys.exit(1)

new_log_handler = """if strings.HasPrefix(logMsg, "__DS_DEBUG__:") {
					d.logger.InfoWithContext(ctx, "[DEEPSEEK-JS] " + strings.TrimPrefix(logMsg, "__DS_DEBUG__:"), nil, nil)
					return false
				}
				if strings.HasPrefix(logMsg, "__DS_REQ__:") {
					d.logger.InfoWithContext(ctx, "[DEEPSEEK-JS] Request: " + strings.TrimPrefix(logMsg, "__DS_REQ__:"), nil, nil)
					return false
				}

				if strings.HasPrefix(logMsg, "__DS_CHUNK__:") {"""

content = content.replace(log_handler_marker, new_log_handler)

with open(path, "w") as f:
    f.write(content)

print("Success")
