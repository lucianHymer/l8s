-- Dim Background on Focus Lost
local original_colors = {}

-- Save original colors on startup
vim.api.nvim_create_autocmd("ColorScheme", {
  pattern = "*",
  callback = function()
    -- Save key highlight groups
    original_colors.Normal = vim.api.nvim_get_hl(0, {name = "Normal"})
    original_colors.Comment = vim.api.nvim_get_hl(0, {name = "Comment"}) 
    original_colors.Keyword = vim.api.nvim_get_hl(0, {name = "Keyword"})
    original_colors.String = vim.api.nvim_get_hl(0, {name = "String"})
    original_colors.Function = vim.api.nvim_get_hl(0, {name = "Function"})
    original_colors.Identifier = vim.api.nvim_get_hl(0, {name = "Identifier"})
    original_colors.Type = vim.api.nvim_get_hl(0, {name = "Type"})
    original_colors.Special = vim.api.nvim_get_hl(0, {name = "Special"})
  end
})

-- Function to dim foreground colors
local function dim_fg_color(color, factor)
  if not color then return nil end
  local r = math.floor(color / 65536) % 256
  local g = math.floor(color / 256) % 256  
  local b = color % 256
  
  r = math.floor(r * factor)
  g = math.floor(g * factor)  
  b = math.floor(b * factor)
  
  return r * 65536 + g * 256 + b
end

-- Dim on focus lost
vim.api.nvim_create_autocmd("FocusLost", {
  pattern = "*",
  callback = function()
    for group_name, original in pairs(original_colors) do
      if original.fg then
        local dimmed_fg = dim_fg_color(original.fg, 0.5)
        vim.api.nvim_set_hl(0, group_name, {fg = dimmed_fg, bg = original.bg})
      end
    end
  end
})

-- Restore on focus gained
vim.api.nvim_create_autocmd("FocusGained", {
  pattern = "*",
  callback = function()
    for group_name, original in pairs(original_colors) do
      vim.api.nvim_set_hl(0, group_name, original)
    end
  end
})