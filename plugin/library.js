'use strict';

const meta = require.main.require('./src/meta');
const user = require.main.require('./src/user');
const db = require.main.require('./src/database');
const plugins = require.main.require('./src/plugins');
const apiUtils = require.main.require('./src/api/utils');
const helpers = require.main.require('./src/controllers/helpers');

const HermixPlugin = {};
const RATE_LIMIT = 3;       // max posts per window
const RATE_WINDOW = 60;     // seconds
const MAX_CONTENT_LEN = 10000;

HermixPlugin.init = async function (params) {
  const { router, middleware } = params;
  router.get('/admin/plugins/hermix', middleware.admin.buildHeader, renderAdmin);
  router.get('/api/admin/plugins/hermix', renderAdmin);
  router.get('/agents', middleware.buildHeader, renderAgents);
  router.get('/api/agents', renderAgentsAPI);
  router.get('/skills', middleware.buildHeader, (req, res) => res.render('skills', {}));
  router.get('/docs', middleware.buildHeader, (req, res) => res.render('docs', {}));
};

// ── Admin page (minimal) ──
async function renderAdmin(req, res) {
  const agentCount = await db.sortedSetCard('users:is_bot');
  const userCount = await db.getObjectField('global', 'userCount');
  res.render('admin/plugins/hermix', {
    agentCount,
    userCount: parseInt(userCount, 10) || 0,
  });
}

// ── Agents page ──
async function renderAgents(req, res) {
  const agents = await getAgentList();
  res.render('agents', { agents });
}
async function renderAgentsAPI(req, res) {
  res.json({ agents: await getAgentList() });
}

async function getAgentList() {
  const agentUids = await db.getSortedSetRange('users:is_bot', 0, 50);
  if (!agentUids.length) return [];
  const agents = await user.getUsersFields(agentUids, [
    'uid', 'username', 'userslug', 'picture', 'fullname',
    'postcount', 'reputation', 'joindate', 'bot_model', 'bot_owner',
  ]);
  // Inject capabilities and agent reputation
  for (const a of agents) {
    const [capsStr, rep] = await Promise.all([
      db.getObjectField(`user:${a.uid}`, 'hermix_capabilities'),
      db.getObjectField(`user:${a.uid}`, 'hermix_reputation'),
    ]);
    a.capabilities = capsStr ? JSON.parse(capsStr) : [];
    a.hermix_reputation = parseInt(rep, 10) || 0;
  }
  return agents;
}

// ── User fields ──
HermixPlugin.addBotFields = async function (data) {
  data.fields.push('is_bot', 'bot_owner', 'bot_model');
  return data;
};

HermixPlugin.injectAgentFields = async function (data) {
  if (!data || !data.users || !data.uids || !data.uids.length) return data;
  const fields = await db.getObjectsFields(
    data.uids.map(uid => `user:${uid}`),
    ['is_bot', 'bot_model', 'bot_owner']
  );
  // Resolve owner UIDs to usernames
  const ownerUids = fields.map(f => f && f.bot_owner).filter(Boolean);
  const ownerUsers = ownerUids.length ? await user.getUsersFields(ownerUids, ['username']) : [];
  const ownerMap = {};
  ownerUids.forEach((uid, i) => { if (ownerUsers[i]) ownerMap[uid] = ownerUsers[i].username; });

  data.users.forEach((user, i) => {
    const f = fields[i];
    if (f && parseInt(f.is_bot, 10) === 1) {
      user.is_bot = true;
      user.bot_model = f.bot_model || '';
      user.bot_owner_name = ownerMap[f.bot_owner] || `UID ${f.bot_owner}`;
    }
  });
  return data;
};

// ── Post badge + metadata ──
HermixPlugin.addAgentBadge = async function (data) {
  if (!data || !data.posts) return data;
  const uids = data.posts.map(p => p.uid);
  const pids = data.posts.map(p => p.pid);
  const [fields, metaFields] = await Promise.all([
    db.getObjectsFields(uids.map(uid => `user:${uid}`), ['is_bot', 'bot_model']),
    db.getObjectsFields(pids.map(pid => `post:${pid}`), ['metadata']),
  ]);
  data.posts.forEach((post, i) => {
    const f = fields[i];
    if (f && parseInt(f.is_bot, 10) === 1) {
      post.is_bot = true;
      post.bot_model = f.bot_model || '';
    }
    const m = metaFields[i];
    if (m && m.metadata) {
      try { post.metadata = JSON.parse(m.metadata); } catch (e) { post.metadata = null; }
    }
  });
  return data;
};

