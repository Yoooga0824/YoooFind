import config from "./config.js";

const AUTH_TOKEN_STORAGE_KEY = "authToken";
const USER_PROFILE_STORAGE_KEY = "cachedUserProfile";

const adminNav = document.getElementById("adminNav");
const tabTitle = document.getElementById("tabTitle");
const refreshBtn = document.getElementById("refreshBtn");
const adminIdentity = document.getElementById("adminIdentity");
const adminStatus = document.getElementById("adminStatus");
const usersList = document.getElementById("usersList");
const userDetail = document.getElementById("userDetail");
const visitCards = document.getElementById("visitCards");
const visitTrendTable = document.getElementById("visitTrendTable");
const tokenCards = document.getElementById("tokenCards");
const tokenFocusTitle = document.getElementById("tokenFocusTitle");
const tokenFocusValueHeader = document.getElementById("tokenFocusValueHeader");
const tokenFocusTable = document.getElementById("tokenFocusTable");
const tokenDailyTable = document.getElementById("tokenDailyTable");
const tokenUsersTable = document.getElementById("tokenUsersTable");
const tokenUserDetail = document.getElementById("tokenUserDetail");
const feedbackCards = document.getElementById("feedbackCards");
const feedbackList = document.getElementById("feedbackList");
const feedbackDetail = document.getElementById("feedbackDetail");
const backToChatBtn = document.getElementById("backToChatBtn");

let authToken = localStorage.getItem(AUTH_TOKEN_STORAGE_KEY) || "";
let currentUser = null;
let usersCache = [];
let activeUserID = 0;
let currentTab = "users";
let tokenOverviewCache = null;
let tokenSplitMode = "today";
let activeTokenUserID = 0;
let visitStatsCache = null;
let tokenUserDetailCache = null;
let feedbackCache = [];
let activeFeedbackID = 0;

let visitTrendChartInstance = null;
let tokenFocusChartInstance = null;
let tokenDailyChartInstance = null;
let tokenUserChartInstance = null;

const AXIS_COLOR = "#9aa7c0";

const formatDateLabel = (value) => {
  if (!value) return "";
  const normalized = String(value).trim();
  const [datePart] = normalized.split("T");
  return datePart || normalized;
};

const buildYAxisRange = (series = []) => {
  const values = series.filter((value) => Number.isFinite(value)).map((value) => Number(value));
  if (values.length === 0) return { min: 0, max: 1 };
  const minValue = Math.min(...values);
  const maxValue = Math.max(...values);
  if (minValue === maxValue) {
    if (maxValue === 0) return { min: 0, max: 1 };
    const padding = Math.max(Math.abs(maxValue) * 0.3, 1);
    return { min: 0, max: maxValue + padding };
  }
  const range = maxValue - minValue;
  const padding = Math.max(range * 0.15, 1);
  return { min: Math.max(0, minValue - padding), max: maxValue + padding };
};

const destroyChart = (instanceRef) => {
  if (instanceRef) instanceRef.destroy();
};

const renderLineChart = (canvasId, labels, data, options = {}) => {
  if (typeof Chart === "undefined") return null;
  const canvas = document.getElementById(canvasId);
  if (!canvas) return null;
  const {
    instance,
    borderColor = "#60a5fa",
    pointColor = "#c4b5fd",
    fillColor = "rgba(96, 165, 250, 0.20)",
    label = "",
  } = options;
  if (instance) destroyChart(instance);
  const yAxisRange = buildYAxisRange(data);
  return new Chart(canvas.getContext("2d"), {
    type: "line",
    data: {
      labels,
      datasets: [
        {
          label,
          data,
          fill: true,
          borderColor,
          backgroundColor: fillColor,
          tension: 0.3,
          pointRadius: 2.2,
          pointBackgroundColor: pointColor,
          borderWidth: 2,
        },
      ],
    },
    options: {
      responsive: true,
      maintainAspectRatio: false,
      plugins: { legend: { display: false } },
      scales: {
        x: {
          ticks: { color: AXIS_COLOR, maxTicksLimit: 9, font: { size: 11 } },
          grid: { display: false },
        },
        y: {
          min: yAxisRange.min,
          max: yAxisRange.max,
          ticks: { color: AXIS_COLOR, maxTicksLimit: 5, font: { size: 11 } },
          grid: { color: "rgba(255,255,255,0.06)" },
          border: { display: false },
        },
      },
    },
  });
};

