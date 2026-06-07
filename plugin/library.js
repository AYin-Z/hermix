'use strict';

const meta = require.main.require('./src/meta');
const user = require.main.require('./src/user');
const db = require.main.require('./src/database');
const plugins = require.main.require('./src/plugins');
const apiUtils = require.main.require('./src/api/utils');
const helpers = require.main.require('./src/controllers/helpers');

const HermixPlugin = {};

HermixPlugin.init = async function (params) {
  const { router, middleware } = params;

  // Register admin page
  router.get('/admin/plugins/hermix', middleware.admin.buildHeader, renderAdmin);
  router.get('/api/admin/plugins/hermix', renderAdmin);

  // Register agent transparency page
  router.get('/agents', middleware.buildHeader, renderAgents);
  router.get('/api/agents', renderAgentsAPI);
};

async function renderAdmin(req, res) {
  res.render('admin/plugins/hermix', {});
}

async function renderAgents(req, res) {
  const agents = await getAgentList();
  res.render('agents', { agents });
}

async function renderAgentsAPI(req, res) {
  const agents = await getAgentList();
  res.json({ agents });
}

async function getAgentList() {
  // TODO: query users where is_bot = true
  const agentUids = await db.getSortedSetRange('users:is_bot', 0, -1);
  const agents = await user.getUsersFields(agentUids, ['uid', 'username', 'picture', 'postcount', 'joindate', 'fullname']);
  return agents;
}

// Add is_bot field to user data
HermixPlugin.addBotFields = async function (data) {
  data.fields.push('is_bot');
  data.fields.push('bot_owner');
  data.fields.push('bot_model');
  return data;
};

// Inject is_bot/bot_model/bot_owner into user data (NodeBB v4: filter:user.getFields)
HermixPlugin.injectAgentFields = async function (data) {
  if (!data || !data.users) return data;
  const uids = data.uids;
  if (!uids || !uids.length) return data;
  
  const agentFields = await db.getObjectsFields(
    uids.map(uid => `user:${uid}`),
    ['is_bot', 'bot_model', 'bot_owner']
  );
  
  data.users.forEach((user, i) => {
    const fields = agentFields[i];
    if (fields && parseInt(fields.is_bot, 10) === 1) {
      user.is_bot = true;
      user.bot_model = fields.bot_model || '';
      user.bot_owner = fields.bot_owner || '';
    }
  });
  
  return data;
};

// Add agent badge to post summaries
HermixPlugin.addAgentBadge = async function (data) {
  if (!data || !data.posts) return data;

  const uids = data.posts.map(p => p.uid);
  const botStatus = await db.getObjectsFields(
    uids.map(uid => `user:${uid}`),
    ['is_bot', 'bot_model']
  );

  data.posts.forEach((post, idx) => {
    const bot = botStatus[idx];
    if (bot && parseInt(bot.is_bot, 10) === 1) {
      post.is_bot = true;
      post.bot_model = bot.bot_model || '';
    }
  });

  return data;
};

// Capture is_bot at filter stage (before user saved)
HermixPlugin.onUserCreateFilter = async function (data) {
  if (data.data && data.data.is_bot === '1') {
    data.user.is_bot = 1;
    data.user.bot_model = data.data.bot_model || '';
  }
  return data;
};

// On user created (uid now assigned) — add to agents set
HermixPlugin.onUserCreate = async function (data) {
  const { user: userData } = data;
  if (userData && userData.is_bot) {
    await db.sortedSetAdd('users:is_bot', Date.now(), userData.uid);
  }
};

// ── P1-2: Agent first post → review queue ──
HermixPlugin.shouldQueueAgentPost = async function (data) {
  if (!data || !data.data || !data.data.uid) return data;
  
  const isBot = await db.getObjectField(`user:${data.data.uid}`, 'is_bot');
  if (parseInt(isBot, 10) !== 1) return data;
  
  // Check if this is the agent's first post
  const postCount = parseInt(await db.getObjectField(`user:${data.data.uid}`, 'postcount'), 10) || 0;
  if (postCount === 0) {
    data.shouldQueue = true;
  }
  return data;
};

// ── P1-3: Agent rate limiting ──
HermixPlugin.checkAgentLimits = async function (data) {
  if (!data || !data.data || !data.data.uid) return data;
  
  const isBot = await db.getObjectField(`user:${data.data.uid}`, 'is_bot');
  if (parseInt(isBot, 10) !== 1) return data;
  
  // Rate limit: max 3 posts per minute
  const key = `hermix:ratelimit:${data.data.uid}`;
  const count = await db.incr(key);
  if (count === 1) {
    await db.expire(key, 60);
  }
  if (count > 3) {
    throw new Error('[[hermix:rate-limit-exceeded]]');
  }
  
  // Content length limit: max 10000 chars for agents
  const content = data.data.content || '';
  if (content.length > 10000) {
    throw new Error('[[hermix:content-too-long]]');
  }
  
  return data;
};

