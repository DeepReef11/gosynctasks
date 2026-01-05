# Neovim DAP Setup for gosynctasks

This directory contains configurations for debugging gosynctasks with Neovim using DAP (Debug Adapter Protocol) and dlv (Delve).

## Prerequisites

1. **Install dlv (Delve)**:
   ```bash
   go install github.com/go-delve/delve/cmd/dlv@latest
   ```

2. **Install Neovim plugins** (using your plugin manager):
   - `mfussenegger/nvim-dap` (required)
   - `leoluz/nvim-dap-go` (recommended for Go-specific features)
   - `rcarriga/nvim-dap-ui` (optional, provides a nice UI)
   - `theHamsta/nvim-dap-virtual-text` (optional, shows variable values inline)

## Setup Methods

### Method 1: Using .vscode/launch.json (Recommended)

Many Neovim DAP plugins can read VSCode's `launch.json` format. The configurations are already in `.vscode/launch.json`.

If you're using a plugin like `nvim-dap-vscode-js` or similar that reads `launch.json`, you may be able to use these configurations directly.

### Method 2: Direct Lua Configuration

Copy the contents of `nvim-dap.lua` to your Neovim configuration, or source it directly:

**Option A: Add to your init.lua**
```lua
-- In your init.lua or after/plugin/dap.lua
require('gosynctasks-project-root/.nvim/nvim-dap')
```

**Option B: Project-local configuration**

If you use a plugin like `klen/nvim-config-local` or `.nvim.lua` support:

1. Create `.nvim.lua` in project root:
   ```lua
   -- Load project-specific DAP config
   require('.nvim/nvim-dap')
   ```

2. Enable local config in your Neovim config:
   ```lua
   -- Using nvim-config-local
   require('config-local').setup({
     config_files = { '.nvim.lua' },
     silent = false,
   })
   ```

### Method 3: Minimal Setup

Minimal configuration without extra plugins:

```lua
local dap = require('dap')

-- Configure dlv adapter
dap.adapters.go = {
  type = 'server',
  port = '${port}',
  executable = {
    command = 'dlv',
    args = {'dap', '-l', '127.0.0.1:${port}'},
  }
}

-- Add a basic configuration
dap.configurations.go = {
  {
    type = "go",
    name = "Debug",
    request = "launch",
    program = "${file}",
  },
}
```

## Usage

### Starting a Debug Session

1. **Open a file** in the gosynctasks project (e.g., `cmd/gosynctasks/main.go`)

2. **Set breakpoints** by placing your cursor on a line and:
   - Run `:lua require'dap'.toggle_breakpoint()`
   - Or use keymap `<Leader>b` (if configured)

3. **Start debugging**:
   - Run `:lua require'dap'.continue()`
   - Or use keymap `<F5>` (if configured)
   - Select a configuration from the list

4. **Control execution**:
   - `<F5>` - Continue/Start
   - `<F10>` - Step Over
   - `<F11>` - Step Into
   - `<F12>` - Step Out
   - `<Leader>dr` - Open REPL
   - `<Leader>dl` - Run last configuration

### Available Configurations

1. **Debug gosynctasks CLI** - Debug the main application
2. **Debug gosynctasks with config** - Debug with test config
3. **Debug gosynctasks interactive** - Debug interactive mode (list selection)
4. **Debug gosynctasks add task** - Debug the 'add task' command
5. **Debug gosynctasks sync** - Debug sync functionality
6. **Debug Current Test** - Debug the test under cursor
7. **Debug All Tests in Current Package** - Debug all tests in current directory
8. **Attach to Process** - Attach to a running Go process

### Example Workflow

```vim
" 1. Open main.go
:e cmd/gosynctasks/main.go

" 2. Set a breakpoint on line 50
:50 | :lua require'dap'.toggle_breakpoint()

" 3. Start debugging
:lua require'dap'.continue()

" 4. Step through code
" Press F10 to step over, F11 to step into
```

### Debugging Tests

1. Open a test file (e.g., `backend/sqliteBackend_test.go`)
2. Place cursor on or inside a test function
3. Start debugging with "Debug Current Test" configuration
4. Or use: `:lua require('dap-go').debug_test()` (if using dap-go)

### Using DAP UI (if installed)

With `nvim-dap-ui`, you get automatic UI panels:

```lua
-- The UI automatically opens when debugging starts
-- Manually toggle:
:lua require'dapui'.toggle()

-- Available windows:
-- - Variables
-- - Watches
-- - Call Stack
-- - Breakpoints
-- - Console/REPL
```

## Customization

### Modify Arguments

To debug with specific arguments, modify the `args` array in configurations:

```lua
{
  type = "go",
  name = "Debug MyList tasks",
  request = "launch",
  program = "${workspaceFolder}/cmd/gosynctasks",
  args = {
    "--config", "./gosynctasks/config",
    "MyList",  -- Your list name
    "get"      -- Command
  },
}
```

### Environment Variables

Add environment variables to any configuration:

```lua
{
  type = "go",
  name = "Debug with env vars",
  request = "launch",
  program = "${workspaceFolder}/cmd/gosynctasks",
  env = {
    GOSYNCTASKS_NEXTCLOUD_USERNAME = "testuser",
    GOSYNCTASKS_DEBUG = "true",
  },
}
```

### Conditional Breakpoints

Set breakpoints with conditions:

```vim
" Set condition breakpoint
:lua require'dap'.set_breakpoint(vim.fn.input('Condition: '))

" Example: Break only when err != nil
" Condition: err != nil
```

## Troubleshooting

### dlv not found
```bash
# Ensure dlv is in PATH
which dlv

# If not found, install it
go install github.com/go-delve/delve/cmd/dlv@latest

# Add GOPATH/bin to PATH if needed
export PATH=$PATH:$(go env GOPATH)/bin
```

### DAP not starting
```vim
" Check DAP status
:lua require'dap'.status()

" View DAP output
:DapShowLog
```

### Port already in use
```vim
" dlv uses random ports by default
" If you get port errors, ensure no other dlv instances are running:
```
```bash
pkill dlv
```

## References

- [nvim-dap documentation](https://github.com/mfussenegger/nvim-dap)
- [nvim-dap-go documentation](https://github.com/leoluz/nvim-dap-go)
- [Delve documentation](https://github.com/go-delve/delve)