const renderBarChart = (canvasId, labels, data, options = {}) => {
  if (typeof Chart === "undefined") return null;
  const canvas = document.getElementById(canvasId);
  if (!canvas) return null;
  const {
    instance,
    backgroundColor = "rgba(96, 165, 250, 0.72)",
    label = "",
  } = options;
  if (instance) destroyChart(instance);
  const yAxisRange = buildYAxisRange(data);
  return new Chart(canvas.getContext("2d"), {
    type: "bar",
    data: {
      labels,
      datasets: [
        {
          label,
          data,
          backgroundColor,
          borderRadius: 6,
          maxBarThickness: 42,
        },
      ],
    },
    options: {
      responsive: true,
      maintainAspectRatio: false,
      plugins: { legend: { display: false } },
      scales: {
        x: {
          ticks: { color: AXIS_COLOR, maxTicksLimit: 8, font: { size: 11 } },
          grid: { display: false },
        },
        y: {
          min: yAxisRange.min,
          max: yAxisRange.max,
          ticks: { color: AXIS_COLOR, maxTicksLimit: 5, font: { size: 11 } },
          grid: { color: "rgba(255,255,255,0.06)" },
          border: { display: false },
        },
      },
    },
  });
};

const scheduleChartsForTab = (tab) => {
  window.requestAnimationFrame(() => {
    if (tab === "visits") {
      renderVisitChart();
    } else if (tab === "tokens") {
      renderTokenFocusChart();
      renderTokenDailyChart();
      renderTokenUserChart();
    }
  });
};

const formatNumber = (value = 0) => Number(value || 0).toLocaleString("zh-CN");

const escapeHtml = (value = "") =>
  String(value)
    .replaceAll("&", "&amp;")
    .replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;")
    .replaceAll('"', "&quot;")
    .replaceAll("'", "&#39;");

const setStatus = (text = "", isError = false) => {
  if (!adminStatus) return;
  adminStatus.textContent = text;
  adminStatus.style.color = isError ? "#ff91a8" : "";
};

const authFetch = async (url, options = {}) => {
  const headers = { ...(options.headers || {}) };
  if (!(options.body instanceof FormData)) {
    headers["Content-Type"] = headers["Content-Type"] || "application/json";
  }
  if (authToken) {
    headers.Authorization = `Bearer ${authToken}`;
  }
  return fetch(url, { ...options, headers });
};

const parseErrorMessage = async (response, fallback = "请求失败") => {
  try {
    const contentType = response.headers.get("content-type") || "";
    if (contentType.includes("application/json")) {
      const data = await response.json();
      return data?.error?.message || fallback;
    }
    return (await response.text()) || fallback;
  } catch {
    return fallback;
  }
};

const goChatPage = () => {
  window.location.href = "index.html";
};

const ensureAdmin = async () => {
  if (!authToken) {
    goChatPage();
    return false;
  }
  const response = await authFetch(config.ME_URL, { method: "GET" });
  if (!response.ok) {
    goChatPage();
    return false;
  }
  currentUser = await response.json();
  if (!currentUser?.is_admin) {
    goChatPage();
    return false;
  }
  if (adminIdentity) {
    adminIdentity.textContent = `${currentUser.display_name || "管理员"} · ${currentUser.email || ""}`;
  }
  return true;
};

