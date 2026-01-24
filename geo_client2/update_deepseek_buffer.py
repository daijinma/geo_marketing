import sys
import os

path = "backend/provider/deepseek.go"
with open(path, "r") as f:
    content = f.read()

# 1. New Hijack Script with Buffer
# It seems my previous attempt failed because I searched for the IIFE version `(() => {` 
# but the file might still have the arrow function `() => {` because I might have reverted or something went wrong?
# Ah, I see from the debug output: `hijackScript := `() => {`
# Wait, I thought I updated it to `(() => {`.
# The previous `update_deepseek.py` claimed "Success".
# Maybe I should just look for `hijackScript :=` and replace until `}`

start_marker = "hijackScript := `"
start_idx = content.find(start_marker)
if start_idx == -1:
    print("Error: Start marker not found")
    sys.exit(1)

# Find the matching closing backtick.
# It should be after some indented `}`
end_marker = "\t})()`" # Try finding the IIFE end first
end_idx = content.find(end_marker, start_idx)

if end_idx == -1:
    # If not found, try the old arrow function end
    end_marker = "\t}`"
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
new_script += "\t\t\t\t\t(async () => {\n"
new_script += "\t\t\t\t\t\ttry {\n"
new_script += "\t\t\t\t\t\t\tlet buffer = '';\n"
new_script += "\t\t\t\t\t\t\twhile (true) {\n"
new_script += "\t\t\t\t\t\t\t\tconst { done, value } = await reader.read();\n"
new_script += "\t\t\t\t\t\t\t\tif (done) break;\n"
new_script += "\t\t\t\t\t\t\t\tconst chunk = decoder.decode(value, { stream: true });\n"
new_script += "\t\t\t\t\t\t\t\tbuffer += chunk;\n"
new_script += "\t\t\t\t\t\t\t\t\n"
new_script += "\t\t\t\t\t\t\t\tconst lines = buffer.split('\\n');\n"
new_script += "\t\t\t\t\t\t\t\tbuffer = lines.pop(); \n"
new_script += "\t\t\t\t\t\t\t\t\n"
new_script += "\t\t\t\t\t\t\t\tfor (const line of lines) {\n"
new_script += "\t\t\t\t\t\t\t\t\tif (line.trim() === '') continue;\n"
new_script += "\t\t\t\t\t\t\t\t\tconsole.log('__DS_LINE__:' + line);\n"
new_script += "\t\t\t\t\t\t\t\t}\n"
new_script += "\t\t\t\t\t\t\t}\n"
new_script += "\t\t\t\t\t\t} catch (e) {\n"
new_script += "\t\t\t\t\t\t\tconsole.error('__DS_ERROR__:', e);\n"
new_script += "\t\t\t\t\t\t}\n"
new_script += "\t\t\t\t\t})();\n"
new_script += "\t\t\t\t}\n"
new_script += "\t\t\t} catch (err) {\n"
new_script += "\t\t\t\t// Ignore non-critical errors\n"
new_script += "\t\t\t}\n"
new_script += "\t\t\t\n"
new_script += "\t\t\treturn response;\n"
new_script += "\t\t};\n"
new_script += "\t})()`"

# Be careful about indentation when replacing.
# The `hijackScript := ` line starts with one tab.
# We should probably preserve `hijackScript := ` from original or just overwrite correctly.
# My new_script starts with `\thijackScript`.

# Adjust start_idx to the beginning of the line
line_start_idx = content.rfind('\n', 0, start_idx) + 1
content = content[:line_start_idx] + new_script + content[end_pos:]


# 2. Update Go loop
chunk_handler_start = 'if strings.HasPrefix(logMsg, "__DS_CHUNK__:") {'
handler_idx = content.find(chunk_handler_start)
if handler_idx == -1:
    print("Error: Chunk handler start not found")
    sys.exit(1)

