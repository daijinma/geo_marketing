import sys
import os

path = "backend/provider/deepseek.go"
with open(path, "r") as f:
    content = f.read()

# 1. Update Hijack Script (Remove strict URL filters, Add RAW logging)
start_marker = "\thijackScript := `(() => {"
end_marker = "\t})()`"

start_idx = content.find(start_marker)
if start_idx == -1:
    print("Error: Start marker not found")
    sys.exit(1)
end_idx = content.find(end_marker, start_idx)
if end_idx == -1:
    print("Error: End marker not found")
    sys.exit(1)
end_pos = end_idx + len(end_marker)

# New script: 
# - More lenient URL match (or just log all fetch responses to be sure)
# - Send __DS_RAW__ for every data line
new_script = "\thijackScript := `(() => {\n"
new_script += "\t\tconsole.log('__DS_DEBUG__: Hijack script injected v3 (RAW mode)');\n"
new_script += "\t\tconst originalFetch = window.fetch;\n"
new_script += "\t\twindow.fetch = async (...args) => {\n"
new_script += "\t\t\tconst response = await originalFetch(...args);\n"
new_script += "\t\t\t\n"
new_script += "\t\t\ttry {\n"
new_script += "\t\t\t\tconst urlStr = args[0] instanceof Request ? args[0].url : String(args[0]);\n"
new_script += "\t\t\t\t// Log all requests for debugging\n"
new_script += "\t\t\t\tconsole.log('__DS_REQ__:' + urlStr);\n"
new_script += "\t\t\t\t\n"
new_script += "\t\t\t\t// Very broad matching to catch everything related to chat\n"
new_script += "\t\t\t\tif (urlStr.includes('chat') || urlStr.includes('completion')) {\n"
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
new_script += "\t\t\t\t\t\t\t\tbuffer = lines.pop();\n"
new_script += "\t\t\t\t\t\t\t\t\n"
new_script += "\t\t\t\t\t\t\t\tfor (const line of lines) {\n"
new_script += "\t\t\t\t\t\t\t\t\tif (line.trim() === '') continue;\n"
new_script += "\t\t\t\t\t\t\t\t\t// Log RAW line for Go to see\n"
new_script += "\t\t\t\t\t\t\t\t\tconsole.log('__DS_RAW__:' + line);\n"
new_script += "\t\t\t\t\t\t\t\t}\n"
new_script += "\t\t\t\t\t\t\t}\n"
new_script += "\t\t\t\t\t\t} catch (e) {\n"
new_script += "\t\t\t\t\t\t\tconsole.error('__DS_ERROR__:', e);\n"
new_script += "\t\t\t\t\t\t}\n"
new_script += "\t\t\t\t\t})();\n"
new_script += "\t\t\t\t}\n"
new_script += "\t\t\t} catch (err) {\n"
new_script += "\t\t\t}\n"
new_script += "\t\t\t\n"
new_script += "\t\t\treturn response;\n"
new_script += "\t\t};\n"
new_script += "\t})()`"

# Indent check: The file uses tabs. 
# hijackScript starts with 1 tab.
content = content[:start_idx] + new_script + content[end_pos:]


# 2. Update Go Logic to handle __DS_RAW__ and print everything
chunk_handler_start = 'if strings.HasPrefix(logMsg, "__DS_LINE__:") {'
handler_idx = content.find(chunk_handler_start)
if handler_idx == -1:
    print("Error: Chunk handler start not found")
    sys.exit(1)

# We want to replace the whole block until the closing brace of the for loop.
# From previous edit, we know the structure.
loop_end_marker = "\t\t\t}\n\t\t\treturn false"
loop_end_idx = content.find(loop_end_marker, handler_idx)
if loop_end_idx == -1:
    print("Error: Loop end marker not found")
    sys.exit(1)

new_handler = """if strings.HasPrefix(logMsg, "__DS_RAW__:") {
					line := strings.TrimPrefix(logMsg, "__DS_RAW__:")
					line = strings.TrimSpace(line)
					
					// 1. 无论如何先打印原始数据 (截断过长的)
					if len(line) > 500 {
						d.logger.InfoWithContext(ctx, "[DEEPSEEK-RAW] " + line[:500] + "...", nil, nil)
					} else {
						d.logger.InfoWithContext(ctx, "[DEEPSEEK-RAW] " + line, nil, nil)
					}

					// 2. 尝试解析逻辑 (保持不变，作为尝试)
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
							
							// DeepSeek 经典结构 p=results
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
											if e.URL == cit.URL { exists = true; break }
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
								if len(packet.V) > 0 {
									if err := json.Unmarshal(packet.V, &vObj); err == nil {
										for _, frag := range vObj.Response.Fragments {
											for _, q := range frag.Queries {
												capturedQueries = append(capturedQueries, q.Query)
											}
										}
									}
								}
							}
							
							// 备用结构探测
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

content = content[:handler_idx] + new_handler + content[loop_end_idx:]

with open(path, "w") as f:
    f.write(content)

print("Success")