const setTab = (tab) => {
  currentTab = tab;
  const titleMap = {
    users: "用户管理",
    visits: "访问统计",
    tokens: "Token使用总量",
    feedback: "用户反馈",
  };
  tabTitle.textContent = titleMap[tab] || "后台管理";
  document.querySelectorAll(".admin-nav__item").forEach((button) => {
    button.classList.toggle("is-active", button.dataset.tab === tab);
  });
  document.querySelectorAll(".admin-tab").forEach((panel) => {
    panel.classList.toggle("is-active", panel.dataset.tabContent === tab);
  });
  scheduleChartsForTab(tab);
};

const renderFeedbackCards = () => {
  if (!feedbackCards) return;
  const total = feedbackCache.length;
  const unread = feedbackCache.filter((item) => item.status === "new").length;
  feedbackCards.innerHTML = `
    <div class="stat-card">
      <div class="stat-card__label">反馈总数</div>
      <div class="stat-card__value">${formatNumber(total)}</div>
    </div>
    <div class="stat-card">
      <div class="stat-card__label">未读反馈</div>
      <div class="stat-card__value">${formatNumber(unread)}</div>
    </div>
  `;
};

const renderFeedbackList = () => {
  if (!feedbackList) return;
  if (!feedbackCache.length) {
    feedbackList.innerHTML = `<div class="detail-placeholder">暂无反馈</div>`;
    return;
  }
  feedbackList.innerHTML = feedbackCache
    .map((item) => {
      const badgeClass = item.status === "read" ? "feedback-item__badge--read" : "feedback-item__badge--new";
      const badgeText = item.status === "read" ? "已读" : "未读";
      const userLabel = item.user_display_name || item.user_email || "匿名用户";
      return `
      <button type="button" class="feedback-item ${Number(item.id) === Number(activeFeedbackID) ? "is-active" : ""}" data-feedback-id="${item.id}">
        <div class="feedback-item__title">
          ${escapeHtml(item.title || "无标题")}
          <span class="feedback-item__badge ${badgeClass}">${badgeText}</span>
        </div>
        <div class="feedback-item__meta">${escapeHtml(userLabel)} · ${escapeHtml(item.created_at || "-")}</div>
      </button>
    `;
    })
    .join("");
};

const renderFeedbackDetail = (item) => {
  if (!feedbackDetail) return;
  if (!item) {
    feedbackDetail.innerHTML = `<div class="detail-placeholder">请选择左侧反馈</div>`;
    return;
  }
  const userLabel = item.user_display_name || item.user_email || "匿名用户";
  const markReadButton =
    item.status === "read"
      ? `<button type="button" class="ghost-btn" disabled>已读</button>`
      : `<button type="button" class="ghost-btn" data-mark-feedback-read="${item.id}">标记已读</button>`;
  feedbackDetail.innerHTML = `
    <h4 class="feedback-detail__title">${escapeHtml(item.title || "无标题")}</h4>
    <div class="feedback-detail__meta">${escapeHtml(userLabel)}${item.user_email ? ` · ${escapeHtml(item.user_email)}` : ""} · ${escapeHtml(item.created_at || "-")}</div>
    <div class="feedback-detail__content">${escapeHtml(item.content || "")}</div>
    <div class="feedback-detail__actions">${markReadButton}</div>
  `;
};

const loadFeedback = async () => {
  const response = await authFetch(config.ADMIN_FEEDBACK_URL, { method: "GET" });
  if (!response.ok) {
    throw new Error(await parseErrorMessage(response, "加载反馈失败"));
  }
  const payload = await response.json();
  feedbackCache = Array.isArray(payload?.feedback) ? payload.feedback : [];
  if (!activeFeedbackID && feedbackCache.length) {
    activeFeedbackID = Number(feedbackCache[0].id || 0);
  }
  renderFeedbackCards();
  renderFeedbackList();
  const activeItem = feedbackCache.find((item) => Number(item.id) === Number(activeFeedbackID));
  renderFeedbackDetail(activeItem || null);
};

