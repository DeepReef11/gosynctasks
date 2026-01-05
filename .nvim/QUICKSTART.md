# DAP Quick Start Guide

Get debugging working in 2 minutes.

## Prerequisites

Only `nvim-dap` is required. Install with your plugin manager:

**lazy.nvim:**
```lua
{
  'mfussenegger/nvim-dap',
}
```

**packer:**
```lua
use 'mfussenegger/nvim-dap'
```

## Setup (Choose One)

### Option 1: Minimal Setup (Fastest)

Add this to your `init.lua` or `~/.config/nvim/after/plugin/dap.lua`:

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

-- Basic configuration
dap.configurations.go = {
  {
    type = "go",
    name = "Debug",
    request = "launch",
    program = "${file}",
  },
  {
    type = "go",
    name = "Debug Test",
    request = "launch",
    mode = "test",
    program = "${file}",
  },
}

-- Keymaps
vim.keymap.set('n', '<F5>', function() dap.continue() end)
vim.keymap.set('n', '<F10>', function() dap.step_over() end)
vim.keymap.set('n', '<Leader>b', function() dap.toggle_breakpoint() end)
```

### Option 2: Project-Specific Setup

Source the provided config in this project:

```lua
-- In your init.lua, add this for project-specific configs
vim.cmd([[
  autocmd DirChanged * lua pcall(dofile, vim.fn.getcwd() .. '/.nvim/nvim-dap.lua')
]])

-- Or manually when in the project:
-- :luafile .nvim/nvim-dap.lua
```

### Option 3: Copy Minimal Config

```bash
cp .nvim/minimal-dap.lua ~/.config/nvim/after/plugin/dap.lua
nvim  # restart Neovim
```

## Usage

1. **Open a Go file:**
   ```bash
   nvim cmd/gosynctasks/main.go
   ```

2. **Set a breakpoint:**
   - Move cursor to a line
   - Press `<Leader>b` (or run `:lua require'dap'.toggle_breakpoint()`)
   - You should see a sign in the gutter

3. **Start debugging:**
   - Press `<F5>` (or run `:lua require'dap'.continue()`)
   - Select a configuration (use arrow keys, press Enter)
   - The debugger should start

4. **Debug commands:**
   - `<F5>` - Continue/Start
   - `<F10>` - Step Over
   - `<F11>` - Step Into (if you set it up)
   - `<Leader>dr` - Open REPL (if you set it up)
   - `:DapTerminate` - Stop debugging

## Test It Now

Quick test to verify it's working:

1. Start Neovim:
   ```bash
   cd /workspace/go/gosynctasks
   nvim cmd/gosynctasks/main.go
   ```

2. In Neovim, run this command:
   ```vim
   :lua require('dap').adapters.go
   ```

   If you see a table output, the adapter is configured! ✅

3. Set a breakpoint on line 20 (or any line in `main()`):
   ```vim
   :20
   :lua require('dap').toggle_breakpoint()
   ```

4. Start debugging:
   ```vim
   :lua require('dap').continue()
   ```

## Troubleshooting

### "module 'dap' not found"
- Install nvim-dap plugin and restart Neovim
- Run `:Lazy sync` (lazy.nvim) or `:PackerSync` (packer)

### "adapter go is not available"
- Make sure you sourced the config (run `:luafile .nvim/minimal-dap.lua`)
- Or add the config to your `init.lua` and restart

### dlv errors
```bash
# Verify dlv is installed and in PATH
which dlv
dlv version

# If not found, install it:
go install github.com/go-delve/delve/cmd/dlv@latest
```

### Check DAP status
```vim
" In Neovim:
:lua print(vim.inspect(require('dap').adapters.go))
:lua print(vim.inspect(require('dap').configurations.go))
```

## Next Steps

Once basic debugging works:

1. **Add UI** - Install `rcarriga/nvim-dap-ui` for visual debugging panels
2. **Add dap-go** - Install `leoluz/nvim-dap-go` for Go-specific features
3. **Customize** - Edit `.nvim/nvim-dap.lua` to add more configurations
4. **Read docs** - Check `.nvim/README.md` for detailed documentation

## Common Workflows

**Debug the CLI:**
```vim
:e cmd/gosynctasks/main.go
:lua require('dap').continue()
" Select 'Debug gosynctasks CLI' or 'Debug gosynctasks with config'
```

**Debug a test:**
```vim
:e backend/sqliteBackend_test.go
" Place cursor in a test function
:lua require('dap').continue()
" Select 'Debug Current Test'
```

**Quick debug current file:**
```vim
" Set breakpoint
<Leader>b

" Start debugging
<F5>

" Select 'Debug' configuration
```
