-- Set up line number colors
vim.api.nvim_create_autocmd("ColorScheme", {
  pattern = "*",
  callback = function()
    vim.api.nvim_set_hl(0, "LineNrAbove", { link = "Special" })
    vim.api.nvim_set_hl(0, "LineNr", { link = "String" })
    vim.api.nvim_set_hl(0, "LineNrBelow", { link = "Identifier" })
  end,
})

-- Set colorscheme with gruvbox hard contrast
vim.g.gruvbox_contrast_dark = 'hard'
vim.g.gruvbox_italic = 1
vim.cmd.colorscheme('gruvbox')