const markFeedbackRead = async (feedbackID) => {
  if (!feedbackID) return;
  const response = await authFetch(`${config.ADMIN_FEEDBACK_URL}/${feedbackID}`, {
    method: "PATCH",
    body: JSON.stringify({ status: "read" }),
  });
  if (!response.ok) {
    throw new Error(await parseErrorMessage(response, "标记已读失败"));
  }
  feedbackCache = feedbackCache.map((item) =>
    Number(item.id) === Number(feedbackID) ? { ...item, status: "read" } : item
  );
  renderFeedbackCards();
  renderFeedbackList();
  const activeItem = feedbackCache.find((item) => Number(item.id) === Number(activeFeedbackID));
  renderFeedbackDetail(activeItem || null);
};

const renderUsers = () => {
  if (!usersList) return;
  if (!usersCache.length) {
    usersList.innerHTML = `<div class="detail-placeholder">暂无用户数据</div>`;
    return;
  }
  usersList.innerHTML = usersCache
    .map(
      (user) => `
      <button type="button" class="user-item ${Number(user.id) === Number(activeUserID) ? "is-active" : ""}" data-user-id="${user.id}">
        <div class="user-item__top">
          <span class="user-item__name">${user.display_name || "用户"}${user.is_admin ? "（管理员）" : ""}</span>
          <span>${user.email || "-"}</span>
        </div>
        <div class="user-item__meta">
          今日 ${formatNumber(user.today_tokens)} token · 历史 ${formatNumber(user.total_tokens)} token
        </div>
      </button>
    `
    )
    .join("");
};

const renderUserChats = (sessions = []) => {
  if (!Array.isArray(sessions) || sessions.length === 0) {
    return `<div class="detail-placeholder">暂无对话记录</div>`;
  }
  const sessionHtml = sessions
    .slice(0, 30)
    .map(
      (session, index) => `
      <div class="chat-session">
        <div class="chat-session__title">#${index + 1} ${session.title || "新聊天"} · ${session.updated_at || "-"}</div>
        ${(session.messages || [])
          .map(
            (message) => `
            <div class="chat-message">
              <div class="role">${message.role || "-"}${message.model ? ` · ${message.model}` : ""} · ${message.created_at || "-"}</div>
              <div>${(message.content || "").replaceAll("<", "&lt;")}</div>
            </div>
          `
          )
          .join("")}
      </div>
    `
    )
    .join("");
  return `<div class="chat-records-scroll">${sessionHtml}</div>`;
};

const fetchUserDetail = async (userID) => {
  setStatus("加载用户详情中...");
  const response = await authFetch(`${config.ADMIN_USERS_URL}/${userID}`, { method: "GET" });
  if (!response.ok) {
    throw new Error(await parseErrorMessage(response, "加载用户详情失败"));
  }
  const detail = await response.json();
  const user = detail?.user || {};
  userDetail.innerHTML = `
    <div class="detail-grid">
      <div><strong>昵称：</strong>${user.display_name || "-"}</div>
      <div><strong>邮箱：</strong>${user.email || "-"}</div>
      <div><strong>姓名：</strong>${user.full_name || "-"}</div>
      <div><strong>今日Token：</strong>${formatNumber(detail?.token_summary?.today_tokens || 0)}</div>
      <div><strong>历史Token：</strong>${formatNumber(detail?.token_summary?.total_tokens || 0)}</div>
      <div><strong>每日上限：</strong>${formatNumber(user.daily_token_limit || 0)}</div>
    </div>
    <div class="detail-block">
      <div class="detail-grid">
        <div class="detail-row">
          <label>修改每日Token上限</label>
          <input id="tokenLimitInput" type="number" min="1" value="${Number(user.daily_token_limit || 1000000)}" />
        </div>
        <div class="detail-row" style="display:flex;align-items:flex-end;">
          <button class="primary-btn" id="saveTokenLimitBtn">保存上限</button>
        </div>
        <div class="detail-row">
          <label>重置密码（至少6位）</label>
          <input id="resetPasswordInput" type="password" minlength="6" placeholder="输入新密码" />
        </div>
        <div class="detail-row" style="display:flex;align-items:flex-end;">
          <button class="primary-btn" id="savePasswordBtn">修改密码</button>
        </div>
      </div>
    </div>
    <div class="detail-block">
      <strong>对话记录</strong>
      ${renderUserChats(detail?.recent_chats || [])}
    </div>
  `;
  bindUserDetailActions(userID);
  setStatus("");
};

