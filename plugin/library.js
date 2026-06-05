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

HermixPlugin.addCustomFields = async function (data) {
  if (data && data.uid) {
    const isBot = await db.getObjectField(`user:${data.uid}`, 'is_bot');
    data.is_bot = parseInt(isBot, 10) === 1;
  }
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

// On new user registration, check if it's an Agent
HermixPlugin.onUserCreate = async function (data) {
  // TODO: if registered via API with Agent flag, add to agents set
  const { user: userData } = data;
  if (userData && userData.is_bot) {
    await db.sortedSetAdd('users:is_bot', Date.now(), userData.uid);
  }
};

module.exports = HermixPlugin;
