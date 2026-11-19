-- Transforme les alertes GitHub (> [!NOTE], > [!TIP]…) en encadrés tcolorbox
-- avec un titre en français, pour le rendu PDF.

local kinds = {
  note      = { title = "Note",      color = "00ADD8" },
  tip       = { title = "Astuce",    color = "3C9F53" },
  important = { title = "Important", color = "7B4FC9" },
  warning   = { title = "Attention", color = "E0A100" },
  caution   = { title = "Prudence",  color = "D1453B" },
}

function Meta(meta)
  local header = pandoc.RawBlock("latex", [[
\usepackage[most]{tcolorbox}
\newtcolorbox{callout}[2]{enhanced, breakable, colback=#1!6, colframe=#1,
  boxrule=0pt, leftrule=3pt, arc=0pt, outer arc=0pt, left=8pt, right=8pt,
  top=6pt, bottom=6pt, fonttitle=\bfseries\color{#1}, coltitle=#1,
  colbacktitle=#1!6, title={#2}, attach title to upper={\par\smallskip}}
]])
  local includes = meta["header-includes"]
  if includes == nil then
    includes = pandoc.MetaList({})
  elseif includes.t ~= "MetaList" then
    includes = pandoc.MetaList({ includes })
  end
  includes:insert(pandoc.MetaBlocks({ header }))
  meta["header-includes"] = includes
  return meta
end

function Div(el)
  local kind = kinds[el.classes[1]]
  if not kind or not FORMAT:match("latex") then
    return nil
  end
  local content = el.content
  if #content > 0 and content[1].t == "Div" and content[1].classes[1] == "title" then
    content = pandoc.List({ table.unpack(content, 2) })
  end
  local open = string.format("\\definecolor{callout-%s}{HTML}{%s}\\begin{callout}{callout-%s}{%s}",
    el.classes[1], kind.color, el.classes[1], kind.title)
  local blocks = pandoc.List({ pandoc.RawBlock("latex", open) })
  blocks:extend(content)
  blocks:insert(pandoc.RawBlock("latex", "\\end{callout}"))
  return blocks
end

return {
  { Meta = Meta },
  { Div = Div },
}