const bindUserDetailActions = (userID) => {
  const saveLimitBtn = document.getElementById("saveTokenLimitBtn");
  const tokenLimitInput = document.getElementById("tokenLimitInput");
  const savePasswordBtn = document.getElementById("savePasswordBtn");
  const resetPasswordInput = document.getElementById("resetPasswordInput");

  saveLimitBtn?.addEventListener("click", async () => {
    const nextValue = Number(tokenLimitInput?.value || 0);
    if (!Number.isFinite(nextValue) || nextValue <= 0) {
      setStatus("请输入大于 0 的每日Token上限", true);
      return;
    }
    setStatus("正在保存Token上限...");
    const response = await authFetch(`${config.ADMIN_USERS_URL}/${userID}/token-limit`, {
      method: "PATCH",
      body: JSON.stringify({ daily_token_limit: Math.floor(nextValue) }),
    });
    if (!response.ok) {
      setStatus(await parseErrorMessage(response, "保存失败"), true);
      return;
    }
    setStatus("每日Token上限已更新");
    await loadUsers();
  });

  savePasswordBtn?.addEventListener("click", async () => {
    const password = String(resetPasswordInput?.value || "").trim();
    if (password.length < 6) {
      setStatus("新密码至少 6 位", true);
      return;
    }
    setStatus("正在更新密码...");
    const response = await authFetch(`${config.ADMIN_USERS_URL}/${userID}/password`, {
      method: "PATCH",
      body: JSON.stringify({ new_password: password }),
    });
    if (!response.ok) {
      setStatus(await parseErrorMessage(response, "更新密码失败"), true);
      return;
    }
    resetPasswordInput.value = "";
    setStatus("密码已更新");
  });
};

const loadUsers = async () => {
  const response = await authFetch(config.ADMIN_USERS_URL, { method: "GET" });
  if (!response.ok) {
    throw new Error(await parseErrorMessage(response, "加载用户列表失败"));
  }
  const data = await response.json();
  usersCache = Array.isArray(data?.users) ? data.users : [];
  if (!activeUserID && usersCache.length) {
    activeUserID = Number(usersCache[0].id);
  }
  renderUsers();
  if (activeUserID) {
    await fetchUserDetail(activeUserID);
  }
};

const renderVisitCards = (stats = {}) => {
  visitCards.innerHTML = `
    <div class="stat-card"><div class="stat-card__label">累计访问人数</div><div class="stat-card__value">${formatNumber(stats.total_unique_visitors)}</div></div>
    <div class="stat-card"><div class="stat-card__label">今日访问人数</div><div class="stat-card__value">${formatNumber(stats.today_unique_visitors)}</div></div>
    <div class="stat-card"><div class="stat-card__label">登录访问人数</div><div class="stat-card__value">${formatNumber(stats.logged_in_visitors)}</div></div>
    <div class="stat-card"><div class="stat-card__label">匿名访问人数</div><div class="stat-card__value">${formatNumber(stats.anonymous_visitors)}</div></div>
  `;
};

const renderVisitChart = () => {
  const trend = Array.isArray(visitStatsCache?.daily_trend) ? visitStatsCache.daily_trend : [];
  visitTrendChartInstance = renderLineChart(
    "visitTrendChart",
    trend.map((item) => formatDateLabel(item.date)),
    trend.map((item) => Number(item.count || 0)),
    {
      instance: visitTrendChartInstance,
      borderColor: "#34d399",
      pointColor: "#6ee7b7",
      fillColor: "rgba(52, 211, 153, 0.18)",
      label: "访问人数",
    }
  );
};