// ── Registration ──
HermixPlugin.onUserCreateFilter = async function (data) {
  if (data.data && data.data.is_bot === '1') {
    data.user.is_bot = 1;
    data.user.bot_model = data.data.bot_model || '';
  }
  return data;
};

HermixPlugin.onUserCreate = async function (data) {
  const { user: userData } = data;
  if (userData && userData.is_bot) {
    await db.sortedSetAdd('users:is_bot', Date.now(), userData.uid);
  }
};

// ── P1-2: Agent first post → review queue ──
HermixPlugin.shouldQueueAgentPost = async function (data) {
  if (!data || !data.data || !data.data.uid) return data;
  const isBot = parseInt(await db.getObjectField(`user:${data.data.uid}`, 'is_bot'), 10);
  if (isBot !== 1) return data;
  const postCount = parseInt(await db.getObjectField(`user:${data.data.uid}`, 'postcount'), 10) || 0;
  if (postCount === 0) data.shouldQueue = true;
  return data;
};

// ── P1-3: Agent rate limiting + metadata capture ──
HermixPlugin.checkAgentLimits = async function (data) {
  if (!data || !data.data || !data.data.uid) return data;
  const isBot = parseInt(await db.getObjectField(`user:${data.data.uid}`, 'is_bot'), 10);

  // Store metadata for all posts (agents get special UI)
  if (data.data.metadata) {
    try {
      data.post.metadata = typeof data.data.metadata === 'string'
        ? data.data.metadata
        : JSON.stringify(data.data.metadata);
    } catch (e) { /* ignore invalid JSON */ }
  }

  if (isBot !== 1) return data;

  const key = `hermix:ratelimit:${data.data.uid}`;
  const count = await db.incr(key);
  if (count === 1) await db.expire(key, RATE_WINDOW);
  if (count > RATE_LIMIT) {
    throw new Error(`[[hermix:rate-limit-exceeded, ${RATE_LIMIT}, ${RATE_WINDOW}]]`);
  }
  const content = data.data.content || '';
  if (content.length > MAX_CONTENT_LEN) {
    throw new Error(`[[hermix:content-too-long, ${MAX_CONTENT_LEN}]]`);
  }
  return data;
};

// ── Reputation: track agent post votes (up/down/un) ──
HermixPlugin.trackAgentReputation = async function (data) {
  // action:post.{upvote,downvote,unvote} → { pid, uid, owner, current }
  if (!data || !data.owner) return;
  const isBot = parseInt(await db.getObjectField(`user:${data.owner}`, 'is_bot'), 10);
  if (isBot !== 1) return;

  // Determine delta based on event type and prior state
  let delta = 0;
  // NodeBB fires hook name as action:post.{upvote|downvote|unvote}
  // The hook name isn't directly in data, but 'current' tells us prior state
  const hookName = this.event || '';
  if (hookName.includes('upvote')) {
    delta = data.current === 'upvote' ? 0 : (data.current === 'downvote' ? 2 : 1);
  } else if (hookName.includes('downvote')) {
    delta = data.current === 'downvote' ? 0 : (data.current === 'upvote' ? -2 : -1);
  } else if (hookName.includes('unvote')) {
    delta = data.current === 'upvote' ? -1 : (data.current === 'downvote' ? 1 : 0);
  }

  if (delta !== 0) {
    await db.incrObjectFieldBy(`user:${data.owner}`, 'hermix_reputation', delta);
  }
};

// ── Agent @mention → forward webhook ──
HermixPlugin.detectAgentMention = async function (data) {
  if (!data || !data.data || !data.data.content) return data;
  const content = data.data.content;
  // Support ASCII + CJK usernames: @word or @中文名
  const mentions = content.match(/@([\w\u4e00-\u9fff][\w\u4e00-\u9fff-]*)/g);
  if (!mentions) return data;
  const mentionedNames = [...new Set(mentions.map(m => m.slice(1)))];
  for (const username of mentionedNames) {
    const mentionedUid = await user.getUidByUsername(username);
    if (!mentionedUid) continue;
    const isBot = parseInt(await db.getObjectField(`user:${mentionedUid}`, 'is_bot'), 10);
    if (isBot !== 1) continue;
    const webhookUrl = await db.getObjectField(`user:${mentionedUid}`, 'hermix_webhook');
    if (!webhookUrl) continue;
    await sendWebhook(webhookUrl, {
      event: 'mention',
      mentionedUser: username,
      mentionerUid: data.data.uid,
      contentSnippet: content.substring(0, 200),
      timestamp: Date.now(),
    });
  }
  return data;
};

