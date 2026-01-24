import sys
import os
import re

path = "backend/provider/deepseek.go"
with open(path, "r") as f:
    content = f.read()

print(f"File size before: {len(content)}")

# 1. Fix RuntimeEnable (Uncomment it)
# Pattern: // proto.RuntimeEnable{}.Call(page)
# We want to replace it with proper error handling
runtime_pattern = r"//\s*proto\.RuntimeEnable\{\}\.Call\(page\)"
runtime_fix = """if err := (proto.RuntimeEnable{}).Call(page); err != nil {
			d.logger.WarnWithContext(ctx, "[DEEPSEEK-RPA] Failed to enable Runtime domain", map[string]interface{}{"error": err.Error()}, nil)
		}"""

if re.search(runtime_pattern, content):
    content = re.sub(runtime_pattern, runtime_fix, content)
    print("Fixed RuntimeEnable")
else:
    print("RuntimeEnable pattern not found (might be already fixed or different)")

# 2. Replace the ENTIRE hijackScript block
# We know it starts with `hijackScript := `...` and ends with `}` (indented)
# We'll use a regex to capture the whole backtick string.
# Be careful with nested backticks (unlikely in JS here) but Go uses backticks for string.
# The script is likely: hijackScript := `...`

# We will search for the variable definition line and the closing backtick.
start_marker = "\thijackScript := `"
end_marker = "\t}`" # This assumes the old format ends with indented }`

start_pos = content.find(start_marker)
if start_pos == -1:
    print("Start marker for hijackScript not found")
    sys.exit(1)

# Find the end. It's the first `}` followed by backtick after start_pos.
# Actually, the file uses `}` line.
end_pos = content.find("}`", start_pos)
if end_pos == -1:
    print("End marker for hijackScript not found")
    sys.exit(1)
end_pos += 2 # include }`

# New JS Content (IIFE + RAW Mode)
new_js = """hijackScript := `(() => {
		console.log('__DS_DEBUG__: Hijack script injected v4 (RAW)');
		const originalFetch = window.fetch;
		window.fetch = async (...args) => {
			const response = await originalFetch(...args);
			
			try {
				const urlStr = args[0] instanceof Request ? args[0].url : String(args[0]);
				console.log('__DS_REQ__:' + urlStr);
				
				// Broad match for any chat/completion API
				if (urlStr.includes('chat') || urlStr.includes('completion')) {
					const clone = response.clone();
					const reader = clone.body.getReader();
					const decoder = new TextDecoder();
					
					(async () => {
						try {
							let buffer = '';
							while (true) {
								const { done, value } = await reader.read();
								if (done) break;
								const chunk = decoder.decode(value, { stream: true });
								buffer += chunk;
								
								const lines = buffer.split('\\n');
								buffer = lines.pop();
								
								for (const line of lines) {
									if (line.trim() === '') continue;
									// Send RAW line to Go
									console.log('__DS_RAW__:' + line);
								}
							}
						} catch (e) {
							console.error('__DS_ERROR__:', e);
						}
					})();
				}
			} catch (err) {
			}
			return response;
		};
	})()`"""

# Verify indentation: The file uses tabs. 
# My replacement string above uses tabs for the first line? No, I put `hijackScript` at start.
# I need to match the indentation of the file.
# The file has `\thijackScript := ...`.
# My `new_js` should replace `hijackScript := ...` so I should prepend `\t`.
new_js_formatted = "\t" + new_js

content = content[:start_pos] + new_js_formatted + content[end_pos:]
print("Replaced hijackScript")


# 3. Replace the Go event handler loop logic
# We want to find the `page.EachEvent` block and replace the logic inside.
# Specifically, replace `if strings.HasPrefix(logMsg, "__DS_CHUNK__:") { ... }`
# with the new `__DS_RAW__` handler.

chunk_handler_start = 'if strings.HasPrefix(logMsg, "__DS_CHUNK__:") {'
start_handler = content.find(chunk_handler_start)