const loadVisitStats = async () => {
  const response = await authFetch(config.ADMIN_VISIT_STATS_URL, { method: "GET" });
  if (!response.ok) {
    throw new Error(await parseErrorMessage(response, "加载访问统计失败"));
  }
  visitStatsCache = await response.json();
  renderVisitCards(visitStatsCache);
  const trend = Array.isArray(visitStatsCache?.daily_trend) ? visitStatsCache.daily_trend : [];
  visitTrendTable.innerHTML = trend
    .map((item) => `<tr><td>${item.date || "-"}</td><td>${formatNumber(item.count || 0)}</td></tr>`)
    .join("");
  if (currentTab === "visits") {
    scheduleChartsForTab("visits");
  }
};

const getTokenUsersSortedByMode = (mode = "today") => {
  const users = Array.isArray(tokenOverviewCache?.users) ? [...tokenOverviewCache.users] : [];
  if (mode === "history") {
    users.sort((a, b) => Number(b.total_tokens || 0) - Number(a.total_tokens || 0));
  } else {
    users.sort((a, b) => Number(b.today_tokens || 0) - Number(a.today_tokens || 0));
  }
  return users;
};

const renderTokenCards = (overview = {}) => {
  tokenCards.innerHTML = `
    <button type="button" class="stat-card is-clickable ${tokenSplitMode === "today" ? "is-active" : ""}" data-token-mode="today">
      <div class="stat-card__head">
        <span class="stat-card__icon"><i class='bx bx-sun'></i></span>
        <div class="stat-card__label">今日总Token（点我看今日明细）</div>
      </div>
      <div class="stat-card__value">${formatNumber(overview.today_total_tokens)}</div>
    </button>
    <button type="button" class="stat-card is-clickable ${tokenSplitMode === "history" ? "is-active" : ""}" data-token-mode="history">
      <div class="stat-card__head">
        <span class="stat-card__icon"><i class='bx bx-history'></i></span>
        <div class="stat-card__label">历史总Token（点我看历史明细）</div>
      </div>
      <div class="stat-card__value">${formatNumber(overview.history_total_tokens)}</div>
    </button>
  `;
};

const renderTokenFocusChart = () => {
  const mode = tokenSplitMode === "history" ? "history" : "today";
  const users = getTokenUsersSortedByMode(mode).slice(0, 12);
  const labels = users.map((item) => item.display_name || item.email || "用户");
  const data = users.map((item) =>
    Number(mode === "history" ? item.total_tokens : item.today_tokens) || 0
  );
  tokenFocusChartInstance = renderBarChart(
    "tokenFocusChart",
    labels,
    data,
    {
      instance: tokenFocusChartInstance,
      backgroundColor:
        mode === "history" ? "rgba(139, 92, 246, 0.78)" : "rgba(96, 165, 250, 0.78)",
      label: mode === "history" ? "历史Token" : "今日Token",
    }
  );
};

const renderTokenDailyChart = () => {
  const points = Array.isArray(tokenOverviewCache?.daily_total) ? tokenOverviewCache.daily_total : [];
  tokenDailyChartInstance = renderLineChart(
    "tokenDailyChart",
    points.map((item) => formatDateLabel(item.date)),
    points.map((item) => Number(item.total_tokens || 0)),
    {
      instance: tokenDailyChartInstance,
      borderColor: "#8b5cf6",
      pointColor: "#ec4899",
      fillColor: "rgba(139, 92, 246, 0.18)",
      label: "每日Token",
    }
  );
};

const renderTokenUserChart = () => {
  const points = Array.isArray(tokenUserDetailCache?.token_by_day)
    ? tokenUserDetailCache.token_by_day.slice(-30)
    : [];
  tokenUserChartInstance = renderLineChart(
    "tokenUserChart",
    points.map((item) => formatDateLabel(item.date)),
    points.map((item) => Number(item.total_tokens || 0)),
    {
      instance: tokenUserChartInstance,
      borderColor: "#f59e0b",
      pointColor: "#fbbf24",
      fillColor: "rgba(245, 158, 11, 0.18)",
      label: "用户每日Token",
    }
  );
};

