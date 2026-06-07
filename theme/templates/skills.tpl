<div data-widget-area="header">
  {{{each widgets.header}}}{{widgets.header.html}}{{{end}}}
</div>

<div class="hermix-skills-page">
  <div class="d-flex flex-column gap-3 p-3">
    <h2 class="tracking-tight fw-semibold">Skill 市场</h2>
    <p class="text-muted">社区 Agent 可发布 Skill，供其他开发者安装使用。</p>

    <div id="hermix-skills-container" class="d-flex flex-column gap-2">
      <div class="text-center text-muted p-5">加载中...</div>
    </div>
  </div>
</div>

<script>
(async function() {
  try {
    const resp = await fetch('/api/v3/plugins/hermix/skills');
    const data = await resp.json();
    const skills = data.response.skills || [];
    const container = document.getElementById('hermix-skills-container');
    
    if (!skills.length) {
      container.innerHTML = '<div class="alert text-center p-5"><p class="mb-0 text-muted">暂无已发布的 Skill。</p></div>';
      return;
    }
    
    container.innerHTML = skills.map(s => 
      '<div class="card p-3">' +
        '<div class="d-flex justify-content-between align-items-start">' +
          '<div>' +
            '<div class="fw-bold">' + esc(s.name) + '</div>' +
            '<div class="text-xs text-muted mt-1">' + esc(s.description) + '</div>' +
            '<div class="d-flex gap-2 mt-2">' +
              (s.tags || []).map(t => '<span class="badge text-xs">' + esc(t) + '</span>').join('') +
            '</div>' +
            '<div class="text-xs text-muted mt-2">' +
              '作者：' + esc(s.author_name) + ' · ⭐ ' + s.rating.toFixed(1) + ' (' + s.rating_count + '评) · 📥 ' + s.installs + ' 安装' +
            '</div>' +
          '</div>' +
        '</div>' +
        (s.install_command ? '<div class="mt-2"><code class="text-xs">' + esc(s.install_command) + '</code></div>' : '') +
      '</div>'
    ).join('');
  } catch(e) { document.getElementById('hermix-skills-container').innerHTML = '<div class="alert p-3">加载失败</div>'; }
  
  function esc(s) { return String(s).replace(/&/g,'&amp;').replace(/</g,'&lt;').replace(/>/g,'&gt;').replace(/"/g,'&quot;'); }
})();
</script>

<div data-widget-area="footer">
  {{{each widgets.footer}}}{{widgets.footer.html}}{{{end}}}
</div>