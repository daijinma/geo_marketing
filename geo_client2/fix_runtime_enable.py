import sys
import os

path = "backend/provider/deepseek.go"
with open(path, "r") as f:
    content = f.read()

# Replace the commented out line with proper error handling
old_line = "// proto.RuntimeEnable{}.Call(page)"
new_line = """if err := (proto.RuntimeEnable{}).Call(page); err != nil {
			d.logger.WarnWithContext(ctx, "[DEEPSEEK-RPA] Failed to enable Runtime domain", map[string]interface{}{"error": err.Error()}, nil)
		}"""

if old_line in content:
    content = content.replace(old_line, new_line)
    print("Replaced commented line")
else:
    # Maybe it's not commented but just needs wrapping?
    # Or maybe it's commented with different spacing?
    # Let's check for the uncommented version too
    old_line_uncommented = "proto.RuntimeEnable{}.Call(page)"
    if old_line_uncommented in content:
        content = content.replace(old_line_uncommented, new_line)
        print("Replaced uncommented line")
    else:
        print("Could not find RuntimeEnable line")
        # Print context
        idx = content.find("RuntimeEnable")
        if idx != -1:
            print(f"Found RuntimeEnable at {idx}: {content[idx-20:idx+40]}")
        sys.exit(1)

with open(path, "w") as f:
    f.write(content)

print("Success")