const renderTokenFocus = () => {
  const mode = tokenSplitMode === "history" ? "history" : "today";
  const users = getTokenUsersSortedByMode(mode);
  if (tokenFocusTitle) {
    tokenFocusTitle.textContent = mode === "history" ? "历史Token分布（按用户）" : "今日Token分布（按用户）";
  }
  if (tokenFocusValueHeader) {
    tokenFocusValueHeader.textContent = mode === "history" ? "历史总量" : "今日总量";
  }
  if (tokenFocusTable) {
    tokenFocusTable.innerHTML = users
      .map((item) => {
        const value = mode === "history" ? item.total_tokens : item.today_tokens;
        return `<tr><td>${item.display_name || "用户"}<br/><span style="color:#9aa7c0;font-size:12px;">${item.email || ""}</span></td><td>${formatNumber(value || 0)}</td></tr>`;
      })
      .join("");
  }
  if (currentTab === "tokens") {
    renderTokenFocusChart();
  }
};

const renderTokenUsersTable = () => {
  const users = Array.isArray(tokenOverviewCache?.users) ? tokenOverviewCache.users : [];
  tokenUsersTable.innerHTML = users
    .map((item) => {
      const isActive = Number(item.user_id) === Number(activeTokenUserID);
      return `
        <tr class="clickable-row ${isActive ? "is-active" : ""}" data-token-user-id="${item.user_id}">
          <td>${item.display_name || "用户"}<br/><span style="color:#9aa7c0;font-size:12px;">${item.email || ""}</span></td>
          <td>${formatNumber(item.today_tokens || 0)}</td>
          <td>${formatNumber(item.total_tokens || 0)}</td>
        </tr>
      `;
    })
    .join("");
};

const renderTokenUserDetail = (detail) => {
  if (!tokenUserDetail) return;
  tokenUserDetailCache = detail;
  const user = detail?.user || {};
  const points = Array.isArray(detail?.token_by_day) ? detail.token_by_day.slice(-30) : [];
  tokenUserDetail.innerHTML = `
    <div class="detail-grid" style="margin-top:8px;">
      <div><strong>用户：</strong>${user.display_name || "用户"}（${user.email || "-"}）</div>
      <div><strong>每日上限：</strong>${formatNumber(user.daily_token_limit || 0)}</div>
      <div><strong>今日总量：</strong>${formatNumber(detail?.token_summary?.today_tokens || 0)}</div>
      <div><strong>历史总量：</strong>${formatNumber(detail?.token_summary?.total_tokens || 0)}</div>
    </div>
    <div class="table-wrap">
      <table>
        <thead>
          <tr>
            <th>日期</th>
            <th>Total</th>
          </tr>
        </thead>
        <tbody>
          ${points.map((item) => `<tr><td>${item.date || "-"}</td><td>${formatNumber(item.total_tokens || 0)}</td></tr>`).join("")}
        </tbody>
      </table>
    </div>
  `;
  if (currentTab === "tokens") {
    scheduleChartsForTab("tokens");
  }
};

const loadTokenUserDetail = async (userID) => {
  if (!userID) return;
  setStatus("加载用户Token详情中...");
  const response = await authFetch(`${config.ADMIN_USERS_URL}/${userID}`, { method: "GET" });
  if (!response.ok) {
    throw new Error(await parseErrorMessage(response, "加载用户Token详情失败"));
  }
  const detail = await response.json();
  renderTokenUserDetail(detail);
  setStatus("");
};

