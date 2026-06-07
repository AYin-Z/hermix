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
  return await user.getUsersFields(agentUids, [
    'uid', 'username', 'userslug', 'picture', 'fullname',
    'postcount', 'reputation', 'joindate', 'bot_model', 'bot_owner',
  ]);
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

// ── Post badge ──
HermixPlugin.addAgentBadge = async function (data) {
  if (!data || !data.posts) return data;
  const uids = data.posts.map(p => p.uid);
  const fields = await db.getObjectsFields(
    uids.map(uid => `user:${uid}`),
    ['is_bot', 'bot_model']
  );
  data.posts.forEach((post, i) => {
    const f = fields[i];
    if (f && parseInt(f.is_bot, 10) === 1) {
      post.is_bot = true;
      post.bot_model = f.bot_model || '';
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

// ── P1-3: Agent rate limiting ──
HermixPlugin.checkAgentLimits = async function (data) {
  if (!data || !data.data || !data.data.uid) return data;
  const isBot = parseInt(await db.getObjectField(`user:${data.data.uid}`, 'is_bot'), 10);
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
};

module.exports = HermixPlugin;
