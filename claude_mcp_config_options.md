# Claude MCP Configuration Options for Filesystem Server

## Option 1: Basic Setup (Recommended)
```json
{
  "mcpServers": {
    "filesystem": {
      "command": "/Users/$USER/github.com/$USER/bin/filesystem",
      "args": [
        "/Users/$USER/Documents",
        "/Users/$USER/Desktop", 
        "/Users/$USER/Projects"
      ]
    }
  }
}
```

## Option 2: Secure Setup (Single Directory)
```json
{
  "mcpServers": {
    "TAK": { ... },
    "OSM": { ... },
    "filesystem": {
      "command": "/Users/$USER/github.com/$USER/bin/filesystem",
      "args": [
        "/Users/$USER/claude_workspace"
      ]
    }
  }
}
```

## Option 3: Development Setup (More Directories)
```json
{
  "mcpServers": {
    "TAK": { ... },
    "OSM": { ... },
    "filesystem": {
      "command": "/Users/$USER/github.com/$USER/bin/filesystem",
      "args": [
        "/Users/$USER/Documents",
        "/Users/$USER/Desktop",
        "/Users/$USER/Projects",
        "/Users/$USER/github.com",
        "/tmp"
      ]
    }
  }
}
```

## Option 4: Using Configuration File
```json
{
  "mcpServers": {
    "TAK": { ... },
    "OSM": { ... },
    "filesystem": {
      "command": "/Users/$USER/github.com/$USER/bin/filesystem",
      "args": [
        "-config",
        "/Users/$USER/github.com/$USER/claude_config.yaml"
      ]
    }
  }
}
```

## Security Recommendations

### ✅ Safe Directories
- `/Users/$USER/Documents`
- `/Users/$USER/Desktop`
- `/Users/$USER/Projects`
- `/Users/$USER/claude_workspace` (dedicated folder)
- `/tmp` (for temporary files)

### ⚠️ Be Careful With
- `/Users/$USER` (entire home directory)
- `/Users/$USER/github.com` (all your code)
- System directories like `/etc`, `/var`

### 🔒 Most Secure Approach
Create a dedicated directory for Claude:
```bash
mkdir -p /Users/$USER/claude_workspace
```

Then use:
```json
"args": ["/Users/$USER/claude_workspace"]
```

## Directory Setup Commands

```bash
# Create dedicated workspace
mkdir -p /Users/$USER/claude_workspace

# Make sure binary is built and executable
cd /Users/$USER/github.com/$USER/filesystem
make build
chmod +x bin/filesystem

# Test the configuration
./bin/filesystem /Users/$USER/claude_workspace
``` 