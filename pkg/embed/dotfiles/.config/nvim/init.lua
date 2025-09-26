-- Set leader key early
vim.g.mapleader = " "
vim.g.maplocalleader = "\\"

-- Core vim settings
vim.opt.number = true
vim.opt.relativenumber = true
vim.opt.signcolumn = "number"
vim.opt.mouse = "a"
vim.opt.tabstop = 2
vim.opt.softtabstop = 2
vim.opt.shiftwidth = 2
vim.opt.expandtab = true
vim.opt.smartindent = true
vim.opt.wrap = false
vim.opt.hlsearch = true
vim.opt.termguicolors = true
vim.opt.updatetime = 300
vim.opt.laststatus = 2
vim.opt.completeopt:remove("preview")
vim.opt.autoread = true

-- Set backup/swap/undo directories
vim.opt.backupdir = vim.fn.expand("~/.vim/backup//")
vim.opt.directory = vim.fn.expand("~/.vim/swap//")
vim.opt.undodir = vim.fn.expand("~/.vim/undo//")

-- Basic keymaps
vim.keymap.set('n', '<leader>w', ':w<CR>')
vim.keymap.set('n', '<leader>h', ':nohlsearch<CR>')

-- Tab navigation
vim.keymap.set('n', '<leader>o', ':tabnext<CR>')
vim.keymap.set('n', '<leader>n', ':tabprevious<CR>')
vim.keymap.set('n', '<leader><leader>n', ':-tabmove<CR>')
vim.keymap.set('n', '<leader><leader>o', ':+tabmove<CR>')
vim.keymap.set('n', '<leader>tn', ':tabnew %<CR>')
vim.keymap.set('n', '<leader>tc', ':tabclose<CR>')

-- Window splitting
vim.keymap.set('n', '<leader>swn', ':topleft vnew<CR>')
vim.keymap.set('n', '<leader>swo', ':botright vnew<CR>')
vim.keymap.set('n', '<leader>swi', ':topleft new<CR>')
vim.keymap.set('n', '<leader>swe', ':botright new<CR>')
vim.keymap.set('n', '<leader>sn', ':leftabove vnew<CR>')
vim.keymap.set('n', '<leader>so', ':rightbelow vnew<CR>')
vim.keymap.set('n', '<leader>si', ':leftabove new<CR>')
vim.keymap.set('n', '<leader>se', ':rightbelow new<CR>')

-- Buffer navigation
vim.keymap.set('n', '<leader>b', ':buffers<CR>')

-- Git diff functions
vim.cmd([[
function! DiffVersion(source)
  let filetype = &filetype
  vnew
  execute 'r !git show '. a:source . ':#'
  let &filetype=filetype
  set buftype=nowrite
  windo diffthis
endfunction

function! DiffHead()
  call DiffVersion("HEAD")
endfunction

function! DiffVersionInteractive()
  let source  = input("Diff against: ")
  call DiffVersion(source)
endfunction
]])

vim.keymap.set('n', '<leader>dh', ':call DiffHead()<CR>')
vim.keymap.set('n', '<leader>dv', ':call DiffVersionInteractive()<CR>')
vim.keymap.set('n', '<leader>do', ':diffoff<CR>')
vim.keymap.set('n', '<leader>dt', ':diffthis<CR>')
vim.keymap.set('n', '<leader>dw', ':windo diffthis<CR>')
vim.keymap.set('n', '<leader>dc', ':windo diffoff<CR>:q<CR>')
vim.keymap.set('n', '<leader>du', ':diffupdate<CR>')

-- FZF commands setup
vim.cmd([[
command! -bang -nargs=* Ag
  \ call fzf#vim#grep(
  \   'rg --ignore .git --ignore node_modules --ignore "*.swp" --ignore "*.pyc" --color -- '.shellescape(<q-args>), 1,
  \   fzf#vim#with_preview({'dir': systemlist('git rev-parse --show-toplevel')[0]}), <bang>0)

command! -bang -nargs=? -complete=dir Files
  \ call fzf#vim#files(<q-args>, {'options': ['--info=inline', '--preview', '~/.vim/plugged/fzf.vim/bin/preview.sh {}'], 'dir': systemlist('git rev-parse --show-toplevel')[0]},<bang>0)
]])

-- FZF keymaps
vim.keymap.set('n', '<C-P>', ':Files<CR>')
vim.keymap.set('n', '<leader><C-P>', ':Buffers<CR>')

-- Automatically reload files when changed externally
vim.api.nvim_create_autocmd({"FocusGained", "BufEnter", "CursorHold", "CursorHoldI"}, {
  pattern = "*",
  command = "if mode() != 'c' | checktime | endif",
})

-- Load lazy.nvim and plugins
require('config.lazy')
require('config.dimming')
require('config.colorscheme')