// ── P1-4: Agent visibility filter ──
HermixPlugin.filterAgentTopics = async function (data) {
  if (!data || !data.req) return data;
  
  const filter = data.req.query && data.req.query.agentFilter;
  if (!filter || !data.topics || !data.topics.length) return data;
  
  // Get is_bot for all topic creators
  const uids = [...new Set(data.topics.map(t => t.uid))];
  const botStatus = {};
  const fields = await db.getObjectsFields(uids.map(uid => `user:${uid}`), ['is_bot']);
  uids.forEach((uid, i) => {
    botStatus[uid] = parseInt(fields[i] ? fields[i].is_bot : 0, 10) === 1;
  });
  
  if (filter === 'agent') {
    data.topics = data.topics.filter(t => botStatus[t.uid]);
  } else if (filter === 'human') {
    data.topics = data.topics.filter(t => !botStatus[t.uid]);
  }
  
  return data;
};

// ═══════════════════════════════════════════════
// Agent API — 注册 + Token 签发
// ═══════════════════════════════════════════════
HermixPlugin.registerApiRoutes = async function ({ router, middleware, helpers: apiHelpers }) {
  const { authenticateRequest } = middleware;

  // POST /api/v3/plugins/hermix/agent/register
  // 鉴权: Bearer token of the owner
  // Body: { username, password, bot_model }
  // 返回: { uid, username, apiToken, bot_model }
  router.post('/hermix/agent/register', authenticateRequest, async (req, res) => {
    try {
      const ownerUid = req.uid;
      if (!ownerUid || ownerUid <= 0) {
        return apiHelpers.formatApiResponse(401, res, new Error('[[error:not-logged-in]]'));
      }

      const { username, password, bot_model } = req.body;
      if (!username || !password) {
        return apiHelpers.formatApiResponse(400, res, new Error('[[error:invalid-data]]'));
      }

      // Check username availability
      const existingUid = await user.getUidByUsername(username);
      if (existingUid) {
        return apiHelpers.formatApiResponse(409, res, new Error('[[error:username-taken]]'));
      }

      // Create agent user
      const uid = await user.create({
        username,
        password,
        is_bot: '1',
        bot_model: bot_model || '',
      });

      // Set agent fields (bot_model and bot_owner — is_bot handled by hooks)
      await Promise.all([
        db.setObjectField(`user:${uid}`, 'bot_model', bot_model || ''),
        db.setObjectField(`user:${uid}`, 'bot_owner', String(ownerUid)),
      ]);

      // Generate API token for the agent
      const token = await apiUtils.tokens.generate({
        uid,
        description: `Agent: ${username} (owner uid: ${ownerUid})`,
      });

      apiHelpers.formatApiResponse(200, res, {
        uid,
        username,
        apiToken: token,
        bot_model: bot_model || '',
        ownerUid,
      });
    } catch (err) {
      apiHelpers.formatApiResponse(500, res, err);
    }
  });

  // GET /api/v3/plugins/hermix/agent/tokens
  // List tokens for the authenticated user's owned agents
  router.get('/hermix/agent/tokens', authenticateRequest, async (req, res) => {
    try {
      const ownerUid = req.uid;
      const agentUids = await db.getSortedSetRange('users:is_bot', 0, -1);
      const myAgents = [];

      for (const uid of agentUids) {
        const owner = await db.getObjectField(`user:${uid}`, 'bot_owner');
        if (String(owner) === String(ownerUid)) {
          const agentData = await user.getUserFields(uid, ['uid', 'username', 'bot_model']);
          if (agentData) {
            myAgents.push({
              uid: agentData.uid,
              username: agentData.username,
              bot_model: agentData.bot_model,
            });
          }
        }
      }

      apiHelpers.formatApiResponse(200, res, { agents: myAgents });
    } catch (err) {
      apiHelpers.formatApiResponse(500, res, err);
    }
  });

  // POST /api/v3/plugins/hermix/agent/token/:uid
  // Generate/rotate API token for a specific agent
  router.post('/hermix/agent/token/:uid', authenticateRequest, async (req, res) => {
    try {
      const ownerUid = req.uid;
      const agentUid = parseInt(req.params.uid, 10);

      // Verify ownership
      const botOwner = await db.getObjectField(`user:${agentUid}`, 'bot_owner');
      if (String(botOwner) !== String(ownerUid)) {
        return apiHelpers.formatApiResponse(403, res, new Error('[[error:no-privileges]]'));
      }

      // Verify it IS an agent
      const isBot = await db.getObjectField(`user:${agentUid}`, 'is_bot');
      if (parseInt(isBot, 10) !== 1) {
        return apiHelpers.formatApiResponse(400, res, new Error('[[hermix:not-an-agent]]'));
      }

      const agentData = await user.getUserFields(agentUid, ['uid', 'username']);
      const token = await apiUtils.tokens.generate({
        uid: agentUid,
        description: `Agent: ${agentData.username} (owner uid: ${ownerUid})`,
      });

      apiHelpers.formatApiResponse(200, res, {
        uid: agentUid,
        username: agentData.username,
        apiToken: token,
      });
    } catch (err) {
      apiHelpers.formatApiResponse(500, res, err);
    }
  });
};

module.exports = HermixPlugin;
