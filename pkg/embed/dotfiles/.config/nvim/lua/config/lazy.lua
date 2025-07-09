-- Bootstrap lazy.nvim
local lazypath = vim.fn.stdpath("data") .. "/lazy/lazy.nvim"
if not (vim.uv or vim.loop).fs_stat(lazypath) then
  local lazyrepo = "https://github.com/folke/lazy.nvim.git"
  local out = vim.fn.system({ "git", "clone", "--filter=blob:none", "--branch=stable", lazyrepo, lazypath })
  if vim.v.shell_error ~= 0 then
    vim.api.nvim_echo({
      { "Failed to clone lazy.nvim:\n", "ErrorMsg" },
      { out, "WarningMsg" },
      { "\nPress any key to exit..." },
    }, true, {})
    vim.fn.getchar()
    os.exit(1)
  end
end
vim.opt.rtp:prepend(lazypath)

-- Setup lazy.nvim
require("lazy").setup({
  spec = {
    -- Colorschemes
    { "morhetz/gruvbox" },
    
    -- Navigation
    {
      "christoomey/vim-tmux-navigator",
      cmd = {
        "TmuxNavigateLeft",
        "TmuxNavigateDown",
        "TmuxNavigateUp",
        "TmuxNavigateRight",
        "TmuxNavigatePrevious",
      },
      keys = {
        { "<c-h>", "<cmd>TmuxNavigateLeft<cr>" },
        { "<c-j>", "<cmd>TmuxNavigateDown<cr>" },
        { "<c-k>", "<cmd>TmuxNavigateUp<cr>" },
        { "<c-l>", "<cmd>TmuxNavigateRight<cr>" },
      },
    },
    
    -- FZF
    {
      "junegunn/fzf",
      build = function()
        vim.fn["fzf#install"]()
      end,
    },
    { "junegunn/fzf.vim" },
    
    -- Project root detection
    { "airblade/vim-rooter" },
    
    -- Text manipulation
    { "tpope/vim-surround" },
    
    -- Copilot
    { 
      "github/copilot.vim",
      config = function()
        vim.g.copilot_filetypes = {
          ['dap-repl'] = false,
          markdown = true,
        }
      end
    },
    
    -- Optional: Treesitter for better syntax highlighting
    {
      "nvim-treesitter/nvim-treesitter",
      build = ":TSUpdate",
      event = "VeryLazy",
      config = function()
        require("nvim-treesitter.configs").setup({
          ensure_installed = { "lua", "vim", "vimdoc", "markdown", "markdown_inline" },
          sync_install = false,
          auto_install = true,
          highlight = {
            enable = true,
            disable = function(lang, buf)
              local max_filesize = 100 * 1024 -- 100 KB
              local ok, stats = pcall(vim.loop.fs_stat, vim.api.nvim_buf_get_name(buf))
              if ok and stats and stats.size > max_filesize then
                return true
              end
            end,
            additional_vim_regex_highlighting = false,
          },
        })
      end
    },
    
    -- Optional: Which-key for key binding help
    {
      "folke/which-key.nvim",
      event = "VeryLazy",
      opts = {},
      keys = {
        {
          "<leader>?",
          function()
            require("which-key").show({ global = false })
          end,
          desc = "Buffer Local Keymaps (which-key)",
        },
      },
    },
  },
  install = { colorscheme = { "gruvbox" } },
  -- Disable automatic update checks since we update via CI
  checker = { enabled = false }
})