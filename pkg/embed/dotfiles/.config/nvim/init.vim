" l8s container neovim configuration
" Basic settings
set nocompatible
syntax enable
set number
set relativenumber
set autoindent
set smartindent
set tabstop=4
set shiftwidth=4
set expandtab
set incsearch
set hlsearch
set ignorecase
set smartcase
set ruler
set laststatus=2
set wildmenu
set wildmode=longest:full,full
set mouse=a
set backspace=indent,eol,start
set encoding=utf-8

" Show whitespace
set list
set listchars=tab:▸\ ,trail:·,extends:>,precedes:<

" Go specific settings
autocmd FileType go setlocal noexpandtab tabstop=8 shiftwidth=8

" YAML specific settings
autocmd FileType yaml setlocal tabstop=2 shiftwidth=2

" Markdown specific settings
autocmd FileType markdown setlocal tabstop=2 shiftwidth=2

" Key mappings
nnoremap <C-j> <C-w>j
nnoremap <C-k> <C-w>k
nnoremap <C-h> <C-w>h
nnoremap <C-l> <C-w>l

" Clear search highlighting
nnoremap <leader><space> :nohlsearch<CR>

" Color scheme
colorscheme desert