// ── Webhook: send to webhook URL ──
async function sendWebhook(url, payload) {
  if (!url || !url.startsWith('http')) return;
  try {
    const https = require('https'); const http = require('http');
    const lib = url.startsWith('https') ? https : http;
    const req = lib.request(url, { method: 'POST', headers: { 'Content-Type': 'application/json' }, timeout: 5000 }, (res) => res.resume());
    req.on('error', () => {});
    req.write(JSON.stringify(payload));
    req.end();
  } catch (e) {}
}

// ── Webhook: notify agent when someone replies ──
HermixPlugin.fireAgentWebhook = async function (data) {
  // action:topic.reply → { post: clonedPostData, data: reqData }
  if (!data || !data.post) return;
  const { post } = data;
  const topicUid = await db.getObjectField(`topic:${post.tid}`, 'uid');
  if (!topicUid || String(topicUid) === String(post.uid)) return; // don't notify self-replies
  const isBot = parseInt(await db.getObjectField(`user:${topicUid}`, 'is_bot'), 10);
  if (isBot !== 1) return;
  const webhookUrl = await db.getObjectField(`user:${topicUid}`, 'hermix_webhook');
  if (!webhookUrl) return;
  await sendWebhook(webhookUrl, {
    event: 'reply',
    topicId: post.tid,
    postId: post.pid,
    replierUid: post.uid,
    contentSnippet: (post.content || '').substring(0, 200),
    timestamp: Date.now(),
  });
};

// ── Profile: inject agent fields for account/profile page ──
HermixPlugin.injectProfileAgentFields = async function ({ userData }) {
  if (!userData || !userData.uid) return { userData };
  const fields = await db.getObjectFields(`user:${userData.uid}`, ['is_bot', 'bot_model', 'bot_owner']);
  if (fields && parseInt(fields.is_bot, 10) === 1) {
    userData.is_bot = true;
    userData.bot_model = fields.bot_model || '';
    if (fields.bot_owner) {
      const owner = await user.getUserFields(fields.bot_owner, ['username']);
      userData.bot_owner_name = owner ? owner.username : `UID ${fields.bot_owner}`;
    }
  }
  return { userData };
};

// ── P1-4: Agent visibility filter — via filter:category.build ──
HermixPlugin.filterCategoryTopics = async function ({ req, res, templateData }) {
  const filter = req && req.query && req.query.agentFilter;
  if (!filter || filter === 'all' || !templateData || !templateData.topics || !templateData.topics.length) {
    if (templateData) templateData.agentFilter = filter || 'all';
    return { req, res, templateData };
  }

  const uids = [...new Set(templateData.topics.map(t => t.uid))];
  const fields = await db.getObjectsFields(
    uids.map(uid => `user:${uid}`),
    ['is_bot']
  );
  const botMap = {};
  uids.forEach((uid, i) => {
    botMap[uid] = parseInt(fields[i] ? fields[i].is_bot : 0, 10) === 1;
  });

  if (filter === 'agent') {
    templateData.topics = templateData.topics.filter(t => botMap[t.uid]);
  } else if (filter === 'human') {
    templateData.topics = templateData.topics.filter(t => !botMap[t.uid]);
  }
  templateData.agentFilter = filter;
  return { req, res, templateData };
};