new_handler = """if strings.HasPrefix(logMsg, "__DS_LINE__:") {
					line := strings.TrimPrefix(logMsg, "__DS_LINE__:")
					line = strings.TrimSpace(line)
					
					if strings.HasPrefix(line, "data: ") {
						jsonStr := strings.TrimPrefix(line, "data: ")
						if jsonStr == "[DONE]" {
							return false
						}

						var packet struct {
							P       string          `json:"p"`
							V       json.RawMessage `json:"v"`
							Choices []struct {
								Delta struct {
									Content string `json:"content"`
								} `json:"delta"`
							} `json:"choices"`
							Type    string          `json:"type"` 
							Results json.RawMessage `json:"results"`
						}

						if err := json.Unmarshal([]byte(jsonStr), &packet); err == nil {
							citationMu.Lock()
							
							// Debug: 如果包含 http 或者是 result 类型，打印一下结构以便调试
							if strings.Contains(jsonStr, "http") || strings.Contains(jsonStr, "result") {
								// d.logger.InfoWithContext(ctx, "[DEEPSEEK-JS] Potential citation packet", map[string]interface{}{"preview": truncateString(jsonStr, 200)}, nil)
							}

							// 1. 原始 p/v 结构 (DeepSeek 经典)
							if strings.Contains(packet.P, "results") {
								var results []struct {
									URL          string `json:"url"`
									Title        string `json:"title"`
									Snippet      string `json:"snippet"`
									QueryIndexes []int  `json:"query_indexes"`
								}
								if err := json.Unmarshal(packet.V, &results); err == nil {
									d.logger.InfoWithContext(ctx, "[DEEPSEEK-JS] Captured citations (p=results)", map[string]interface{}{"count": len(results)}, nil)
									for _, res := range results {
										cit := Citation{
											URL:          res.URL,
											Title:        res.Title,
											Snippet:      res.Snippet,
											QueryIndexes: res.QueryIndexes,
										}
										if u, err := url.Parse(res.URL); err == nil {
											cit.Domain = u.Host
										}
										
										exists := false
										for _, e := range capturedCitations {
											if e.URL == cit.URL {
												exists = true
												break
											}
										}
										if !exists && cit.URL != "" {
											capturedCitations = append(capturedCitations, cit)
										}
									}
								}
							} else if packet.P == "response" || packet.P == "" {
								// 提取 Queries
								var vObj struct {
									Response struct {
										Fragments []struct {
											Queries []struct {
												Query string `json:"query"`
											} `json:"queries"`
										} `json:"fragments"`
									} `json:"response"`
								}
								// 尝试解析 V
								if len(packet.V) > 0 {
									if err := json.Unmarshal(packet.V, &vObj); err == nil {
										for _, frag := range vObj.Response.Fragments {
											for _, q := range frag.Queries {
												exists := false
												for _, eq := range capturedQueries {
													if eq == q.Query {
														exists = true
														break
													}
												}
												if !exists && q.Query != "" {
													capturedQueries = append(capturedQueries, q.Query)
												}
											}
										}
									}
								}
							}
							
							// 2. 备用结构探测
							if len(packet.Results) > 0 {
								var results []struct {
									URL string `json:"url"`
									Title string `json:"title"`
								}
								if err := json.Unmarshal(packet.Results, &results); err == nil && len(results) > 0 {
									d.logger.InfoWithContext(ctx, "[DEEPSEEK-JS] Captured citations (root results)", map[string]interface{}{"count": len(results)}, nil)
									for _, res := range results {
										cit := Citation{URL: res.URL, Title: res.Title}
										if u, err := url.Parse(res.URL); err == nil {
											cit.Domain = u.Host
										}
										exists := false
										for _, e := range capturedCitations {
											if e.URL == cit.URL { exists = true; break }
										}
										if !exists && cit.URL != "" {
											capturedCitations = append(capturedCitations, cit)
										}
									}
								}
							}

							citationMu.Unlock()
						}
					}
				}"""

loop_end_marker = "\t\t\t}\n\t\t\treturn false"
loop_end_idx = content.find(loop_end_marker, handler_idx)
if loop_end_idx == -1:
    print("Error: Loop end marker not found")
    sys.exit(1)

content = content[:handler_idx] + new_handler + content[loop_end_idx:]

with open(path, "w") as f:
    f.write(content)

print("Success")