if start_handler == -1:
    print("Chunk handler start not found")
    # It might be `__DS_LINE__` if I partially updated it?
    # Or `JSON.stringify` logic?
    # Let's try to find the start of the `if` block more loosely.
    # Look for the logMsg prefix check.
    match = re.search(r'if strings\.HasPrefix\(logMsg, "__DS_.*":\) \{', content)
    if match:
        start_handler = match.start()
        print(f"Found handler start at {start_handler}")
    else:
        print("Could not find log handler block")
        sys.exit(1)

# Find the end of this block.
# We assume it ends before the closing of the `range e.Args` loop.
# The loop closes with `\t\t\t}` (3 tabs) followed by `\t\t\treturn false`?
# In the file I read:
# 00294| 			}
# 00295| 			return false
# This is `page.EachEvent` return.
# The args loop is inside.

# Let's find the `citationMu.Unlock()` as a reference point for the old block end?
# Or just find the `return false` at the end of `EachEvent` callback and work backwards.

callback_return = content.find("return false", start_handler)
if callback_return == -1:
    print("Callback return not found")
    sys.exit(1)

# The `}` before `return false` closes the `range e.Args`.
# The `}` before THAT should be the end of our handler block?
# Not necessarily, there are many braces.

# Let's use the anchor: `citationMu.Unlock()`
unlock_pos = content.find("citationMu.Unlock()", start_handler)
if unlock_pos != -1:
    # Find the closing braces after unlock
    # Usually `}\n\t\t\t\t\t}` etc.
    # We want to replace EVERYTHING from `start_handler` to the end of the if block.
    # The if block is inside `for _, arg := range e.Args {`.
    # So we replace up to the closing brace of the if block.
    
    # Let's define the new handler first.
    new_handler = """if strings.HasPrefix(logMsg, "__DS_RAW__:") {
					line := strings.TrimPrefix(logMsg, "__DS_RAW__:")
					line = strings.TrimSpace(line)
					
					// Raw debug log
					if len(line) > 500 {
						d.logger.InfoWithContext(ctx, "[DEEPSEEK-RAW] " + line[:500] + "...", nil, nil)
					} else {
						d.logger.InfoWithContext(ctx, "[DEEPSEEK-RAW] " + line, nil, nil)
					}

					// Parse logic
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
							
							// 1. DeepSeek p=results
							if strings.Contains(packet.P, "results") {
								var results []struct {
									URL          string `json:"url"`
									Title        string `json:"title"`
									Snippet      string `json:"snippet"`
									QueryIndexes []int  `json:"query_indexes"`
								}
								if err := json.Unmarshal(packet.V, &results); err == nil {
									d.logger.InfoWithContext(ctx, "[DEEPSEEK-JS] Captured citations", map[string]interface{}{"count": len(results)}, nil)
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
								// Queries
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
							
							// 2. Root results
							if len(packet.Results) > 0 {
								var results []struct {
									URL string `json:"url"`
									Title string `json:"title"`
								}
								if err := json.Unmarshal(packet.Results, &results); err == nil && len(results) > 0 {
									d.logger.InfoWithContext(ctx, "[DEEPSEEK-JS] Captured citations (root)", map[string]interface{}{"count": len(results)}, nil)
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
    
    # Look for the end of the current handler.
    # It ends when we find the closing brace that matches the indentation of the start_handler.
    # But counting indentation in regex is hard.
    # Let's use the fact that the next thing is the end of the loop args.
    
    # Search for the loop end sequence:
    # 			}
    # 			return false
    # This assumes the handler is the last thing in the loop.
    # In the file:
    # 00294| 			}
    # 00295| 			return false
    
    # So we replace from `start_handler` to `loop_end_marker`.
    loop_end_pattern = r"\n\s*\}\n\s*return false"
    loop_end_match = re.search(loop_end_pattern, content[start_handler:])
    
    if loop_end_match:
        end_handler_rel = loop_end_match.start()
        end_handler_abs = start_handler + end_handler_rel
        
        content = content[:start_handler] + new_handler + content[end_handler_abs:]
        print("Replaced log handler")
    else:
        print("Loop end pattern not found")
        sys.exit(1)

with open(path, "w") as f:
    f.write(content)

print(f"File size after: {len(content)}")
print("Success")
