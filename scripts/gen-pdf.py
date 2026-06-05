import markdown
import weasyprint
import os

HERMES_VENV = os.path.expanduser("~/.hermes/hermes-agent/venv")
FONT_DIR = os.path.join(HERMES_VENV, "lib/python3.12/site-packages/weasyprint/text/fonts")

md_path = os.path.expanduser("~/hermix/PRD.md")
pdf_path = os.path.expanduser("~/hermix/Hermix-PRD.pdf")

with open(md_path, "r", encoding="utf-8") as f:
    md_content = f.read()

# Convert Markdown to HTML with tables support
html_body = markdown.markdown(md_content, extensions=["tables", "fenced_code", "codehilite", "toc"])

# Full HTML with Hermes CN brand styling
html_full = f"""<!DOCTYPE html>
<html lang="zh-CN">
<head>
<meta charset="utf-8">
<style>
  @page {{
    size: A4;
    margin: 1.2cm 1.5cm;
    @bottom-center {{
      content: counter(page) " / " counter(pages);
      font-size: 8pt;
      color: #607D8B;
      font-family: 'PingFang SC', 'Microsoft YaHei', 'Noto Color Emoji', sans-serif;
    }}
  }}
  * {{ box-sizing: border-box; }}
  html {{
    background: #0d1a1a;
    margin: 0;
    padding: 0;
  }}
  body {{
    font-family: 'PingFang SC', 'Microsoft YaHei', 'Noto Color Emoji', 'Inter', sans-serif;
    font-size: 10pt;
    line-height: 1.65;
    color: #E0E0E0;
    background: #0d1a1a;
    padding: 0;
    margin: 0;
  }}
  .page {{
    max-width: 100%;
  }}
  h1 {{
    font-size: 22pt;
    color: #FFD700;
    border-bottom: 2px solid #FFD700;
    padding-bottom: 8px;
    margin-top: 30px;
  }}
  h2 {{
    font-size: 16pt;
    color: #FFD700;
    border-bottom: 1px solid #2a4040;
    padding-bottom: 4px;
    margin-top: 24px;
  }}
  h3 {{
    font-size: 13pt;
    color: #FFFFFF;
    margin-top: 18px;
  }}
  h4 {{
    font-size: 11pt;
    color: #B0BEC5;
    margin-top: 14px;
  }}
  p {{ margin: 6px 0; }}
  a {{ color: #4FC3F7; text-decoration: none; }}
  blockquote {{
    border-left: 4px solid #FFD700;
    padding-left: 14px;
    margin: 12px 0;
    color: #B0BEC5;
    background: #162626;
    padding: 10px 14px;
    border-radius: 4px;
  }}
  code {{
    font-family: 'JetBrains Mono', 'Fira Code', monospace;
    font-size: 9pt;
    background: #1a3030;
    padding: 1px 5px;
    border-radius: 3px;
    color: #E0E0E0;
  }}
  pre {{
    background: #162626;
    border: 1px solid #2a4040;
    border-radius: 6px;
    padding: 12px 16px;
    overflow-x: auto;
    font-size: 8.5pt;
    line-height: 1.5;
  }}
  pre code {{
    background: none;
    padding: 0;
  }}
  table {{
    width: 100%;
    border-collapse: collapse;
    margin: 12px 0;
    font-size: 9.5pt;
  }}
  th, td {{
    border: 1px solid #2a4040;
    padding: 6px 10px;
    text-align: left;
  }}
  th {{
    background: #162626;
    color: #FFD700;
    font-weight: 600;
  }}
  td {{
    background: #0d1a1a;
  }}
  tr:nth-child(even) td {{
    background: #162626;
  }}
  hr {{
    border: none;
    border-top: 1px solid #2a4040;
    margin: 24px 0;
  }}
  ul, ol {{ margin: 6px 0; padding-left: 20px; }}
  li {{ margin: 3px 0; }}
  strong {{ color: #FFD700; }}
  em {{ color: #4FC3F7; font-style: normal; }}
  .header-meta {{
    color: #607D8B;
    font-size: 9pt;
    margin-top: -8px;
    margin-bottom: 20px;
  }}
  .cover {{
    text-align: center;
    padding: 60px 0 40px 0;
  }}
  .cover h1 {{
    font-size: 28pt;
    border: none;
    margin-bottom: 8px;
  }}
  .cover .subtitle {{
    font-size: 14pt;
    color: #B0BEC5;
    margin-bottom: 20px;
  }}
  .cover .meta {{
    font-size: 10pt;
    color: #607D8B;
  }}
  .cover .line {{
    width: 60px;
    border-top: 2px solid #FFD700;
    margin: 20px auto;
  }}
  .toc {{
    background: #162626;
    border: 1px solid #2a4040;
    border-radius: 6px;
    padding: 16px 20px;
    margin: 20px 0;
  }}
  .toc h2 {{ margin-top: 0; border: none; }}
  .toc ul {{ list-style: none; padding-left: 0; }}
  .toc li {{ margin: 4px 0; }}
  .toc a {{ color: #B0BEC5; }}
</style>
</head>
<body>
<div class="cover">
  <h1>Hermix</h1>
  <div class="line"></div>
  <div class="subtitle">Hermes 中文社区混合论坛 — 产品需求文档</div>
  <div class="meta">
    项目代号：Hermix (Hermes + Mix)<br>
    基于 NodeBB v4.12.0<br>
    v0.1.0 — 2026-06
  </div>
</div>

{html_body}

</body>
</html>"""

# Generate PDF
weasyprint.HTML(string=html_full).write_pdf(pdf_path)

print(f"PDF generated: {pdf_path}")
print(f"Size: {os.path.getsize(pdf_path) / 1024:.1f} KB")
