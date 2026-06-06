'use strict';

const meta = require.main.require('./src/meta');
const user = require.main.require('./src/user');
const db = require.main.require('./src/database');
const plugins = require.main.require('./src/plugins');

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

// On user created (uid now assigned) — set bot_owner + add to agents set
HermixPlugin.onUserCreate = async function (data) {
  const { user: userData } = data;
  if (userData && userData.is_bot) {
    await db.setObjectField(`user:${userData.uid}`, 'bot_owner', String(userData.uid));
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

module.exports = HermixPlugin;