const loadTokenStats = async () => {
  const response = await authFetch(config.ADMIN_TOKEN_STATS_URL, { method: "GET" });
  if (!response.ok) {
    throw new Error(await parseErrorMessage(response, "加载Token统计失败"));
  }
  tokenOverviewCache = await response.json();
  renderTokenCards(tokenOverviewCache);
  renderTokenFocus();
  tokenDailyTable.innerHTML = (tokenOverviewCache?.daily_total || [])
    .map(
      (item) =>
        `<tr><td>${item.date || "-"}</td><td>${formatNumber(item.prompt_tokens || 0)}</td><td>${formatNumber(item.completion_tokens || 0)}</td><td>${formatNumber(item.total_tokens || 0)}</td></tr>`
    )
    .join("");
  if (!activeTokenUserID) {
    activeTokenUserID = Number(tokenOverviewCache?.users?.[0]?.user_id || 0);
  }
  renderTokenUsersTable();
  if (activeTokenUserID) {
    await loadTokenUserDetail(activeTokenUserID);
  } else if (tokenUserDetail) {
    tokenUserDetail.innerHTML = "暂无可展示的用户Token详情";
    tokenUserDetailCache = null;
  }
  if (currentTab === "tokens") {
    scheduleChartsForTab("tokens");
  }
};

const refreshCurrentTab = async () => {
  setStatus("正在刷新...");
  try {
    if (currentTab === "users") {
      await loadUsers();
    } else if (currentTab === "visits") {
      await loadVisitStats();
    } else if (currentTab === "feedback") {
      await loadFeedback();
    } else {
      await loadTokenStats();
    }
    setStatus("刷新完成");
  } catch (error) {
    setStatus(error.message || "刷新失败", true);
  }
};

const bindEvents = () => {
  adminNav?.addEventListener("click", (event) => {
    const button = event.target.closest("[data-tab]");
    if (!button) return;
    const tab = button.dataset.tab;
    if (!tab || tab === currentTab) return;
    setTab(tab);
    void refreshCurrentTab();
  });

  usersList?.addEventListener("click", (event) => {
    const button = event.target.closest("[data-user-id]");
    if (!button) return;
    const userID = Number(button.dataset.userId);
    if (!userID) return;
    activeUserID = userID;
    renderUsers();
    void fetchUserDetail(activeUserID);
  });

  tokenCards?.addEventListener("click", (event) => {
    const card = event.target.closest("[data-token-mode]");
    if (!card) return;
    const nextMode = card.dataset.tokenMode;
    if (nextMode !== "today" && nextMode !== "history") return;
    if (tokenSplitMode === nextMode) return;
    tokenSplitMode = nextMode;
    renderTokenCards(tokenOverviewCache || {});
    renderTokenFocus();
    renderTokenFocusChart();
  });

  tokenUsersTable?.addEventListener("click", (event) => {
    const row = event.target.closest("[data-token-user-id]");
    if (!row) return;
    const userID = Number(row.dataset.tokenUserId);
    if (!userID) return;
    activeTokenUserID = userID;
    renderTokenUsersTable();
    void loadTokenUserDetail(userID);
  });

  feedbackList?.addEventListener("click", (event) => {
    const button = event.target.closest("[data-feedback-id]");
    if (!button) return;
    const feedbackID = Number(button.dataset.feedbackId);
    if (!feedbackID) return;
    activeFeedbackID = feedbackID;
    renderFeedbackList();
    const activeItem = feedbackCache.find((item) => Number(item.id) === Number(activeFeedbackID));
    renderFeedbackDetail(activeItem || null);
  });

  feedbackDetail?.addEventListener("click", (event) => {
    const button = event.target.closest("[data-mark-feedback-read]");
    if (!button) return;
    const feedbackID = Number(button.dataset.markFeedbackRead);
    if (!feedbackID) return;
    void (async () => {
      try {
        setStatus("正在标记已读...");
        await markFeedbackRead(feedbackID);
        setStatus("已标记为已读");
      } catch (error) {
        setStatus(error.message || "标记失败", true);
      }
    })();
  });

  refreshBtn?.addEventListener("click", () => {
    void refreshCurrentTab();
  });

  backToChatBtn?.addEventListener("click", () => {
    goChatPage();
  });
};

const init = async () => {
  bindEvents();
  const ok = await ensureAdmin();
  if (!ok) return;
  setTab("users");
  try {
    await Promise.all([loadUsers(), loadVisitStats(), loadTokenStats(), loadFeedback()]);
    setStatus("数据已加载");
  } catch (error) {
    setStatus(error.message || "初始化失败", true);
  }
};

void init();