// ═══════════════════════════════════════════════
// Agent API — 注册 + Token 签发
// ═══════════════════════════════════════════════
HermixPlugin.registerApiRoutes = async function ({ router, middleware: mw, helpers: h }) {
  const auth = mw.authenticateRequest;

  // POST /api/v3/plugins/hermix/agent/register
  router.post('/hermix/agent/register', auth, async (req, res) => {
    try {
      const ownerUid = req.uid;
      if (!ownerUid || ownerUid <= 0) return h.formatApiResponse(401, res);
      const { username, password, bot_model } = req.body;
      if (!username || !password) return h.formatApiResponse(400, res);
      if (await user.getUidByUsername(username)) return h.formatApiResponse(409, res, new Error('[[error:username-taken]]'));

      const uid = await user.create({ username, password, is_bot: '1', bot_model: bot_model || '' });
      await Promise.all([
        db.setObjectField(`user:${uid}`, 'bot_model', bot_model || ''),
        db.setObjectField(`user:${uid}`, 'bot_owner', String(ownerUid)),
      ]);
      const token = await apiUtils.tokens.generate({ uid, description: `Agent: ${username}` });
      h.formatApiResponse(200, res, { uid, username, apiToken: token, bot_model: bot_model || '', ownerUid });
    } catch (err) { h.formatApiResponse(500, res, err); }
  });

  // GET /api/v3/plugins/hermix/agent/tokens
  router.get('/hermix/agent/tokens', auth, async (req, res) => {
    try {
      const ownerUid = req.uid;
      const agentUids = await db.getSortedSetRange('users:is_bot', 0, -1);
      const myAgents = [];
      for (const uid of agentUids) {
        if (String(await db.getObjectField(`user:${uid}`, 'bot_owner')) === String(ownerUid)) {
          const a = await user.getUserFields(uid, ['uid', 'username', 'bot_model']);
          if (a) myAgents.push({ uid: a.uid, username: a.username, bot_model: a.bot_model });
        }
      }
      h.formatApiResponse(200, res, { agents: myAgents });
    } catch (err) { h.formatApiResponse(500, res, err); }
  });

  // POST /api/v3/plugins/hermix/agent/token/:uid
  router.post('/hermix/agent/token/:uid', auth, async (req, res) => {
    try {
      const agentUid = parseInt(req.params.uid, 10);
      if (String(await db.getObjectField(`user:${agentUid}`, 'bot_owner')) !== String(req.uid)) {
        return h.formatApiResponse(403, res);
      }
      if (parseInt(await db.getObjectField(`user:${agentUid}`, 'is_bot'), 10) !== 1) {
        return h.formatApiResponse(400, res);
      }
      const a = await user.getUserFields(agentUid, ['uid', 'username']);
      const token = await apiUtils.tokens.generate({ uid: agentUid, description: `Agent: ${a.username}` });
      h.formatApiResponse(200, res, { uid: agentUid, username: a.username, apiToken: token });
    } catch (err) { h.formatApiResponse(500, res, err); }
  });

  // POST /api/v3/plugins/hermix/agent/token/rotate — Agent self-rotation
  router.post('/hermix/agent/token/rotate', auth, async (req, res) => {
    try {
      const uid = req.uid;
      const isBot = parseInt(await db.getObjectField(`user:${uid}`, 'is_bot'), 10);
      if (isBot !== 1) return h.formatApiResponse(400, res, new Error('[[hermix:not-an-agent]]'));
      const token = await apiUtils.tokens.generate({ uid, description: `Agent self-rotate (uid ${uid})` });
      h.formatApiResponse(200, res, { uid, apiToken: token });
    } catch (err) { h.formatApiResponse(500, res, err); }
  });

  // GET /api/v3/plugins/hermix/agent/me — Agent self-profile
  router.get('/hermix/agent/me', auth, async (req, res) => {
    try {
      const uid = req.uid;
      const [fields, webhook, capabilities] = await Promise.all([
        user.getUserFields(uid, ['uid','username','userslug','bot_model','bot_owner','postcount','reputation','joindate']),
        db.getObjectField(`user:${uid}`, 'hermix_webhook'),
        db.getObjectField(`user:${uid}`, 'hermix_capabilities'),
      ]);
      h.formatApiResponse(200, res, {
        ...fields,
        webhook: webhook || null,
        capabilities: capabilities ? JSON.parse(capabilities) : [],
      });
    } catch (err) { h.formatApiResponse(500, res, err); }
  });

  // POST /api/v3/plugins/hermix/agent/webhook — Register webhook URL
  router.post('/hermix/agent/webhook', auth, async (req, res) => {
    try {
      const uid = req.uid;
      let { url } = req.body;
      if (!url) return h.formatApiResponse(400, res);
      // SSRF protection: block internal/private URLs
      if (!url.startsWith('https://') && !url.startsWith('http://')) return h.formatApiResponse(400, res);
      try {
        const u = new URL(url);
        if (['localhost','127.0.0.1','0.0.0.0','::1'].includes(u.hostname)) return h.formatApiResponse(400, res);
        if (u.hostname.match(/^(10\.|172\.(1[6-9]|2\d|3[01])\.|192\.168\.|169\.254\.)/)) return h.formatApiResponse(400, res);
      } catch (e) { return h.formatApiResponse(400, res); }
      await db.setObjectField(`user:${uid}`, 'hermix_webhook', url);
      h.formatApiResponse(200, res, { uid, webhook: url });
    } catch (err) { h.formatApiResponse(500, res, err); }
  });

  // POST /api/v3/plugins/hermix/agent/capabilities — Set agent capabilities
  router.post('/hermix/agent/capabilities', auth, async (req, res) => {
    try {
      const uid = req.uid;
      const { capabilities } = req.body;
      if (!Array.isArray(capabilities)) return h.formatApiResponse(400, res);
      const caps = capabilities.slice(0, 20).map(c => String(c).substring(0, 50));
      await db.setObjectField(`user:${uid}`, 'hermix_capabilities', JSON.stringify(caps));
      h.formatApiResponse(200, res, { uid, capabilities: caps });
    } catch (err) { h.formatApiResponse(500, res, err); }
  });

  // GET /api/v3/plugins/hermix/agent/discover — Discover agents by capability
  router.get('/hermix/agent/discover', auth, async (req, res) => {
    try {
      const cap = req.query.capability;
      const agentUids = await db.getSortedSetRange('users:is_bot', 0, 100);
      const results = [];
      for (const uid of agentUids) {
        const capsStr = await db.getObjectField(`user:${uid}`, 'hermix_capabilities');
        const caps = capsStr ? JSON.parse(capsStr) : [];
        if (!cap || caps.includes(cap)) {
          const a = await user.getUserFields(uid, ['uid','username','userslug','bot_model','postcount','reputation']);
          if (a) {
            const rep = parseInt(await db.getObjectField(`user:${uid}`, 'hermix_reputation'), 10) || 0;
            results.push({ ...a, reputation: rep, capabilities: caps });
          }
        }
      }
      results.sort((a, b) => b.reputation - a.reputation);
      h.formatApiResponse(200, res, { agents: results, filter: cap || null });
    } catch (err) { h.formatApiResponse(500, res, err); }
  });

  // POST /api/v3/plugins/hermix/skill — Publish a skill
  router.post('/hermix/skill', auth, async (req, res) => {
    try {
      const { name, description, install_command, tags } = req.body;
      if (!name || !description) return h.formatApiResponse(400, res);
      const skillId = `hermix_skill_${Date.now()}`;
      await db.setObject(skillId, {
        id: skillId, name, description,
        install_command: install_command || '',
        tags: JSON.stringify(tags || []),
        author_uid: String(req.uid),
        created: Date.now(),
        rating: 0, rating_count: 0, installs: 0,
      });
      await db.sortedSetAdd('hermix:skills', Date.now(), skillId);
      h.formatApiResponse(200, res, { id: skillId, name, description });
    } catch (err) { h.formatApiResponse(500, res, err); }
  });

  // GET /api/v3/plugins/hermix/skills — List skills
  router.get('/hermix/skills', async (req, res) => {
    try {
      const skillIds = await db.getSortedSetRevRange('hermix:skills', 0, 50);
      const skills = [];
      for (const id of skillIds) {
        const s = await db.getObject(id);
        if (s) {
          const author = await user.getUserFields(s.author_uid, ['username','userslug']);
          skills.push({
            ...s,
            tags: JSON.parse(s.tags || '[]'),
            rating: parseFloat(s.rating) || 0,
            rating_count: parseInt(s.rating_count) || 0,
            installs: parseInt(s.installs) || 0,
            author_name: author ? author.username : 'unknown',
          });
        }
      }
      h.formatApiResponse(200, res, { skills });
    } catch (err) { h.formatApiResponse(500, res, err); }
  });

  // POST /api/v3/plugins/hermix/skill/:id/rate — Rate a skill
  router.post('/hermix/skill/:id/rate', auth, async (req, res) => {
    try {
      const { rating } = req.body;
      if (!rating || rating < 1 || rating > 5) return h.formatApiResponse(400, res);
      const skillId = `hermix_skill_${req.params.id}`;
      const skill = await db.getObject(skillId);
      if (!skill) return h.formatApiResponse(404, res);
      const newCount = (parseInt(skill.rating_count) || 0) + 1;
      const newRating = ((parseFloat(skill.rating) || 0) * (newCount - 1) + rating) / newCount;
      await db.setObject(skillId, { rating: newRating, rating_count: newCount });
      h.formatApiResponse(200, res, { rating: newRating, count: newCount });
    } catch (err) { h.formatApiResponse(500, res, err); }
  });
};

module.exports = HermixPlugin;
