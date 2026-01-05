-- nvim-dap configuration for gosynctasks
-- Copy this to your Neovim config or use with a project-local plugin like .nvim.lua

-- Required plugins:
-- - mfussenegger/nvim-dap
-- - leoluz/nvim-dap-go (optional but recommended)
-- - rcarriga/nvim-dap-ui (optional but recommended for UI)

local dap = require('dap')
local dapgo = require('dap-go') -- Optional: provides Go-specific helpers

-- Setup dap-go with default configurations
-- This will configure dlv adapter automatically
dapgo.setup({
  -- Additional dap-go configuration
  dap_configurations = {
    {
      type = "go",
      name = "Debug gosynctasks CLI",
      request = "launch",
      program = "${workspaceFolder}/cmd/gosynctasks",
      args = {},
    },
    {
      type = "go",
      name = "Debug gosynctasks with config",
      request = "launch",
      program = "${workspaceFolder}/cmd/gosynctasks",
      args = { "--config", "${HOME}/gosynctasks/config/" },
    },
    {
      type = "go",
      name = "Debug gosynctasks interactive",
      request = "launch",
      program = "${workspaceFolder}/cmd/gosynctasks",
      args = { "--config", "${HOME}/gosynctasks/config/" },
      console = "integratedTerminal",
    },
    {
      type = "go",
      name = "Debug gosynctasks add task",
      request = "launch",
      program = "${workspaceFolder}/cmd/gosynctasks",
      args = {
        "--config", "${HOME}/gosynctasks/config/",
        "MyList", "add", "Debug test task"
      },
    },
    {
      type = "go",
      name = "Debug gosynctasks sync",
      request = "launch",
      program = "${workspaceFolder}/cmd/gosynctasks",
      args = { "--config", "${HOME}/gosynctasks/config/", "sync" },
    },
    {
      type = "go",
      name = "Debug Current Test",
      request = "launch",
      mode = "test",
      program = "./${relativeFileDirname}",
    },
    {
      type = "go",
      name = "Attach to Process",
      mode = "local",
      request = "attach",
      processId = require('dap.utils').pick_process,
    },
  },
  delve = {
    -- Path to dlv executable (default: "dlv")
    path = "dlv",
    -- Set to true for verbose logging
    initialize_timeout_sec = 20,
    port = "${port}",
    args = {},
  },
})

-- Manual dlv adapter configuration (if not using dap-go)
-- Uncomment if you want to configure dlv manually without dap-go
--[[
dap.adapters.go = {
  type = 'server',
  port = '${port}',
  executable = {
    command = 'dlv',
    args = {'dap', '-l', '127.0.0.1:${port}'},
  }
}
]]

-- Key mappings (optional, adjust to your preference)
vim.keymap.set('n', '<F5>', function() dap.continue() end, { desc = 'Debug: Start/Continue' })
vim.keymap.set('n', '<F10>', function() dap.step_over() end, { desc = 'Debug: Step Over' })
vim.keymap.set('n', '<F11>', function() dap.step_into() end, { desc = 'Debug: Step Into' })
vim.keymap.set('n', '<F12>', function() dap.step_out() end, { desc = 'Debug: Step Out' })
vim.keymap.set('n', '<Leader>b', function() dap.toggle_breakpoint() end, { desc = 'Debug: Toggle Breakpoint' })
vim.keymap.set('n', '<Leader>B', function()
  dap.set_breakpoint(vim.fn.input('Breakpoint condition: '))
end, { desc = 'Debug: Set Conditional Breakpoint' })
vim.keymap.set('n', '<Leader>dr', function() dap.repl.open() end, { desc = 'Debug: Open REPL' })
vim.keymap.set('n', '<Leader>dl', function() dap.run_last() end, { desc = 'Debug: Run Last' })

-- DAP UI configuration (optional, requires nvim-dap-ui)
-- Uncomment if you have nvim-dap-ui installed
--[[
local dapui = require("dapui")
dapui.setup()

-- Auto-open/close UI
dap.listeners.after.event_initialized["dapui_config"] = function()
  dapui.open()
end
dap.listeners.before.event_terminated["dapui_config"] = function()
  dapui.close()
end
dap.listeners.before.event_exited["dapui_config"] = function()
  dapui.close()
end
]]

-- Virtual text configuration (optional, requires nvim-dap-virtual-text)
-- Uncomment if you have nvim-dap-virtual-text installed
--[[
require("nvim-dap-virtual-text").setup()
]]

return dap
