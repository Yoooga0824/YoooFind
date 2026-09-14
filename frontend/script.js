import { createThinkingRadialGlow } from "./thinking-radial-glow.js";

const messageForm = document.querySelector(".prompt__form");
const chatHistoryContainer = document.querySelector(".chats");
const promptContainer = document.querySelector(".prompt");

createThinkingRadialGlow({ chatContainer: chatHistoryContainer });

const themeToggleButton = document.getElementById("themeToggler");
const sidebarElement = document.getElementById("appSidebar");
const sidebarBackdrop = document.getElementById("sidebarBackdrop");
const sidebarMobileToggleButton = document.getElementById("sidebarMobileToggle");
const sidebarCollapseButton = document.getElementById("sidebarCollapseButton");
const sidebarExpandButton = document.getElementById("sidebarExpandButton");
const newChatButton = document.getElementById("newChatButton");
const sidebarHistory = document.getElementById("sidebarHistory");
const voiceInputButton = document.getElementById("voiceButton");
const attachButton = document.getElementById("attachButton");
const attachMenu = document.getElementById("attachMenu");
const fileInput = document.getElementById("fileInput");
const attachmentList = document.getElementById("attachmentList");
const sidebarUserCard = document.getElementById("sidebarUserCard");
const sidebarUserAvatar = document.getElementById("sidebarUserAvatar");
const sidebarUserLabel = document.getElementById("sidebarUserLabel");
const sidebarSettingsButton = document.getElementById("sidebarSettingsButton");
const authModal = document.getElementById("authModal");
const authForm = document.getElementById("authForm");
const authEmailInput = document.getElementById("authEmail");
const authPasswordInput = document.getElementById("authPassword");
const authSubmitButton = document.getElementById("authSubmitButton");
const authStatus = document.getElementById("authStatus");
const profileModal = document.getElementById("profileModal");
const settingsModal = document.getElementById("settingsModal");
const settingsNav = document.getElementById("settingsNav");
const sponsorMarkdownContent = document.getElementById("sponsorMarkdownContent");
const feedbackForm = document.getElementById("feedbackForm");
const feedbackTitleInput = document.getElementById("feedbackTitle");
const feedbackContentInput = document.getElementById("feedbackContent");
const feedbackStatus = document.getElementById("feedbackStatus");
const feedbackSubmitButton = document.getElementById("feedbackSubmitButton");
const profileForm = document.getElementById("profileForm");
const profileDisplayNameInput = document.getElementById("profileDisplayName");
const profileFullNameInput = document.getElementById("profileFullName");
const profileBioInput = document.getElementById("profileBio");
const profileStatus = document.getElementById("profileStatus");
const profileAvatarPreview = document.getElementById("profileAvatarPreview");
const avatarInput = document.getElementById("avatarInput");
const logoutButton = document.getElementById("logoutButton");
const adminEntryButton = document.getElementById("adminEntryButton");
const usageSummaryText = document.getElementById("usageSummaryText");
const usageToggleGroup = document.getElementById("usageToggleGroup");
const usageChartCanvas = document.getElementById("usageChartCanvas");
const modelPicker = document.getElementById("modelPicker");
const modelPickerTrigger = document.getElementById("modelPickerTrigger");
const modelPickerPanel = document.getElementById("modelPickerPanel");
const modelPickerSummary = document.getElementById("modelPickerSummary");
const modelPickerOptions = document.getElementById("modelPickerOptions");
const deepSearchToggle = document.getElementById("deepSearchToggle");

// State variables
let currentUserMessage = null;
const sessionGenerationState = new Map();
let pendingAttachments = [];
let chatSessions = [];
let activeSessionId = null;
let authMode = "login";
let authToken = "";
let currentUser = null;
let usageChartInstance = null;
let usageChartMode = "recent30";
let usageChartDataCache = {
  recentSummary: null,
  totalSummary: null,
};
let selectedModelKeys = [DEFAULT_MODEL_KEY];
let modelPickerHideTimer = null;
let deepSearchEnabled = false;

const MAX_ATTACHMENT_COUNT = 6;
const MAX_ATTACHMENT_SIZE = 5 * 1024 * 1024;
const MAX_ATTACHMENT_TEXT_CHARS = 12000;
const IMAGE_ACCEPT = "image/*";
const FILE_ACCEPT =
  ".txt,.md,.json,.csv,.js,.ts,.tsx,.go,.py,.java,.c,.cpp,.html,.css,.xml,.yaml,.yml,.pdf";

import config from "./config.js";
import {
  DEFAULT_MODEL_KEY,
  MAX_SELECTED_MODELS,
  MODEL_CATALOG,
  MODEL_LABELS,
  getModelIcon,
  getModelLabel,
  normalizeModelSelection,
} from "./src/model-catalog.js";

// Initialize highlight.js fallback with common languages.
hljs.configure({
  languages: ["javascript", "python", "bash", "typescript", "json", "html", "css"],
});

const API_REQUEST_URL = config.BACKEND_API_URL;
const CHAT_SESSIONS_URL = config.CHAT_SESSIONS_URL;
const AUTH_TOKEN_STORAGE_KEY = "authToken";
const MODEL_SELECTION_STORAGE_KEY = "selectedModels";
const DEEP_SEARCH_STORAGE_KEY = "deepSearchEnabled";
const USER_PROFILE_STORAGE_KEY = "cachedUserProfile";
const MAX_CLOUD_SESSIONS = 30;
const promptInput = messageForm.querySelector(".prompt__form-input");
const themeRoot = document.documentElement;
const headerTypingTitle = document.querySelector(".header__typing-title");
const headerTypingText = document.querySelector(".header__typing-text");
const headerCursor = document.querySelector(".header__cursor");
const PROMPT_INPUT_MIN_HEIGHT = 64;
const PROMPT_INPUT_MAX_HEIGHT = 180;
const SCROLL_FOLLOW_THRESHOLD = 80;
let shouldAutoScroll = true;
const pageScrollRoot = document.scrollingElement || document.documentElement;
const THINK_TAG_PATTERN = /<think>\s*([\s\S]*?)\s*<\/think>/gi;
const ENABLE_REASONING_OUTPUT = true;
const REASONING_SCROLL_FOLLOW_THRESHOLD = 20;
const SHORT_CODE_BLOCK_MAX_CHARS = 72;
const MOBILE_BREAKPOINT = 980;
let shikiHighlighterPromise = null;
const STREAM_REASONING_CHUNK_SIZE = 1;
const STREAM_REASONING_CHUNK_DELAY_MS = 10;
const STREAM_CONTENT_CHUNK_SIZE = 1;
const STREAM_CONTENT_CHUNK_DELAY_MS = 8;

const SHIKI_LANGUAGES = [
  "javascript",
  "typescript",
  "jsx",
  "tsx",
  "json",
  "bash",
  "shell",
  "go",
  "python",
  "java",
  "c",
  "cpp",
  "html",
  "css",
  "markdown",
  "yaml",
  "sql",
  "text",
];

marked.setOptions({
  gfm: true,
  breaks: true,
});

const formatBytes = (bytes = 0) => {
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
};

const splitTextByRuneChunks = (text = "", chunkSize = 16) => {
  const source = String(text || "");
  if (!source) return [];
  if (chunkSize <= 0) return [source];
  const runes = Array.from(source);
  const chunks = [];
  for (let index = 0; index < runes.length; index += chunkSize) {
    chunks.push(runes.slice(index, index + chunkSize).join(""));
  }
  return chunks;
};

const sleep = (ms) =>
  new Promise((resolve) => {
    window.setTimeout(resolve, ms);
  });

const escapeHtml = (text = "") =>
  text
    .replace(/&/g, "&amp;")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;")
    .replace(/"/g, "&quot;")
    .replace(/'/g, "&#39;");

const saveModelSelection = () => {
  localStorage.setItem(MODEL_SELECTION_STORAGE_KEY, JSON.stringify(selectedModelKeys));
};

const renderModelPickerSummary = () => {
  if (!modelPickerSummary) return;
  modelPickerSummary.textContent = "";
  selectedModelKeys.forEach((key, index) => {
    const item = document.createElement("span");
    item.className = "model-picker__summary-item";

    const icon = getModelIcon(key);
    if (icon) {
      const iconImage = document.createElement("img");
      iconImage.className = "model-picker__summary-icon";
      iconImage.src = icon;
      iconImage.alt = "";
      iconImage.loading = "lazy";
      iconImage.decoding = "async";
      iconImage.setAttribute("aria-hidden", "true");
      item.appendChild(iconImage);
    }

    const label = document.createElement("span");
    label.className = "model-picker__summary-label";
    label.textContent = getModelLabel(key);
    item.appendChild(label);
    modelPickerSummary.appendChild(item);

    if (index < selectedModelKeys.length - 1) {
      const separator = document.createElement("span");
      separator.className = "model-picker__summary-separator";
      separator.textContent = "+";
      modelPickerSummary.appendChild(separator);
    }
  });
};

const syncModelPickerOptionStates = () => {
  if (!modelPickerPanel) return;
  modelPickerPanel.querySelectorAll("[data-model-option]").forEach((option) => {
    const modelKey = String(option.dataset.modelKey || "");
    const isSelected = selectedModelKeys.includes(modelKey);
    const isDisabled = !isSelected && selectedModelKeys.length >= MAX_SELECTED_MODELS;
    option.classList.toggle("is-selected", isSelected);
    option.classList.toggle("is-disabled", isDisabled);
    option.setAttribute("aria-pressed", isSelected ? "true" : "false");
    option.setAttribute("aria-disabled", isDisabled ? "true" : "false");
  });
};

const setSelectedModels = (nextModels = []) => {
  selectedModelKeys = normalizeModelSelection(nextModels);
  renderModelPickerSummary();
  syncModelPickerOptionStates();
  saveModelSelection();
};

const saveDeepSearchState = () => {
  localStorage.setItem(DEEP_SEARCH_STORAGE_KEY, deepSearchEnabled ? "true" : "false");
};

const renderDeepSearchToggle = () => {
  if (!deepSearchToggle) return;
  deepSearchToggle.classList.toggle("is-active", deepSearchEnabled);
  deepSearchToggle.setAttribute("aria-pressed", deepSearchEnabled ? "true" : "false");
};

const setDeepSearchEnabled = (enabled) => {
  deepSearchEnabled = !!enabled;
  renderDeepSearchToggle();
  saveDeepSearchState();
};

const renderModelPickerOptions = () => {
  if (!modelPickerOptions) return;
  modelPickerOptions.innerHTML = MODEL_CATALOG
    .map(
      (item) => `
        <button
          type="button"
          class="model-picker__option"
          data-model-option
          data-model-key="${escapeHtml(item.key)}"
          aria-pressed="false"
          aria-disabled="false"
        >
          <span class="model-picker__option-main">
            <img
              class="model-picker__option-icon"
              src="${escapeHtml(item.icon || "")}"
              alt=""
              loading="lazy"
              decoding="async"
              aria-hidden="true"
            />
            <span class="model-picker__option-label">${escapeHtml(item.label)}</span>
          </span>
          <span class="model-picker__option-check" aria-hidden="true">
            <svg viewBox="0 0 20 20" focusable="false">
              <path d="M7.6 13.4 4.8 10.6a1 1 0 0 0-1.4 1.4l3.5 3.5a1 1 0 0 0 1.4 0l8.2-8.2a1 1 0 0 0-1.4-1.4z"></path>
            </svg>
          </span>
        </button>
      `
    )
    .join("");
};

const normalizeAvatarUrl = (rawUrl = "") => {
  if (!rawUrl) return "assets/profile.png";
  if (rawUrl.startsWith("http://") || rawUrl.startsWith("https://")) return rawUrl;
  return `${config.BACKEND_BASE_URL}${rawUrl}`;
};

const normalizeComparableUrl = (url = "") => {
  try {
    return new URL(url, window.location.href).href;
  } catch {
    return String(url || "");
  }
};

const setImageSourceIfChanged = (imageElement, nextSrc) => {
  if (!imageElement) return;
  const next = String(nextSrc || "");
  const current = String(imageElement.getAttribute("src") || "");
  if (normalizeComparableUrl(current) === normalizeComparableUrl(next)) return;
  imageElement.src = next;
};

const getCachedUserProfile = () => {
  try {
    const raw = localStorage.getItem(USER_PROFILE_STORAGE_KEY);
    if (!raw) return null;
    const parsed = JSON.parse(raw);
    if (!parsed || typeof parsed !== "object") return null;
    return parsed;
  } catch {
    return null;
  }
};

const persistCurrentUserProfile = () => {
  if (!currentUser || typeof currentUser !== "object") return;
  const profile = {
    display_name: currentUser.display_name || "",
    full_name: currentUser.full_name || "",
    bio: currentUser.bio || "",
    avatar_url: normalizeAvatarUrl(currentUser.avatar_url),
  };
  localStorage.setItem(USER_PROFILE_STORAGE_KEY, JSON.stringify(profile));
};

const authFetch = async (url, options = {}) => {
  const headers = {
    ...(options.headers || {}),
  };
  if (!(options.body instanceof FormData)) {
    headers["Content-Type"] = headers["Content-Type"] || "application/json";
  }
  if (authToken) {
    headers.Authorization = `Bearer ${authToken}`;
  }
  return fetch(url, { ...options, headers });
};

const setAuthStatusText = (text = "", isError = false) => {
  if (!authStatus) return;
  authStatus.textContent = text;
  authStatus.classList.toggle("modal-status--error", isError);
};

const setProfileStatusText = (text = "", isError = false) => {
  if (!profileStatus) return;
  profileStatus.textContent = text;
  profileStatus.classList.toggle("modal-status--error", isError);
};

const modalCloseTimers = new WeakMap();

const getModalCard = (modalElement) => modalElement?.querySelector(".modal-card");

const openModal = (modalElement) => {
  if (!modalElement) return;
  const pendingTimer = modalCloseTimers.get(modalElement);
  if (pendingTimer) {
    window.clearTimeout(pendingTimer);
    modalCloseTimers.delete(modalElement);
  }
  modalElement.classList.remove("hide");
  modalElement.setAttribute("aria-hidden", "false");
  const modalCard = getModalCard(modalElement);
  modalCard?.classList.remove("is-visible");
  modalElement.classList.remove("is-visible");
  requestAnimationFrame(() => {
    modalElement.classList.add("is-visible");
    modalCard?.classList.add("is-visible");
  });
};

const closeModal = (modalElement) => {
  if (!modalElement || modalElement.classList.contains("hide")) return;
  modalElement.classList.remove("is-visible");
  getModalCard(modalElement)?.classList.remove("is-visible");
  const timer = window.setTimeout(() => {
    if (!modalElement.classList.contains("is-visible")) {
      modalElement.classList.add("hide");
      modalElement.setAttribute("aria-hidden", "true");
    }
    modalCloseTimers.delete(modalElement);
  }, 240);
  modalCloseTimers.set(modalElement, timer);
};

let settingsTab = "sponsor";
let sponsorMarkdownCache = null;
let sponsorMarkdownLoading = null;

const setSettingsTab = (tab) => {
  settingsTab = tab === "feedback" ? "feedback" : "sponsor";
  settingsNav?.querySelectorAll("[data-settings-tab]").forEach((button) => {
    button.classList.toggle("is-active", button.dataset.settingsTab === settingsTab);
  });
  document.querySelectorAll("[data-settings-panel]").forEach((panel) => {
    panel.classList.toggle("is-active", panel.dataset.settingsPanel === settingsTab);
  });
};

const setFeedbackStatusText = (text = "", isError = false) => {
  if (!feedbackStatus) return;
  feedbackStatus.textContent = text;
  feedbackStatus.classList.toggle("modal-status--error", isError);
};

const loadSponsorMarkdown = async () => {
  if (sponsorMarkdownCache && sponsorMarkdownContent) {
    sponsorMarkdownContent.innerHTML = marked.parse(sponsorMarkdownCache);
    return;
  }
  if (sponsorMarkdownLoading) {
    await sponsorMarkdownLoading;
    return;
  }
  sponsorMarkdownLoading = (async () => {
    try {
      const response = await fetch("assets/content/sponsor.md", { cache: "no-cache" });
      if (!response.ok) throw new Error("加载失败");
      const markdown = await response.text();
      sponsorMarkdownCache = markdown;
      if (sponsorMarkdownContent) {
        sponsorMarkdownContent.innerHTML = marked.parse(markdown);
      }
    } catch {
      if (sponsorMarkdownContent) {
        sponsorMarkdownContent.innerHTML = marked.parse("> 赞助内容加载失败，请稍后重试。");
      }
    } finally {
      sponsorMarkdownLoading = null;
    }
  })();
  await sponsorMarkdownLoading;
};

const openSettingsModal = () => {
  setSettingsTab("sponsor");
  setFeedbackStatusText("");
  openModal(settingsModal);
  void loadSponsorMarkdown();
};

const handleFeedbackSubmit = async (event) => {
  event.preventDefault();
  const title = feedbackTitleInput?.value?.trim() || "";
  const content = feedbackContentInput?.value?.trim() || "";
  if (!title || !content) {
    setFeedbackStatusText("请填写标题和内容", true);
    return;
  }
  if (feedbackSubmitButton) feedbackSubmitButton.disabled = true;
  try {
    const response = await authFetch(config.FEEDBACK_URL, {
      method: "POST",
      body: JSON.stringify({ title, content }),
    });
    if (!response.ok) {
      setFeedbackStatusText(await parseErrorMessage(response, "提交失败"), true);
      return;
    }
    feedbackForm?.reset();
    setFeedbackStatusText("反馈已提交，感谢你的意见！");
  } catch (error) {
    setFeedbackStatusText(error?.message || "提交失败", true);
  } finally {
    if (feedbackSubmitButton) feedbackSubmitButton.disabled = false;
  }
};

const setAuthMode = (mode = "login") => {
  authMode = mode === "register" ? "register" : "login";
  document.querySelectorAll("[data-auth-mode]").forEach((button) => {
    button.classList.toggle("is-active", button.dataset.authMode === authMode);
  });
  if (authSubmitButton) {
    authSubmitButton.textContent = authMode === "register" ? "注册" : "登录";
  }
  setAuthStatusText("");
};

const applyUserProfileToUI = () => {
  const isLoggedIn = !!authToken && !!currentUser;
  if (sidebarUserLabel) {
    sidebarUserLabel.textContent = isLoggedIn ? (currentUser.display_name || "用户") : "登录";
  }
  if (adminEntryButton) {
    const isAdmin = Boolean(isLoggedIn && currentUser?.is_admin);
    adminEntryButton.classList.toggle("hide", !isAdmin);
  }
  const avatarUrl = isLoggedIn ? normalizeAvatarUrl(currentUser.avatar_url) : "assets/profile.png";
  setImageSourceIfChanged(sidebarUserAvatar, avatarUrl);
  setImageSourceIfChanged(profileAvatarPreview, avatarUrl);
};

const recordVisit = async () => {
  try {
    await authFetch(config.VISIT_URL, { method: "POST" });
  } catch (error) {
    console.warn("record visit failed:", error);
  }
};

const logout = () => {
  authToken = "";
  currentUser = null;
  chatSessions = [createSession("新聊天")];
  activeSessionId = chatSessions[0].id;
  renderSidebarSessions();
  renderActiveSessionMessages();
  localStorage.removeItem(AUTH_TOKEN_STORAGE_KEY);
  localStorage.removeItem(USER_PROFILE_STORAGE_KEY);
  applyUserProfileToUI();
  closeModal(profileModal);
  setAuthMode("login");
  openModal(authModal);
};

const parseErrorMessage = async (response, fallback = "请求失败") => {
  try {
    const contentType = response.headers.get("content-type") || "";
    if (contentType.includes("application/json")) {
      const data = await response.json();
      return data?.error?.message || fallback;
    }
    const text = await response.text();
    return text || fallback;
  } catch {
    return fallback;
  }
};

const getCurrentCodeTheme = () =>
  themeRoot.classList.contains("light_mode") ? "github-light" : "github-dark";

const getShikiHighlighter = async () => {
  if (shikiHighlighterPromise) return shikiHighlighterPromise;
  shikiHighlighterPromise = (async () => {
    try {
      const { createHighlighter } = await import("https://cdn.jsdelivr.net/npm/shiki@1.29.2/+esm");
      return await createHighlighter({
        themes: ["github-dark", "github-light"],
        langs: SHIKI_LANGUAGES,
      });
    } catch (error) {
      console.warn("Shiki unavailable, using highlight.js fallback.", error);
      return null;
    }
  })();
  return shikiHighlighterPromise;
};

const getCodeLanguage = (codeElement) => {
  const languageClass = [...(codeElement?.classList || [])]
    .find((cls) => cls.startsWith("language-"))
    ?.replace("language-", "")
    ?.toLowerCase();
  if (!languageClass) return "text";
  if (languageClass === "sh" || languageClass === "zsh") return "bash";
  return languageClass;
};

const normalizeShortCodeBlocks = (rootElement) => {
  if (!rootElement) return;
  const codeBlocks = rootElement.querySelectorAll("pre code");
  codeBlocks.forEach((codeElement) => {
    const preElement = codeElement.closest("pre");
    if (!preElement) return;
    const rawCode = (codeElement.textContent || "").trim();
    if (!rawCode) return;
    const lineCount = rawCode.split(/\r?\n/).length;
    if (lineCount > 1 || rawCode.length > SHORT_CODE_BLOCK_MAX_CHARS) return;
    const inlineWrapper = document.createElement("p");
    inlineWrapper.className = "message__inline-snippet";
    const inlineCode = document.createElement("code");
    inlineCode.textContent = rawCode;
    inlineWrapper.appendChild(inlineCode);
    preElement.replaceWith(inlineWrapper);
  });
};

const applyContentGroupCards = (rootElement) => {
  if (!rootElement) return;
  rootElement.classList.add("message__text--enhanced");
};

const applyShikiToMessageCodeBlocks = async (rootElement) => {
  if (!rootElement) return;
  const highlighter = await getShikiHighlighter();
  if (!highlighter) {
    rootElement.querySelectorAll("pre code").forEach((codeElement) => {
      hljs.highlightElement(codeElement);
    });
    return;
  }

  const theme = getCurrentCodeTheme();
  const loadedLanguages = new Set(highlighter.getLoadedLanguages().map((lang) => String(lang)));
  const codeBlocks = rootElement.querySelectorAll("pre code");
  for (const codeElement of codeBlocks) {
    if (codeElement.dataset.shikiReady === "true") continue;
    const preElement = codeElement.closest("pre");
    if (!preElement) continue;

    const sourceCode = codeElement.textContent || "";
    const preferredLang = getCodeLanguage(codeElement);
    const resolvedLang = loadedLanguages.has(preferredLang) ? preferredLang : "text";
    const shikiHtml = highlighter.codeToHtml(sourceCode, {
      lang: resolvedLang,
      theme,
    });

    const wrapper = document.createElement("div");
    wrapper.innerHTML = shikiHtml;
    const shikiPre = wrapper.querySelector("pre");
    if (!shikiPre) continue;
    shikiPre.dataset.language = resolvedLang;
    shikiPre.classList.add("code__block");
    const shikiCode = shikiPre.querySelector("code");
    if (shikiCode) shikiCode.dataset.shikiReady = "true";
    preElement.replaceWith(shikiPre);
  }
};

const enhanceMessageBody = async (messageElement) => {
  if (!messageElement) return;
  normalizeShortCodeBlocks(messageElement);
  applyContentGroupCards(messageElement);
  await applyShikiToMessageCodeBlocks(messageElement);
  addCopyButtonToCodeBlocks(messageElement);
};

const renderPendingAttachments = () => {
  if (!attachmentList) return;
  if (pendingAttachments.length === 0) {
    attachmentList.innerHTML = "";
    updateChatBottomSafeSpace();
    return;
  }

  attachmentList.innerHTML = pendingAttachments
    .map(
      (file, index) => `
        <div class="prompt__attachment">
          <span class="prompt__attachment-name" title="${escapeHtml(file.name)}">
            ${escapeHtml(file.name)} · ${formatBytes(file.size)}
          </span>
          <button
            type="button"
            class="prompt__attachment-remove"
            data-attachment-index="${index}"
            aria-label="移除附件"
            title="移除"
          >
            ×
          </button>
        </div>
      `
    )
    .join("");
  updateChatBottomSafeSpace();
};

const isProbablyTextFile = (file) => {
  const type = (file?.type || "").toLowerCase();
  if (type.startsWith("text/")) return true;
  if (
    [
      "application/json",
      "application/javascript",
      "application/xml",
      "application/x-yaml",
    ].includes(type)
  ) {
    return true;
  }
  const name = (file?.name || "").toLowerCase();
  return /\.(txt|md|json|csv|js|ts|tsx|go|py|java|c|cpp|html|css|xml|ya?ml)$/i.test(
    name
  );
};

const readFileAsText = (file) =>
  new Promise((resolve, reject) => {
    const reader = new FileReader();
    reader.onload = () => resolve(typeof reader.result === "string" ? reader.result : "");
    reader.onerror = () => reject(new Error(`读取文件失败: ${file.name}`));
    reader.readAsText(file);
  });

const buildMessageWithAttachments = async (messageText, attachments) => {
  const baseText = (messageText || "").trim();
  if (!attachments || attachments.length === 0) return baseText;

  const attachmentSections = [];
  for (const file of attachments) {
    if (file.type?.startsWith("image/")) {
      attachmentSections.push(`- 图片: ${file.name} (${formatBytes(file.size)})`);
      continue;
    }

    if (isProbablyTextFile(file)) {
      try {
        const fullText = await readFileAsText(file);
        const clippedText = fullText.slice(0, MAX_ATTACHMENT_TEXT_CHARS);
        const clippedSuffix = fullText.length > MAX_ATTACHMENT_TEXT_CHARS ? "\n...[内容已截断]" : "";
        attachmentSections.push(
          `- 文件: ${file.name} (${formatBytes(file.size)})\n\`\`\`\n${clippedText}${clippedSuffix}\n\`\`\``
        );
      } catch {
        attachmentSections.push(`- 文件: ${file.name} (${formatBytes(file.size)}) [读取失败]`);
      }
      continue;
    }

    attachmentSections.push(
      `- 文件: ${file.name} (${formatBytes(file.size)}) [二进制文件，未内联内容]`
    );
  }

  const prefix = baseText || "请结合以下附件信息进行回答：";
  return `${prefix}\n\n[附件]\n${attachmentSections.join("\n")}`.trim();
};

const getActiveScrollElement = () => {
  const canScrollInChats =
    chatHistoryContainer.scrollHeight > chatHistoryContainer.clientHeight + 1;
  return canScrollInChats ? chatHistoryContainer : pageScrollRoot;
};

const isActiveScrollNearBottom = () => {
  const scrollElement = getActiveScrollElement();
  const distanceToBottom =
    scrollElement.scrollHeight -
    scrollElement.scrollTop -
    scrollElement.clientHeight;
  return distanceToBottom <= SCROLL_FOLLOW_THRESHOLD;
};

const scrollChatsToBottom = (behavior = "smooth", force = false) => {
  if (!force && !shouldAutoScroll) return;
  const scrollElement = getActiveScrollElement();
  const targetTop = scrollElement.scrollHeight;

  if (scrollElement === pageScrollRoot) {
    window.scrollTo({ top: targetTop, behavior });
    return;
  }

  scrollElement.scrollTo({ top: targetTop, behavior });
};

const updateChatBottomSafeSpace = () => {
  if (!themeRoot || !promptContainer) return;
  if (!document.body.classList.contains("hide-header")) {
    themeRoot.style.setProperty("--chat-bottom-safe-space", "0px");
    return;
  }
  const promptRect = promptContainer.getBoundingClientRect();
  const occupiedFromBottom = Math.max(0, window.innerHeight - promptRect.top);
  const attachmentsHeight = attachmentList ? Math.ceil(attachmentList.getBoundingClientRect().height) : 0;
  const spacingBuffer = attachmentsHeight > 0 ? 18 : 10;
  const nextInset = Math.max(72, Math.ceil(occupiedFromBottom) + attachmentsHeight + spacingBuffer);
  themeRoot.style.setProperty("--chat-bottom-safe-space", `${nextInset}px`);
};

const adjustPromptInputHeight = () => {
  if (!promptInput) return;
  promptInput.style.height = "auto";
  const nextHeight = Math.min(
    Math.max(promptInput.scrollHeight, PROMPT_INPUT_MIN_HEIGHT),
    PROMPT_INPUT_MAX_HEIGHT
  );
  const isExpanded = nextHeight > PROMPT_INPUT_MIN_HEIGHT + 2;
  promptInput.style.height = `${nextHeight}px`;
  promptInput.style.overflowY =
    promptInput.scrollHeight > PROMPT_INPUT_MAX_HEIGHT ? "auto" : "hidden";
  promptInput.classList.toggle("prompt__form-input--expanded", isExpanded);
  const shouldLiftHeader = !document.body.classList.contains("hide-header");
  const headerLiftOffset = shouldLiftHeader
    ? Math.max(0, Math.round((nextHeight - PROMPT_INPUT_MIN_HEIGHT) * 0.9))
    : 0;
  themeRoot.style.setProperty("--prompt-expand-shift", `${headerLiftOffset}px`);
  updateChatBottomSafeSpace();
};

const extractReasoningAndContentFromMessage = (responseMessage = {}) => {
  const rawContent = typeof responseMessage?.content === "string"
    ? responseMessage.content
    : "";

  const reasoningCandidates = [
    responseMessage?.reasoning_content,
    responseMessage?.reasoning,
    responseMessage?.reasoningContent,
  ];

  const reasoningParts = [];
  for (const candidate of reasoningCandidates) {
    if (typeof candidate === "string" && candidate.trim()) {
      reasoningParts.push(candidate.trim());
      break;
    }
  }

  let cleanedContent = rawContent;
  const firstThinkTagIndex = rawContent.indexOf("<think>");
  const lastThinkEndTagIndex = rawContent.lastIndexOf("</think>");

  if (firstThinkTagIndex !== -1 && lastThinkEndTagIndex > firstThinkTagIndex) {
    const thinkBlockStart = firstThinkTagIndex + "<think>".length;
    const thinkBlock = rawContent
      .slice(thinkBlockStart, lastThinkEndTagIndex)
      .replace(/^\s*<think>\s*/i, "")
      .replace(/\s*<\/think>\s*$/i, "")
      .trim();

    if (thinkBlock) {
      reasoningParts.push(thinkBlock);
    }

    cleanedContent = (
      rawContent.slice(0, firstThinkTagIndex) +
      rawContent.slice(lastThinkEndTagIndex + "</think>".length)
    ).trim();
  } else {
    const tagMatches = Array.from(rawContent.matchAll(THINK_TAG_PATTERN));
    for (const match of tagMatches) {
      const chunk = (match?.[1] || "").trim();
      if (chunk) {
        reasoningParts.push(chunk);
      }
    }
    cleanedContent = rawContent.replace(THINK_TAG_PATTERN, "").trim();
  }

  cleanedContent = cleanedContent.replace(/<\/?think>/gi, "").trim();
  const dedupedReasoning = [...new Set(reasoningParts)].join("\n\n").trim();

  return {
    content: cleanedContent,
    reasoning: dedupedReasoning,
  };
};

const initReasoningPanelToggle = (incomingMessageElement) => {
  const reasoningPanel = incomingMessageElement.querySelector(".message__reasoning");
  const reasoningTextElement = incomingMessageElement.querySelector(".message__reasoning-text");
  if (!reasoningPanel || reasoningPanel.dataset.bound === "true") return;

  reasoningPanel.dataset.bound = "true";
  reasoningPanel.dataset.autoFollow = "true";
  reasoningPanel.dataset.userCollapsed = "false";
  reasoningPanel.addEventListener("click", () => {
    if (reasoningPanel.dataset.collapsible === "false") return;
    const selection = window.getSelection();
    const hasSelectedReasoningText =
      !!selection &&
      !selection.isCollapsed &&
      reasoningPanel.contains(selection.anchorNode) &&
      reasoningPanel.contains(selection.focusNode) &&
      !!selection.toString().trim();
    if (hasSelectedReasoningText) return;
    const collapsed = reasoningPanel.classList.toggle("message__reasoning--collapsed");
    reasoningPanel.dataset.expanded = collapsed ? "false" : "true";
    reasoningPanel.dataset.userCollapsed = collapsed ? "true" : "false";
    scrollChatsToBottom("smooth");
  });

  if (!reasoningTextElement) return;
  reasoningTextElement.addEventListener("scroll", () => {
    if (reasoningTextElement.dataset.internalScrollSync === "true") return;
    const distanceToBottom =
      reasoningTextElement.scrollHeight -
      reasoningTextElement.scrollTop -
      reasoningTextElement.clientHeight;
    reasoningPanel.dataset.autoFollow =
      distanceToBottom <= REASONING_SCROLL_FOLLOW_THRESHOLD ? "true" : "false";
  });
};

const renderReasoningPanel = (
  incomingMessageElement,
  reasoningText = "",
  options = {}
) => {
  const { collapseByDefault = true, streaming = false } = options;
  const reasoningPanel = incomingMessageElement.querySelector(".message__reasoning");
  const reasoningTextElement = incomingMessageElement.querySelector(
    ".message__reasoning-text"
  );
  if (!reasoningPanel || !reasoningTextElement) return;
  if (!ENABLE_REASONING_OUTPUT) {
    reasoningPanel.classList.add("hide");
    reasoningTextElement.innerHTML = "";
    reasoningPanel.dataset.collapsible = "false";
    reasoningPanel.dataset.expanded = "false";
    return;
  }

  initReasoningPanelToggle(incomingMessageElement);

  const trimmedReasoning = reasoningText.trim();
  const shouldAutoFollow = reasoningPanel.dataset.autoFollow !== "false";
  const previousScrollTop = reasoningTextElement.scrollTop;
  if (!trimmedReasoning) {
    reasoningPanel.classList.remove("hide");
    reasoningPanel.classList.add("message__reasoning--collapsed");
    reasoningTextElement.innerHTML = `<p>本轮没有可展示的思考内容。</p>`;
    reasoningPanel.dataset.collapsible = "false";
    reasoningPanel.dataset.expanded = "false";
    if (!streaming) {
      reasoningPanel.dataset.userCollapsed = "true";
    } else {
      delete reasoningPanel.dataset.userCollapsed;
    }
    reasoningPanel.classList.add("message__reasoning--empty");
    return;
  }

  reasoningPanel.dataset.collapsible = "true";
  reasoningPanel.classList.remove("message__reasoning--empty");
  reasoningPanel.classList.remove("message__reasoning--loading");
  reasoningPanel.classList.remove("hide");
  const hasUserCollapsedPreference = reasoningPanel.dataset.userCollapsed === "true";
  const hasUserExpandedPreference = reasoningPanel.dataset.userCollapsed === "false";
  const hasInitializedState = reasoningPanel.dataset.expanded === "true" || reasoningPanel.dataset.expanded === "false";

  if (!hasInitializedState) {
    const shouldCollapseByDefault = collapseByDefault;
    reasoningPanel.classList.toggle("message__reasoning--collapsed", shouldCollapseByDefault);
    reasoningPanel.dataset.expanded = shouldCollapseByDefault ? "false" : "true";
    reasoningPanel.dataset.userCollapsed = shouldCollapseByDefault ? "true" : "false";
  } else if (hasUserCollapsedPreference) {
    reasoningPanel.classList.add("message__reasoning--collapsed");
    reasoningPanel.dataset.expanded = "false";
  } else if (!hasUserExpandedPreference && collapseByDefault) {
    reasoningPanel.classList.add("message__reasoning--collapsed");
    reasoningPanel.dataset.expanded = "false";
  } else {
    reasoningPanel.classList.remove("message__reasoning--collapsed");
    reasoningPanel.dataset.expanded = "true";
  }
  reasoningTextElement.innerHTML = marked.parse(trimmedReasoning);
  if (shouldAutoFollow) {
    reasoningTextElement.dataset.internalScrollSync = "true";
    reasoningTextElement.scrollTop = reasoningTextElement.scrollHeight;
    reasoningTextElement.dataset.internalScrollSync = "false";
    reasoningPanel.dataset.autoFollow = "true";
  } else {
    reasoningTextElement.dataset.internalScrollSync = "true";
    reasoningTextElement.scrollTop = previousScrollTop;
    reasoningTextElement.dataset.internalScrollSync = "false";
  }
};

const showReasoningLoading = (incomingMessageElement) => {
  const reasoningPanel = incomingMessageElement.querySelector(".message__reasoning");
  if (!reasoningPanel) return;
  if (!ENABLE_REASONING_OUTPUT) {
    reasoningPanel.classList.add("hide");
    return;
  }
  reasoningPanel.dataset.collapsible = "false";
  delete reasoningPanel.dataset.expanded;
  delete reasoningPanel.dataset.userCollapsed;
  reasoningPanel.classList.add("hide");
};

// Play opening title typing animation on each refresh
const playHeaderTypingAnimation = () => {
  if (!headerTypingTitle || !headerTypingText) return;

  const fullText =
    headerTypingTitle.dataset.fullText || headerTypingText.textContent || "";
  const characters = Array.from(fullText);
  const totalDuration = 500;
  const typingDelay = Math.max(totalDuration / Math.max(characters.length, 1), 1);

  headerTypingText.textContent = "";

  characters.forEach((char, index) => {
    window.setTimeout(() => {
      headerTypingText.textContent += char;
    }, typingDelay * (index + 1));
  });
};

// Pause header cursor while prompt input is focused
const setHeaderCursorPaused = (paused) => {
  if (!headerCursor) return;
  headerCursor.classList.toggle("header__cursor--paused", paused);
};

const isMobileViewport = () => window.matchMedia(`(max-width: ${MOBILE_BREAKPOINT}px)`).matches;

const closeSidebarDrawer = () => {
  document.body.classList.remove("sidebar-drawer-open");
};

const openSidebarDrawer = () => {
  if (!isMobileViewport()) return;
  document.body.classList.add("sidebar-drawer-open");
};

const setSidebarCollapsed = (collapsed) => {
  if (isMobileViewport()) return;
  document.body.classList.toggle("sidebar-collapsed", collapsed);
  sidebarElement?.classList.toggle("app-sidebar--collapsed", collapsed);
  localStorage.setItem("sidebarCollapsed", collapsed ? "true" : "false");
};

const getSessionTitleFromText = (text = "") => {
  const normalized = text.replace(/\s+/g, " ").trim();
  if (!normalized) return "新聊天";
  return normalized.length > 18 ? `${normalized.slice(0, 18)}...` : normalized;
};

const createSession = (title = "新聊天", options = {}) => ({
  id: options.id || `local-${Date.now()}-${Math.random().toString(36).slice(2, 7)}`,
  title,
  messages: options.messages || [],
  loaded: options.loaded ?? true,
});

const normalizeServerSession = (session = {}) =>
  createSession(session.title || "新聊天", {
    id: String(session.id),
    messages: [],
    loaded: false,
  });

const getActiveSession = () =>
  chatSessions.find((session) => session.id === activeSessionId) || null;

const getPersistedSessionId = (sessionId = "") => {
  const numericId = Number(sessionId);
  if (!Number.isFinite(numericId) || numericId <= 0) return null;
  return Math.trunc(numericId);
};

const renderSidebarSessions = () => {
  if (!sidebarHistory) return;
  sidebarHistory.innerHTML = chatSessions
    .map(
      (session) => `
        <div class="sidebar__session-item ${session.id === activeSessionId ? "is-active" : ""}">
          <button
            type="button"
            class="sidebar__session ${session.id === activeSessionId ? "is-active" : ""}"
            data-session-id="${session.id}"
            title="${escapeHtml(session.title)}"
          >
            <i class="bx bx-message-rounded-dots sidebar__session-icon"></i>
            <span class="sidebar__session-title">${escapeHtml(session.title)}</span>
          </button>
          <button
            type="button"
            class="sidebar__session-delete"
            data-delete-session-id="${session.id}"
            title="删除会话"
            aria-label="删除会话"
          >
            <i class="bx bx-trash"></i>
          </button>
        </div>
      `
    )
    .join("");
};

const setActiveSession = (sessionId) => {
  if (!sessionId) return;
  activeSessionId = sessionId;
  renderSidebarSessions();
};

const updateActiveSessionTitle = (nextTitle) => {
  if (!activeSessionId || !nextTitle) return;
  chatSessions = chatSessions.map((session) =>
    session.id === activeSessionId ? { ...session, title: nextTitle } : session
  );
  renderSidebarSessions();
};

const appendMessageToActiveSession = (message) => {
  if (!activeSessionId || !message) return;
  appendMessageToSession(activeSessionId, message);
};

const appendMessageToSession = (sessionId, message) => {
  if (!sessionId || !message) return;
  chatSessions = chatSessions.map((session) => {
    if (session.id !== sessionId) return session;
    return {
      ...session,
      messages: [...(session.messages || []), message],
      loaded: true,
    };
  });
};

const updateStreamingAssistantInSession = (sessionId, patch = {}) => {
  if (!sessionId) return;
  chatSessions = chatSessions.map((session) => {
    if (session.id !== sessionId) return session;
    const messages = [...(session.messages || [])];
    for (let i = messages.length - 1; i >= 0; i -= 1) {
      if (messages[i]?.role === "assistant" && messages[i]?.streaming) {
        messages[i] = { ...messages[i], ...patch };
        break;
      }
    }
    return { ...session, messages };
  });
};

const finalizeStreamingAssistantInSession = (sessionId, finalMessage = {}) => {
  if (!sessionId) return;
  chatSessions = chatSessions.map((session) => {
    if (session.id !== sessionId) return session;
    const messages = [...(session.messages || [])];
    for (let i = messages.length - 1; i >= 0; i -= 1) {
      if (messages[i]?.role === "assistant" && messages[i]?.streaming) {
        const nextMessage = { ...messages[i], ...finalMessage };
        delete nextMessage.streaming;
        messages[i] = nextMessage;
        break;
      }
    }
    return { ...session, messages, loaded: true };
  });
};

const isSessionGenerating = (sessionId) => {
  if (!sessionId) return false;
  return sessionGenerationState.get(sessionId)?.generating === true;
};

const isActiveSessionGenerating = () => isSessionGenerating(activeSessionId);

const setSessionGenerating = (sessionId, generating, extra = {}) => {
  if (!sessionId) return;
  if (generating) {
    sessionGenerationState.set(sessionId, { generating: true, ...extra });
    return;
  }
  sessionGenerationState.delete(sessionId);
};

const registerSessionStreamDom = (sessionId, incomingMessageElement, messageTextElement) => {
  if (!sessionId || !incomingMessageElement || !messageTextElement) return;
  const existing = sessionGenerationState.get(sessionId) || { generating: true };
  sessionGenerationState.set(sessionId, {
    ...existing,
    generating: true,
    incomingMessageElement,
    messageTextElement,
  });
};

const getSessionStreamDom = (sessionId) => {
  const state = sessionGenerationState.get(sessionId);
  if (!state?.incomingMessageElement || !state?.messageTextElement) return null;
  if (!document.contains(state.incomingMessageElement)) return null;
  return {
    incomingMessageElement: state.incomingMessageElement,
    messageTextElement: state.messageTextElement,
  };
};

const remapSessionGenerationState = (oldId, newId) => {
  if (!oldId || !newId || oldId === newId) return;
  const existing = sessionGenerationState.get(oldId);
  if (!existing) return;
  sessionGenerationState.delete(oldId);
  sessionGenerationState.set(newId, existing);
};

const appendAssistantToActiveSession = (content = "", reasoning = "") => {
  appendMessageToActiveSession({
    role: "assistant",
    content,
    reasoning_content: reasoning,
  });
};

const appendAssistantResponsesToActiveSession = (responses = [], selectedModel = "") => {
  const cleanedResponses = (responses || [])
    .map((item) => ({
      model: String(item?.model || "").trim().toLowerCase(),
      content: item?.content || "",
      reasoning_content: item?.reasoning_content || "",
      error: item?.error || "",
    }))
    .filter((item) => item.model && (item.content || item.reasoning_content || item.error));
  if (cleanedResponses.length === 0) {
    appendAssistantToActiveSession("", "");
    return;
  }
  const selected = cleanedResponses.find((item) => item.model === selectedModel) || cleanedResponses[0];
  appendMessageToActiveSession({
    role: "assistant",
    model: selected.model,
    content: selected.content,
    reasoning_content: selected.reasoning_content || "",
    selected_model: selected.model,
    model_responses: cleanedResponses,
  });
};

const extractAssistantModelResponsesFromSession = (serverSession = null, fallbackModels = []) => {
  const messages = Array.isArray(serverSession?.messages) ? serverSession.messages : [];
  const latestAssistantMessage = [...messages]
    .reverse()
    .find((item) => String(item?.role || "").toLowerCase() === "assistant");

  const candidateResponses = [];
  if (Array.isArray(latestAssistantMessage?.model_responses)) {
    candidateResponses.push(...latestAssistantMessage.model_responses);
  }
  if (Array.isArray(serverSession?.model_responses)) {
    candidateResponses.push(...serverSession.model_responses);
  }

  const normalized = candidateResponses
    .map((item, index) => ({
      model: String(item?.model || fallbackModels[index] || fallbackModels[0] || DEFAULT_MODEL_KEY)
        .trim()
        .toLowerCase(),
      content: item?.content || "",
      reasoning_content: item?.reasoning_content || "",
      error: item?.error || "",
    }))
    .filter((item) => item.model && (item.content || item.reasoning_content || item.error));

  if (normalized.length > 0) return normalized;
  if (!latestAssistantMessage) return [];

  const fallbackSingle = {
    model: String(latestAssistantMessage?.model || fallbackModels[0] || DEFAULT_MODEL_KEY)
      .trim()
      .toLowerCase(),
    content: latestAssistantMessage?.content || "",
    reasoning_content: latestAssistantMessage?.reasoning_content || "",
  };
  return fallbackSingle.content || fallbackSingle.reasoning_content ? [fallbackSingle] : [];
};

const syncSessionFromServer = (serverSession = null, targetSessionId = null, sessionRef = null) => {
  if (!serverSession?.id) return;
  const nextId = String(serverSession.id);
  const lookupId = targetSessionId ?? activeSessionId;
  if (!lookupId) return;

  const exists = chatSessions.some((session) => session.id === nextId);
  chatSessions = chatSessions.map((session) => {
    if (session.id !== lookupId) return session;
    return {
      ...session,
      id: nextId,
      title: serverSession.title || session.title,
      loaded: true,
    };
  });
  if (lookupId !== nextId && exists) {
    chatSessions = chatSessions.filter((session) => session.id !== lookupId);
  }
  remapSessionGenerationState(lookupId, nextId);
  if (sessionRef && sessionRef.id === lookupId) {
    sessionRef.id = nextId;
  }
  if (activeSessionId === lookupId) {
    activeSessionId = nextId;
  }
  renderSidebarSessions();
};

const syncActiveSessionFromServer = (serverSession = null) => {
  syncSessionFromServer(serverSession, activeSessionId);
};

const renderOutgoingMessage = (text) => {
  const outgoingAvatarSrc = normalizeAvatarUrl(currentUser?.avatar_url || "");
  const outgoingMessageHtml = `
        <div class="message__content">
            <img class="message__avatar" src="${outgoingAvatarSrc}" alt="User avatar">
            <div class="message__text"></div>
        </div>
    `;
  const outgoingMessageElement = createChatMessageElement(
    outgoingMessageHtml,
    "message--outgoing"
  );
  outgoingMessageElement.querySelector(".message__text").innerText = text || "";
  chatHistoryContainer.appendChild(outgoingMessageElement);
};

const createIncomingMessageHtml = () => `
        <div class="message__content">
            <img class="message__avatar" src="assets/YoooFind.png" alt="Gemini avatar">
            <div class="message__body">
                <div class="message__model-track hide">
                    <div class="message__model-track-label">回答轨道</div>
                    <div class="message__model-tabs"></div>
                </div>
                <div class="message__reasoning hide">
                    <div class="message__reasoning-header">
                        <div class="message__reasoning-title">深度思考</div>
                    </div>
                    <div class="message__reasoning-text"></div>
                </div>
                <div class="message__text"></div>
            </div>
        </div>
        <div class="message__actions">
            <button type="button" class="message__action-btn" data-action="copy" title="复制"><i class='bx bx-copy-alt'></i></button>
            <button type="button" class="message__action-btn" data-action="like" title="点赞"><i class='bx bx-like'></i></button>
            <button type="button" class="message__action-btn" data-action="dislike" title="点踩"><i class='bx bx-dislike'></i></button>
            <button type="button" class="message__action-btn" data-action="retry" title="重试"><i class='bx bx-refresh'></i></button>
        </div>
    `;

const pickModelResponse = (responses = [], preferredModel = "") => {
  const normalizedPreferred = String(preferredModel || "").trim().toLowerCase();
  if (normalizedPreferred) {
    const matched = responses.find((item) => item.model === normalizedPreferred);
    if (matched) return matched;
  }
  return responses[0] || {
    model: DEFAULT_MODEL_KEY,
    content: "",
    reasoning_content: "",
  };
};

const normalizeModelResponses = (responses = [], fallbackModels = []) =>
  (responses || [])
    .map((item, index) => ({
      model: String(item?.model || fallbackModels[index] || fallbackModels[0] || DEFAULT_MODEL_KEY)
        .trim()
        .toLowerCase(),
      content: item?.content || "",
      reasoning_content: item?.reasoning_content || "",
      error: item?.error || "",
    }))
    .filter((item) => item.model && (item.content || item.reasoning_content || item.error));

const renderAssistantPayloadIntoElement = (incomingMessageElement, response = {}) => {
  const messageTextElement = incomingMessageElement.querySelector(".message__text");
  if (!messageTextElement) return;
  const fallbackErrorText = response.error ? `该模型回答失败：${response.error}` : "";
  messageTextElement.innerHTML = marked.parse(response.content || fallbackErrorText);
  renderReasoningPanel(incomingMessageElement, response.reasoning_content || "", {
    collapseByDefault: true,
  });
  void enhanceMessageBody(messageTextElement);
  scrollChatsToBottom("auto");
};

const setupAssistantModelTrack = (
  incomingMessageElement,
  responses = [],
  selectedModel = ""
) => {
  const trackElement = incomingMessageElement.querySelector(".message__model-track");
  const tabsElement = incomingMessageElement.querySelector(".message__model-tabs");
  if (!trackElement || !tabsElement) return;
  if (!Array.isArray(responses) || responses.length <= 1) {
    trackElement.classList.add("hide");
    return;
  }

  trackElement.classList.remove("hide");
  const picked = pickModelResponse(responses, selectedModel);
  tabsElement.innerHTML = responses
    .map(
      (item) => `
      <button
        type="button"
        class="message__model-tab ${item.model === picked.model ? "is-active" : ""} ${item.error ? "is-error" : ""}"
        data-model-tab="${item.model}"
      >
        ${escapeHtml(getModelLabel(item.model))}${item.error ? '<span class="message__model-tab-error-mark">!</span>' : ""}
      </button>
    `
    )
    .join("");

  tabsElement.addEventListener("click", (event) => {
    const tab = event.target.closest("[data-model-tab]");
    if (!tab) return;
    const modelKey = tab.dataset.modelTab;
    const activeResponse = pickModelResponse(responses, modelKey);
    tabsElement.querySelectorAll(".message__model-tab").forEach((button) => {
      button.classList.toggle("is-active", button.dataset.modelTab === activeResponse.model);
    });
    renderAssistantPayloadIntoElement(incomingMessageElement, activeResponse);
  });
};

const renderStreamingAssistantShell = (message = {}, options = {}) => {
  const loadingHtml = `
        <div class="message__content">
            <img class="message__avatar" src="assets/YoooFind.png" alt="Gemini avatar">
            <div class="message__body">
                <div class="message__thinking-status">${escapeHtml(options.statusText || "正在生成回答...")}</div>
                <div class="message__model-track hide">
                    <div class="message__model-track-label">回答轨道</div>
                    <div class="message__model-tabs"></div>
                </div>
                <div class="message__reasoning hide">
                    <div class="message__reasoning-header">
                        <div class="message__reasoning-title">深度思考</div>
                    </div>
                    <div class="message__reasoning-text"></div>
                </div>
                <div class="message__text"></div>
            </div>
        </div>
        <div class="message__actions hide">
            <button type="button" class="message__action-btn" data-action="copy" title="复制"><i class='bx bx-copy-alt'></i></button>
            <button type="button" class="message__action-btn" data-action="like" title="点赞"><i class='bx bx-like'></i></button>
            <button type="button" class="message__action-btn" data-action="dislike" title="点踩"><i class='bx bx-dislike'></i></button>
            <button type="button" class="message__action-btn" data-action="retry" title="重试"><i class='bx bx-refresh'></i></button>
        </div>
    `;

  const incomingMessageElement = createChatMessageElement(
    loadingHtml,
    "message--incoming",
    "message--loading"
  );
  chatHistoryContainer.appendChild(incomingMessageElement);
  const messageTextElement = incomingMessageElement.querySelector(".message__text");
  const modelResponses = Array.isArray(message.model_responses) ? message.model_responses : [];
  if (modelResponses.length > 1) {
    setupAssistantModelTrack(
      incomingMessageElement,
      modelResponses,
      message.selected_model || message.model || modelResponses[0]?.model
    );
    const active = pickModelResponse(
      modelResponses,
      message.selected_model || message.model || modelResponses[0]?.model
    );
    renderAssistantPayloadIntoElement(incomingMessageElement, active);
  } else if (message.content || message.reasoning_content) {
    messageTextElement.innerHTML = marked.parse(message.content || "");
    renderReasoningPanel(incomingMessageElement, message.reasoning_content || "", {
      collapseByDefault: false,
      streaming: true,
    });
    void enhanceMessageBody(messageTextElement);
  } else {
    showReasoningLoading(incomingMessageElement);
  }
  return { incomingMessageElement, messageTextElement };
};

const renderAssistantMessage = (content = "", reasoning = "", options = {}) => {
  const normalizedResponses = Array.isArray(options.modelResponses)
    ? options.modelResponses
      .map((item) => ({
        model: String(item?.model || "").trim().toLowerCase(),
        content: item?.content || "",
        reasoning_content: item?.reasoning_content || "",
        error: item?.error || "",
      }))
      .filter((item) => item.model && (item.content || item.reasoning_content || item.error))
    : [];

  const incomingMessageElement = createChatMessageElement(
    createIncomingMessageHtml(),
    "message--incoming"
  );
  chatHistoryContainer.appendChild(incomingMessageElement);
  if (normalizedResponses.length > 0) {
    const active = pickModelResponse(
      normalizedResponses,
      options.selectedModel || normalizedResponses[0]?.model
    );
    setupAssistantModelTrack(
      incomingMessageElement,
      normalizedResponses,
      active.model
    );
    renderAssistantPayloadIntoElement(incomingMessageElement, active);
    return;
  }

  const messageTextElement = incomingMessageElement.querySelector(".message__text");
  const parsedResponse = marked.parse(content || "");
  showTypingEffect(
    content || "",
    parsedResponse,
    messageTextElement,
    incomingMessageElement,
    true,
    reasoning || ""
  );
};

const renderActiveSessionMessages = () => {
  const session = getActiveSession();
  clearChatHistoryDom();
  if (!session || !Array.isArray(session.messages)) return;
  session.messages.forEach((message) => {
    if (message.role === "assistant") {
      if (message.streaming) {
        const { incomingMessageElement, messageTextElement } = renderStreamingAssistantShell(message, {
          statusText: isSessionGenerating(session.id) ? "正在生成回答..." : "回答未完成",
        });
        if (isSessionGenerating(session.id)) {
          registerSessionStreamDom(session.id, incomingMessageElement, messageTextElement);
        }
        return;
      }
      renderAssistantMessage(message.content || "", message.reasoning_content || "", {
        modelResponses: message.model_responses || [],
        selectedModel: message.selected_model || message.model || "",
      });
      return;
    }
    renderOutgoingMessage(message.content || "");
  });
  if (session.messages.length > 0) {
    document.body.classList.add("hide-header");
    updateChatBottomSafeSpace();
  }
  scrollChatsToBottom("auto", true);
};

const fetchSessionMessages = async (sessionId) => {
  const persistedId = getPersistedSessionId(sessionId);
  if (!persistedId) {
    renderActiveSessionMessages();
    return;
  }
  const response = await authFetch(`${CHAT_SESSIONS_URL}/${persistedId}`, { method: "GET" });
  if (response.status === 401) {
    logout();
    return;
  }
  if (!response.ok) {
    throw new Error(await parseErrorMessage(response, "加载会话失败"));
  }
  const data = await response.json();
  const messages = Array.isArray(data?.messages) ? data.messages : [];
  chatSessions = chatSessions.map((session) =>
    session.id === String(persistedId)
      ? {
        ...session,
        title: data?.session?.title || session.title,
        messages,
        loaded: true,
      }
      : session
  );
  renderSidebarSessions();
  if (activeSessionId === String(persistedId)) {
    renderActiveSessionMessages();
  }
};

const loadRemoteChatSessions = async () => {
  if (!authToken) return;
  const response = await authFetch(CHAT_SESSIONS_URL, { method: "GET" });
  if (response.status === 401) {
    logout();
    return;
  }
  if (!response.ok) {
    throw new Error(await parseErrorMessage(response, "加载历史会话失败"));
  }
  const data = await response.json();
  const sessions = (Array.isArray(data?.sessions) ? data.sessions : [])
    .slice(0, MAX_CLOUD_SESSIONS)
    .map((session) => normalizeServerSession(session));
  const freshSession = createSession("新聊天");
  if (sessions.length === 0) {
    chatSessions = [freshSession];
    setActiveSession(chatSessions[0].id);
    renderActiveSessionMessages();
    return;
  }
  // Always land on a fresh chat after reload/login, while keeping history available in sidebar.
  chatSessions = [freshSession, ...sessions];
  setActiveSession(freshSession.id);
  renderActiveSessionMessages();
};

const clearChatHistoryDom = () => {
  chatHistoryContainer.innerHTML = "";
  document.body.classList.remove("hide-header");
  shouldAutoScroll = true;
};

const resetChatCanvas = () => {
  clearChatHistoryDom();
  pendingAttachments = [];
  renderPendingAttachments();
  messageForm.reset();
  if (fileInput) fileInput.value = "";
  adjustPromptInputHeight();
};

const handleCreateNewChat = () => {
  const currentSession = chatSessions.find((session) => session.id === activeSessionId);
  const isAlreadyFreshChat =
    !!currentSession &&
    currentSession.title === "新聊天" &&
    (currentSession.messages || []).length === 0;
  if (isAlreadyFreshChat) {
    if (isMobileViewport()) closeSidebarDrawer();
    return;
  }
  const existingFreshSession = chatSessions.find(
    (session) => session.title === "新聊天" && (session.messages || []).length === 0
  );
  if (existingFreshSession) {
    // Reuse the only empty "新聊天" session so the sidebar never shows duplicates.
    chatSessions = [
      existingFreshSession,
      ...chatSessions.filter((session) => session.id !== existingFreshSession.id),
    ];
    setActiveSession(existingFreshSession.id);
    resetChatCanvas();
    if (isMobileViewport()) closeSidebarDrawer();
    return;
  }

  const newSession = createSession("新聊天");
  chatSessions = [newSession, ...chatSessions];
  setActiveSession(newSession.id);
  resetChatCanvas();
  if (isMobileViewport()) closeSidebarDrawer();
};

const fetchCurrentUser = async () => {
  if (!authToken) return null;
  const response = await authFetch(config.ME_URL, { method: "GET" });
  if (!response.ok) {
    if (response.status === 401) {
      logout();
      return null;
    }
    throw new Error(await parseErrorMessage(response, "获取用户信息失败"));
  }
  currentUser = await response.json();
  persistCurrentUserProfile();
  applyUserProfileToUI();
  return currentUser;
};

const deleteSession = async (sessionId) => {
  if (!sessionId) return;
  const session = chatSessions.find((item) => item.id === sessionId);
  if (!session) return;

  const persistedId = getPersistedSessionId(sessionId);
  if (persistedId && authToken) {
    const response = await authFetch(`${CHAT_SESSIONS_URL}/${persistedId}`, { method: "DELETE" });
    if (response.status === 401) {
      logout();
      return;
    }
    if (!response.ok) {
      throw new Error(await parseErrorMessage(response, "删除历史会话失败"));
    }
  }

  chatSessions = chatSessions.filter((item) => item.id !== sessionId);
  if (chatSessions.length === 0) {
    const newSession = createSession("新聊天");
    chatSessions = [newSession];
    setActiveSession(newSession.id);
    renderActiveSessionMessages();
    return;
  }

  if (activeSessionId === sessionId) {
    const fallbackSession = chatSessions.find(
      (item) => item.title === "新聊天" && (item.messages || []).length === 0
    ) || chatSessions[0];
    setActiveSession(fallbackSession.id);
    if (fallbackSession.loaded) {
      renderActiveSessionMessages();
    } else {
      await fetchSessionMessages(fallbackSession.id);
    }
    return;
  }

  renderSidebarSessions();
};

const handleAuthSubmit = async (event) => {
  event.preventDefault();
  const email = authEmailInput?.value?.trim() || "";
  const password = authPasswordInput?.value || "";
  if (!email || !password) {
    setAuthStatusText("请填写邮箱和密码", true);
    return;
  }
  const url = authMode === "register" ? config.AUTH_REGISTER_URL : config.AUTH_LOGIN_URL;
  setAuthStatusText("正在提交...");
  const response = await fetch(url, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ email, password }),
  });
  if (!response.ok) {
    setAuthStatusText(await parseErrorMessage(response, "登录失败"), true);
    return;
  }
  const data = await response.json();
  authToken = data.token || "";
  currentUser = data.user || null;
  if (!authToken || !currentUser) {
    setAuthStatusText("登录响应异常，请重试", true);
    return;
  }
  localStorage.setItem(AUTH_TOKEN_STORAGE_KEY, authToken);
  persistCurrentUserProfile();
  applyUserProfileToUI();
  try {
    await loadRemoteChatSessions();
  } catch (error) {
    console.warn("Failed to load cloud chat sessions:", error);
    chatSessions = [createSession("新聊天")];
    setActiveSession(chatSessions[0].id);
    renderActiveSessionMessages();
  }
  authForm?.reset();
  closeModal(authModal);
  setAuthStatusText("");
  void recordVisit();
};

const openProfileModal = async () => {
  if (!authToken) {
    setAuthMode("login");
    openModal(authModal);
    return;
  }
  try {
    await fetchCurrentUser();
    profileDisplayNameInput.value = currentUser?.display_name || "";
    profileFullNameInput.value = currentUser?.full_name || "";
    profileBioInput.value = currentUser?.bio || "";
    setProfileStatusText("");
    try {
      await loadUsagePanelData();
    } catch (usageError) {
      if (usageSummaryText) {
        usageSummaryText.textContent = usageError.message || "Token 数据加载失败";
      }
    }
    openModal(profileModal);
  } catch (error) {
    setProfileStatusText(error.message || "加载个人信息失败", true);
    openModal(profileModal);
  }
};

const handleProfileSubmit = async (event) => {
  event.preventDefault();
  if (!authToken) {
    logout();
    return;
  }
  const payload = {
    display_name: profileDisplayNameInput?.value?.trim() || "用户",
    full_name: profileFullNameInput?.value?.trim() || "",
    bio: profileBioInput?.value?.trim() || "",
  };
  const response = await authFetch(config.ME_URL, {
    method: "PATCH",
    body: JSON.stringify(payload),
  });
  if (!response.ok) {
    setProfileStatusText(await parseErrorMessage(response, "保存失败"), true);
    return;
  }
  currentUser = await response.json();
  persistCurrentUserProfile();
  applyUserProfileToUI();
  setProfileStatusText("");
  closeModal(profileModal);
};

const handleAvatarUpload = async (event) => {
  if (!authToken) {
    logout();
    return;
  }
  const file = event.target?.files?.[0];
  if (!file) return;
  const formData = new FormData();
  formData.append("avatar", file);
  const response = await authFetch(config.AVATAR_UPLOAD_URL, {
    method: "POST",
    body: formData,
  });
  if (!response.ok) {
    setProfileStatusText(await parseErrorMessage(response, "头像上传失败"), true);
    return;
  }
  currentUser = await response.json();
  persistCurrentUserProfile();
  applyUserProfileToUI();
  setProfileStatusText("头像已更新");
  avatarInput.value = "";
};

const updateUsageToggleUI = () => {
  if (!usageToggleGroup) return;
  usageToggleGroup.querySelectorAll("[data-usage-mode]").forEach((button) => {
    button.classList.toggle("is-active", button.dataset.usageMode === usageChartMode);
  });
};

const loadUsagePanelData = async () => {
  if (!authToken || !usageSummaryText) return;
  const [recentResponse, totalResponse] = await Promise.all([
    authFetch(`${config.USAGE_URL}?days=30`, { method: "GET" }),
    authFetch(`${config.USAGE_URL}?days=0`, { method: "GET" }),
  ]);
  if (!recentResponse.ok || !totalResponse.ok) {
    const badResponse = !recentResponse.ok ? recentResponse : totalResponse;
    throw new Error(await parseErrorMessage(badResponse, "获取 token 数据失败"));
  }

  const recentSummary = await recentResponse.json();
  const totalSummary = await totalResponse.json();
  usageChartDataCache = { recentSummary, totalSummary };

  const recentTotalTokens = recentSummary?.total?.total_tokens || 0;
  const allTimeTotalTokens = totalSummary?.total?.total_tokens || 0;
  usageSummaryText.textContent = `近30天 ${recentTotalTokens} token · 历史总计 ${allTimeTotalTokens} token`;
  renderUsageChart();
};

const buildUsageYAxisRange = (series = []) => {
  const values = series.filter((value) => Number.isFinite(value)).map((value) => Number(value));
  if (values.length === 0) {
    return { min: 0, max: 1 };
  }

  const minValue = Math.min(...values);
  const maxValue = Math.max(...values);
  if (minValue === maxValue) {
    if (maxValue === 0) {
      return { min: 0, max: 1 };
    }
    const singlePadding = Math.max(Math.abs(maxValue) * 0.3, 1);
    return {
      min: Math.max(0, minValue - singlePadding),
      max: maxValue + singlePadding,
    };
  }

  const range = maxValue - minValue;
  const padding = Math.max(range * 0.15, 1);
  return {
    min: Math.max(0, minValue - padding),
    max: maxValue + padding,
  };
};

const formatUsageDateLabel = (value) => {
  if (!value) return "";
  const normalized = String(value).trim();
  if (!normalized) return "";
  const [datePart] = normalized.split("T");
  return datePart || normalized;
};

const usageValueLabelPlugin = {
  id: "usageValueLabelPlugin",
  afterDatasetsDraw(chart) {
    const { ctx } = chart;
    const datasetMeta = chart.getDatasetMeta(0);
    const datasetValues = chart?.data?.datasets?.[0]?.data || [];
    if (!datasetMeta || !datasetMeta.data) return;

    ctx.save();
    ctx.font = "11px sans-serif";
    ctx.fillStyle = getComputedStyle(document.body).getPropertyValue("--text-secondary-color");
    ctx.textAlign = "center";
    ctx.textBaseline = "bottom";

    datasetMeta.data.forEach((point, index) => {
      const value = datasetValues[index];
      if (!Number.isFinite(value)) return;
      ctx.fillText(String(value), point.x, point.y - 6);
    });
    ctx.restore();
  },
};

const renderUsageChart = () => {
  if (!usageChartCanvas || !usageChartDataCache.recentSummary || !usageChartDataCache.totalSummary) return;

  const recentSummary = usageChartDataCache.recentSummary;
  const totalSummary = usageChartDataCache.totalSummary;
  const axisColor = getComputedStyle(document.body).getPropertyValue("--text-secondary-color");

  let labels = [];
  let data = [];
  let label = "";
  let borderColor = "#60a5fa";
  let pointColor = "#c4b5fd";
  let fillColor = "rgba(96, 165, 250, 0.20)";

  if (usageChartMode === "alltime") {
    labels = (totalSummary?.by_day || []).map((item) => formatUsageDateLabel(item.date));
    let runningTotal = 0;
    data = (totalSummary?.by_day || []).map((item) => {
      runningTotal += item?.total_tokens || 0;
      return runningTotal;
    });
    label = "历史累计 Token";
    borderColor = "#8b5cf6";
    pointColor = "#ec4899";
    fillColor = "rgba(139, 92, 246, 0.18)";
  } else {
    labels = (recentSummary?.by_day || []).map((item) => formatUsageDateLabel(item.date));
    data = (recentSummary?.by_day || []).map((item) => item.total_tokens);
    label = "近30天每日 Token";
  }

  if (usageChartInstance) {
    usageChartInstance.destroy();
  }
  const yAxisRange = buildUsageYAxisRange(data);
  usageChartInstance = new Chart(usageChartCanvas.getContext("2d"), {
    type: "line",
    plugins: [usageValueLabelPlugin],
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
      plugins: {
        legend: { display: false },
      },
      scales: {
        x: {
          ticks: { color: axisColor, maxTicksLimit: 9, font: { size: 11 } },
          grid: { display: false },
        },
        y: {
          min: yAxisRange.min,
          max: yAxisRange.max,
          ticks: { display: false },
          grid: { display: false },
          border: { display: false },
        },
      },
    },
  });
  updateUsageToggleUI();
};

const bindModalEvents = () => {
  document.querySelectorAll("[data-close-modal]").forEach((button) => {
    button.addEventListener("click", (event) => {
      event.stopPropagation();
      const modalId = button.getAttribute("data-close-modal");
      if (!modalId) return;
      closeModal(document.getElementById(modalId));
    });
  });
  document.querySelectorAll(".modal-overlay").forEach((overlay) => {
    overlay.addEventListener("click", (event) => {
      if (event.target === overlay) closeModal(overlay);
    });
  });
  document.querySelectorAll("[data-auth-mode]").forEach((button) => {
    button.addEventListener("click", () => setAuthMode(button.dataset.authMode || "login"));
  });
  authForm?.addEventListener("submit", (event) => {
    void handleAuthSubmit(event);
  });
  profileForm?.addEventListener("submit", (event) => {
    void handleProfileSubmit(event);
  });
  avatarInput?.addEventListener("change", (event) => {
    void handleAvatarUpload(event);
  });
  logoutButton?.addEventListener("click", () => {
    logout();
  });
  adminEntryButton?.addEventListener("click", () => {
    if (!currentUser?.is_admin) return;
    window.location.href = "admin.html";
  });
  sidebarUserCard?.addEventListener("click", () => {
    void openProfileModal();
  });
  sidebarSettingsButton?.addEventListener("click", () => {
    openSettingsModal();
  });
  settingsNav?.addEventListener("click", (event) => {
    const button = event.target.closest("[data-settings-tab]");
    if (!button) return;
    const tab = button.dataset.settingsTab;
    if (!tab || tab === settingsTab) return;
    setSettingsTab(tab);
  });
  feedbackForm?.addEventListener("submit", (event) => {
    void handleFeedbackSubmit(event);
  });
  usageToggleGroup?.addEventListener("click", (event) => {
    const modeButton = event.target.closest("[data-usage-mode]");
    if (!modeButton) return;
    const nextMode = modeButton.dataset.usageMode;
    if (nextMode !== "recent30" && nextMode !== "alltime") return;
    if (usageChartMode === nextMode) return;
    usageChartMode = nextMode;
    renderUsageChart();
  });
};

const bindSidebarEvents = () => {
  sidebarMobileToggleButton?.addEventListener("click", () => {
    if (isMobileViewport()) {
      openSidebarDrawer();
      return;
    }
    setSidebarCollapsed(!document.body.classList.contains("sidebar-collapsed"));
  });

  sidebarExpandButton?.addEventListener("click", () => {
    setSidebarCollapsed(false);
  });

  sidebarCollapseButton?.addEventListener("click", () => {
    setSidebarCollapsed(true);
  });

  sidebarBackdrop?.addEventListener("click", () => {
    closeSidebarDrawer();
  });

  newChatButton?.addEventListener("click", () => {
    handleCreateNewChat();
  });

  sidebarHistory?.addEventListener("click", (event) => {
    const deleteButton = event.target.closest("[data-delete-session-id]");
    if (deleteButton) {
      const { deleteSessionId } = deleteButton.dataset;
      if (!deleteSessionId) return;
      const confirmDelete = window.confirm("确认删除这条历史会话吗？");
      if (!confirmDelete) return;
      void (async () => {
        try {
          await deleteSession(deleteSessionId);
        } catch (error) {
          alert(error?.message || "删除失败，请稍后重试");
        }
      })();
      return;
    }

    const sessionButton = event.target.closest("[data-session-id]");
    if (!sessionButton) return;
    const { sessionId } = sessionButton.dataset;
    if (!sessionId) return;
    setActiveSession(sessionId);
    const selectedSession = chatSessions.find((session) => session.id === sessionId);
    if (selectedSession?.loaded) {
      renderActiveSessionMessages();
    } else {
      void fetchSessionMessages(sessionId);
    }
    if (isMobileViewport()) closeSidebarDrawer();
  });

  window.addEventListener("keydown", (event) => {
    if (event.key !== "Escape") return;
    closeModelPickerPanel();
    closeSidebarDrawer();
  });

  window.addEventListener("resize", () => {
    if (!isMobileViewport()) {
      closeSidebarDrawer();
    }
    updateChatBottomSafeSpace();
  });
};

const openModelPickerPanel = () => {
  if (!modelPickerPanel) return;
  if (modelPickerHideTimer) {
    window.clearTimeout(modelPickerHideTimer);
    modelPickerHideTimer = null;
  }
  modelPickerPanel.classList.remove("hide");
  requestAnimationFrame(() => {
    modelPickerPanel.classList.add("model-picker__panel--visible");
  });
  modelPicker?.classList.add("is-open");
  modelPickerTrigger?.setAttribute("aria-expanded", "true");
};

const closeModelPickerPanel = () => {
  if (!modelPickerPanel) return;
  if (modelPickerHideTimer) {
    window.clearTimeout(modelPickerHideTimer);
    modelPickerHideTimer = null;
  }
  modelPickerPanel.classList.remove("model-picker__panel--visible");
  modelPicker?.classList.remove("is-open");
  modelPickerTrigger?.setAttribute("aria-expanded", "false");
  modelPickerHideTimer = window.setTimeout(() => {
    if (!modelPickerPanel.classList.contains("model-picker__panel--visible")) {
      modelPickerPanel.classList.add("hide");
    }
    modelPickerHideTimer = null;
  }, 220);
};

const bindModelPickerEvents = () => {
  modelPickerTrigger?.addEventListener("click", () => {
    const isOpen = modelPickerPanel?.classList.contains("model-picker__panel--visible");
    if (isOpen) {
      closeModelPickerPanel();
      return;
    }
    openModelPickerPanel();
  });

  modelPickerPanel?.addEventListener("click", (event) => {
    const option = event.target.closest("[data-model-option]");
    if (!option) return;
    const modelKey = String(option.dataset.modelKey || "");
    if (!MODEL_LABELS[modelKey]) return;

    const isSelected = selectedModelKeys.includes(modelKey);
    if (!isSelected && selectedModelKeys.length >= MAX_SELECTED_MODELS) {
      return;
    }
    if (isSelected && selectedModelKeys.length === 1) {
      return;
    }

    let next = [...selectedModelKeys];
    if (!isSelected) {
      next.push(modelKey);
    } else {
      next = next.filter((item) => item !== modelKey);
      if (next.length === 0) {
        next = [DEFAULT_MODEL_KEY];
      }
    }
    setSelectedModels(next);
  });

  deepSearchToggle?.addEventListener("click", () => {
    setDeepSearchEnabled(!deepSearchEnabled);
  });
};

// Load saved data from local storage
const loadSavedChatHistory = () => {
  localStorage.removeItem("saved-api-chats");
  const isLightTheme = localStorage.getItem("themeColor") === "light_mode";
  const savedSidebarCollapsed = localStorage.getItem("sidebarCollapsed");
  const shouldCollapseSidebar = savedSidebarCollapsed === null
    ? true
    : savedSidebarCollapsed === "true";

  themeRoot.classList.toggle("light_mode", isLightTheme);
  themeToggleButton.innerHTML = isLightTheme
    ? '<i class="bx bx-moon"></i>'
    : '<i class="bx bx-sun"></i>';

  chatSessions = [createSession("新聊天")];
  activeSessionId = chatSessions[0].id;
  renderSidebarSessions();
  if (isMobileViewport()) {
    document.body.classList.add("sidebar-collapsed");
    sidebarElement?.classList.add("app-sidebar--collapsed");
  } else {
    setSidebarCollapsed(shouldCollapseSidebar);
  }
  closeSidebarDrawer();

  chatHistoryContainer.innerHTML = "";
  document.body.classList.remove("hide-header");
  authToken = localStorage.getItem(AUTH_TOKEN_STORAGE_KEY) || "";
  try {
    const rawModels = JSON.parse(localStorage.getItem(MODEL_SELECTION_STORAGE_KEY) || "[]");
    setSelectedModels(rawModels);
  } catch {
    setSelectedModels([DEFAULT_MODEL_KEY]);
  }
  setDeepSearchEnabled(localStorage.getItem(DEEP_SEARCH_STORAGE_KEY) === "true");
  currentUser = authToken ? getCachedUserProfile() : null;
  applyUserProfileToUI();
  renderActiveSessionMessages();
};


// create a new chat message element
const createChatMessageElement = (htmlContent, ...cssClasses) => {
  const messageElement = document.createElement("div");
  messageElement.classList.add("message", ...cssClasses);
  messageElement.innerHTML = htmlContent;
  return messageElement;
};

// Show typing effect
const showTypingEffect = (
  rawText,
  htmlText,
  messageElement,
  incomingMessageElement,
  skipEffect = false,
  reasoningText = "",
  requestSessionId = null
) => {
  const actionsElement = incomingMessageElement.querySelector(".message__actions");
  actionsElement?.classList.add("hide");
  renderReasoningPanel(incomingMessageElement, reasoningText, {
    collapseByDefault: true,
  });

  const finishTyping = () => {
    if (requestSessionId) {
      setSessionGenerating(requestSessionId, false);
    }
  };

  if (skipEffect) {
    // Display content directly without typing
    messageElement.innerHTML = htmlText;
    void enhanceMessageBody(messageElement);
    actionsElement?.classList.remove("hide");
    finishTyping();
    scrollChatsToBottom("auto");
    return;
  }

  const tokens = Array.from(rawText || "");
  if (tokens.length === 0) {
    messageElement.innerHTML = htmlText;
    void enhanceMessageBody(messageElement);
    actionsElement?.classList.remove("hide");
    finishTyping();
    scrollChatsToBottom("auto");
    return;
  }

  let tokenIndex = 0;
  let streamedText = "";

  const typingInterval = setInterval(() => {
    streamedText += tokens[tokenIndex++];
    messageElement.innerHTML = marked.parse(streamedText);
    scrollChatsToBottom("auto");
    if (tokenIndex === tokens.length) {
      clearInterval(typingInterval);
      finishTyping();
      messageElement.innerHTML = htmlText;
      void enhanceMessageBody(messageElement);
      actionsElement?.classList.remove("hide");
      scrollChatsToBottom("auto");
    }
  }, 25);
};

const resolveStreamDom = (requestSessionId, fallbackIncoming, fallbackText, streamSessionRef = null) => {
  const boundSessionId = streamSessionRef?.id || requestSessionId;
  if (activeSessionId !== boundSessionId) return null;
  const registered = getSessionStreamDom(boundSessionId);
  if (registered) return registered;
  if (fallbackIncoming && document.contains(fallbackIncoming) && fallbackText) {
    return {
      incomingMessageElement: fallbackIncoming,
      messageTextElement: fallbackText,
    };
  }
  return null;
};

const consumeAssistantEventStream = async (
  response,
  messageElement,
  incomingMessageElement,
  requestSessionId = null,
  streamSessionRef = null
) => {
  if (!response.body) {
    throw new Error("stream body unavailable");
  }

  const getStreamDom = () =>
    resolveStreamDom(requestSessionId, incomingMessageElement, messageElement, streamSessionRef);

  const actionsElement = incomingMessageElement.querySelector(".message__actions");
  actionsElement?.classList.add("hide");

  const reader = response.body.getReader();
  const decoder = new TextDecoder("utf-8");
  let buffered = "";
  let visibleContent = "";
  let visibleReasoning = "";
  let renderedContent = "";
  let thinkMode = false;
  let pendingTagPrefix = "";
  let hasReasoningSignal = false;
  let streamStarted = false;
  let streamCompleted = false;
  let resolvedSession = null;
  let contentTypewriterTimer = null;
  const CONTENT_TYPEWRITER_DELAY_MS = 16;
  const syncThinkingAvatar = () => {
    const dom = getStreamDom();
    if (!dom) return;
    dom.incomingMessageElement.classList.toggle(
      "message--thinking",
      hasReasoningSignal && !streamCompleted
    );
  };

  const getBoundSessionId = () => streamSessionRef?.id || requestSessionId;

  const syncStreamingSessionState = () => {
    const boundSessionId = getBoundSessionId();
    if (!boundSessionId) return;
    updateStreamingAssistantInSession(boundSessionId, {
      content: visibleContent || "",
      reasoning_content: visibleReasoning || "",
    });
  };

  const stripReasoningOverlap = (contentText, reasoningText) => {
    const content = contentText || "";
    const reasoning = (reasoningText || "").trim();
    if (!reasoning) return content;

    if (content.startsWith(reasoning)) {
      return content.slice(reasoning.length).replace(/^\s+/, "");
    }

    const normalizedReasoning = reasoning.replace(/\s+/g, " ").trim();
    if (!normalizedReasoning) return content;
    const normalizedContentPrefix = content.slice(0, reasoning.length + 32).replace(/\s+/g, " ").trim();
    if (
      normalizedContentPrefix &&
      (normalizedContentPrefix === normalizedReasoning ||
        normalizedContentPrefix.startsWith(normalizedReasoning))
    ) {
      return content.slice(Math.min(content.length, reasoning.length)).replace(/^\s+/, "");
    }

    return content;
  };

  const shouldTreatDeltaContentAsReasoning = (deltaContent, deltaReasoning) => {
    const contentNorm = (deltaContent || "").replace(/\s+/g, " ").trim();
    const reasoningNorm = (deltaReasoning || "").replace(/\s+/g, " ").trim();
    if (!contentNorm || !reasoningNorm) return false;
    return (
      contentNorm === reasoningNorm ||
      contentNorm.startsWith(reasoningNorm) ||
      reasoningNorm.startsWith(contentNorm)
    );
  };

  const getContentTargetForCurrentPhase = () => {
    const contentForDisplay = stripReasoningOverlap(visibleContent, visibleReasoning);
    // Only hold back content while explicit reasoning exists.
    // If backend streams plain content, render it directly in the正文 area.
    if (!streamCompleted && hasReasoningSignal) return "";

    const hasReasoning = ENABLE_REASONING_OUTPUT && (visibleReasoning || "").trim().length > 0;
    if (!hasReasoning) return contentForDisplay || "";

    // Two-phase streaming: reasoning first, then content.
    // Content starts only after reasoning stream is complete.
    return contentForDisplay || "";
  };

  const renderCurrentFrame = () => {
    syncStreamingSessionState();
    const dom = getStreamDom();
    if (!dom) return;
    dom.messageTextElement.innerHTML = marked.parse(renderedContent || "");
    renderReasoningPanel(dom.incomingMessageElement, visibleReasoning, {
      collapseByDefault: false,
      streaming: true,
    });
    scrollChatsToBottom("auto");
  };

  const syncContentTypewriter = () => {
    const targetContent = getContentTargetForCurrentPhase();

    if (
      renderedContent.length > targetContent.length ||
      !targetContent.startsWith(renderedContent)
    ) {
      renderedContent = targetContent;
      renderCurrentFrame();
      return;
    }

    if (renderedContent === targetContent) return;
    if (contentTypewriterTimer) return;

    contentTypewriterTimer = setInterval(() => {
      const latestTarget = getContentTargetForCurrentPhase();
      if (
        renderedContent.length > latestTarget.length ||
        !latestTarget.startsWith(renderedContent)
      ) {
        renderedContent = latestTarget;
        clearInterval(contentTypewriterTimer);
        contentTypewriterTimer = null;
        renderCurrentFrame();
        return;
      }

      if (renderedContent.length < latestTarget.length) {
        renderedContent += latestTarget.charAt(renderedContent.length);
        renderCurrentFrame();
      }

      if (renderedContent.length >= latestTarget.length) {
        clearInterval(contentTypewriterTimer);
        contentTypewriterTimer = null;
      }
    }, CONTENT_TYPEWRITER_DELAY_MS);
  };

  const applyRender = () => {
    renderCurrentFrame();
    syncContentTypewriter();
  };

  const appendReasoningDeltaSmoothly = async (deltaText = "") => {
    const chunks = splitTextByRuneChunks(
      deltaText,
      STREAM_REASONING_CHUNK_SIZE
    );
    if (chunks.length === 0) return;
    for (const chunk of chunks) {
      hasReasoningSignal = true;
      visibleReasoning += chunk;
      syncThinkingAvatar();
      if (!streamStarted) {
        const dom = getStreamDom();
        dom?.incomingMessageElement.classList.remove("message--loading");
        streamStarted = true;
      }
      applyRender();
      await sleep(STREAM_REASONING_CHUNK_DELAY_MS);
    }
  };

  const splitIncompleteTagSuffix = (segment, tag) => {
    const lowered = segment.toLowerCase();
    const loweredTag = tag.toLowerCase();
    const maxPrefix = Math.min(lowered.length, loweredTag.length - 1);
    for (let i = maxPrefix; i >= 1; i--) {
      if (lowered.endsWith(loweredTag.slice(0, i))) {
        return {
          stablePart: segment.slice(0, -i),
          carryPart: segment.slice(-i),
        };
      }
    }
    return { stablePart: segment, carryPart: "" };
  };

  const routeMixedDelta = (deltaText) => {
    if (!deltaText) return;
    let working = pendingTagPrefix + deltaText;
    pendingTagPrefix = "";

    while (working) {
      if (thinkMode) {
        const closeIndex = working.toLowerCase().indexOf("</think>");
        if (closeIndex === -1) {
          const { stablePart, carryPart } = splitIncompleteTagSuffix(
            working,
            "</think>"
          );
          visibleReasoning += stablePart;
          pendingTagPrefix = carryPart;
          break;
        }
        visibleReasoning += working.slice(0, closeIndex);
        working = working.slice(closeIndex + "</think>".length);
        thinkMode = false;
      } else {
        const openIndex = working.toLowerCase().indexOf("<think>");
        if (openIndex === -1) {
          const { stablePart, carryPart } = splitIncompleteTagSuffix(
            working,
            "<think>"
          );
          visibleContent += stablePart;
          pendingTagPrefix = carryPart;
          break;
        }
        visibleContent += working.slice(0, openIndex);
        working = working.slice(openIndex + "<think>".length);
        thinkMode = true;
      }
    }
  };

  while (true) {
    const { value, done } = await reader.read();
    if (done) break;

    buffered += decoder.decode(value, { stream: true });

    let boundaryIndex = buffered.indexOf("\n\n");
    while (boundaryIndex !== -1) {
      const rawEvent = buffered.slice(0, boundaryIndex);
      buffered = buffered.slice(boundaryIndex + 2);

      const dataPayload = rawEvent
        .split("\n")
        .map((line) => line.trim())
        .filter((line) => line.startsWith("data:"))
        .map((line) => line.slice(5).trim())
        .join("\n");

      if (!dataPayload) {
        boundaryIndex = buffered.indexOf("\n\n");
        continue;
      }

      let eventData;
      try {
        eventData = JSON.parse(dataPayload);
      } catch {
        boundaryIndex = buffered.indexOf("\n\n");
        continue;
      }

      if (eventData.type === "error") {
        throw new Error(eventData.error || "stream failed");
      }

      if (eventData.type === "session") {
        syncSessionFromServer(eventData?.session || null, requestSessionId, streamSessionRef);
        boundaryIndex = buffered.indexOf("\n\n");
        continue;
      }

      if (eventData.type === "delta" || eventData.type === "done") {
        if (eventData.type === "done") {
          streamCompleted = true;
          resolvedSession = eventData?.session || null;
          const finalParsed = extractReasoningAndContentFromMessage({
            content:
              typeof eventData.content === "string" ? eventData.content : visibleContent,
            reasoning_content:
              typeof eventData.reasoning_content === "string"
                ? eventData.reasoning_content
                : visibleReasoning,
          });
          visibleContent = finalParsed.content || "";
          visibleReasoning = ENABLE_REASONING_OUTPUT ? finalParsed.reasoning || "" : "";
          if ((visibleReasoning || "").trim()) hasReasoningSignal = true;
          syncThinkingAvatar();
          thinkMode = false;
          pendingTagPrefix = "";
        } else {
          const hasExplicitReasoningDelta =
            ENABLE_REASONING_OUTPUT &&
            typeof eventData.reasoning_content === "string" &&
            !!eventData.reasoning_content;

          if (
            hasExplicitReasoningDelta
          ) {
            await appendReasoningDeltaSmoothly(eventData.reasoning_content);
          }

          if (typeof eventData.content === "string" && eventData.content) {
            const contentChunk = eventData.content;
            const loweredChunk = contentChunk.toLowerCase();
            const chunkIncludesThinkTag =
              loweredChunk.includes("<think") || loweredChunk.includes("</think");
            if (chunkIncludesThinkTag) hasReasoningSignal = true;
            syncThinkingAvatar();
            const overlappedWithReasoning = shouldTreatDeltaContentAsReasoning(
              contentChunk,
              typeof eventData.reasoning_content === "string"
                ? eventData.reasoning_content
                : ""
            );
            if (!overlappedWithReasoning) {
              routeMixedDelta(contentChunk);
            }
          }
        }

        if (!streamStarted) {
          const dom = getStreamDom();
          dom?.incomingMessageElement.classList.remove("message--loading");
          streamStarted = true;
        }
        applyRender();
      }

      boundaryIndex = buffered.indexOf("\n\n");
    }
  }

  const finalDom = getStreamDom();
  finalDom?.incomingMessageElement.classList.remove("message--loading");
  finalDom?.incomingMessageElement.classList.remove("message--thinking");
  if (contentTypewriterTimer) {
    clearInterval(contentTypewriterTimer);
    contentTypewriterTimer = null;
  }
  applyRender();
  if (finalDom?.messageTextElement) {
    await enhanceMessageBody(finalDom.messageTextElement);
    finalDom.incomingMessageElement.querySelector(".message__actions")?.classList.remove("hide");
  }
  if (streamSessionRef?.id || requestSessionId) {
    setSessionGenerating(streamSessionRef?.id || requestSessionId, false);
  }
  scrollChatsToBottom("auto");
  return {
    content: visibleContent || "",
    reasoning: visibleReasoning || "",
    session: resolvedSession,
  };
};

const consumeAssistantEventStreamMulti = async (
  response,
  messageElement,
  incomingMessageElement,
  requestedModels = [],
  requestSessionId = null,
  streamSessionRef = null
) => {
  if (!response.body) {
    throw new Error("stream body unavailable");
  }

  const getStreamDom = () =>
    resolveStreamDom(requestSessionId, incomingMessageElement, messageElement, streamSessionRef);

  const actionsElement = incomingMessageElement.querySelector(".message__actions");
  actionsElement?.classList.add("hide");
  const trackElement = incomingMessageElement.querySelector(".message__model-track");
  const tabsElement = incomingMessageElement.querySelector(".message__model-tabs");

  const modelOrder = normalizeModelSelection(requestedModels);
  const trackStates = new Map(
    modelOrder.map((modelKey) => [
      modelKey,
      { model: modelKey, content: "", reasoning_content: "", error: "", done: false },
    ])
  );
  let activeModel = modelOrder[0] || DEFAULT_MODEL_KEY;
  let tabsRenderKey = "";
  let streamStarted = false;
  let streamCompleted = false;
  let hasReasoningSignal = false;
  let resolvedSession = null;
  const syncThinkingAvatar = () => {
    const dom = getStreamDom();
    if (!dom) return;
    dom.incomingMessageElement.classList.toggle(
      "message--thinking",
      hasReasoningSignal && !streamCompleted
    );
  };

  const getBoundSessionId = () => streamSessionRef?.id || requestSessionId;

  const syncStreamingSessionState = () => {
    const boundSessionId = getBoundSessionId();
    if (!boundSessionId) return;
    const modelResponses = [...trackStates.values()]
      .map((item) => ({
        model: item.model,
        content: item.content || "",
        reasoning_content: item.reasoning_content || "",
        error: item.error || "",
      }))
      .filter((item) => item.model && (item.content || item.reasoning_content || item.error));
    const activeState = trackStates.get(activeModel);
    updateStreamingAssistantInSession(boundSessionId, {
      model: activeModel,
      content: activeState?.content || "",
      reasoning_content: activeState?.reasoning_content || "",
      selected_model: activeModel,
      model_responses: modelResponses,
    });
  };

  const ensureTrackState = (modelKey) => {
    const key = String(modelKey || "").trim().toLowerCase();
    if (!key) return null;
    if (!trackStates.has(key)) {
      trackStates.set(key, { model: key, content: "", reasoning_content: "", error: "", done: false });
    }
    return trackStates.get(key);
  };

  const renderTrackTabs = () => {
    const dom = getStreamDom();
    if (!dom) return;
    const localTrackElement = dom.incomingMessageElement.querySelector(".message__model-track");
    const localTabsElement = dom.incomingMessageElement.querySelector(".message__model-tabs");
    if (!localTrackElement || !localTabsElement) return;
    localTrackElement.classList.remove("hide");
    const orderedModels = [...new Set([...modelOrder, ...trackStates.keys()])];
    const nextRenderKey = `${activeModel}::${orderedModels.join("|")}`;
    if (tabsRenderKey === nextRenderKey) return;
    tabsRenderKey = nextRenderKey;
    localTabsElement.innerHTML = orderedModels
      .map(
        (modelKey) => {
          const modelState = trackStates.get(modelKey);
          return `
      <button
        type="button"
        class="message__model-tab ${modelKey === activeModel ? "is-active" : ""} ${modelState?.error ? "is-error" : ""}"
        data-model-tab="${escapeHtml(modelKey)}"
      >
        ${escapeHtml(getModelLabel(modelKey))}${modelState?.error ? '<span class="message__model-tab-error-mark">!</span>' : ""}
      </button>
    `;
        }
      )
      .join("");
  };

  const renderActiveTrack = () => {
    syncStreamingSessionState();
    const dom = getStreamDom();
    if (!dom) return;
    const state = trackStates.get(activeModel) || { content: "", reasoning_content: "", error: "" };
    const fallbackErrorText = state.error ? `该模型回答失败：${state.error}` : "";
    dom.messageTextElement.innerHTML = marked.parse(state.content || fallbackErrorText);
    renderReasoningPanel(dom.incomingMessageElement, state.reasoning_content || "", {
      collapseByDefault: false,
      streaming: true,
    });
    scrollChatsToBottom("auto");
  };

  const appendTrackReasoningSmoothly = async (state, deltaText = "") => {
    const chunks = splitTextByRuneChunks(
      deltaText,
      STREAM_REASONING_CHUNK_SIZE
    );
    if (chunks.length === 0) return;
    for (const chunk of chunks) {
      state.reasoning_content += chunk;
      if (!streamStarted) {
        const dom = getStreamDom();
        dom?.incomingMessageElement.classList.remove("message--loading");
        streamStarted = true;
      }
      if (state.model === activeModel) {
        renderActiveTrack();
      }
      await sleep(STREAM_REASONING_CHUNK_DELAY_MS);
    }
  };

  const appendTrackContentSmoothly = async (state, deltaText = "") => {
    const chunks = splitTextByRuneChunks(
      deltaText,
      STREAM_CONTENT_CHUNK_SIZE
    );
    if (chunks.length === 0) return;
    for (const chunk of chunks) {
      state.content += chunk;
      if (!streamStarted) {
        const dom = getStreamDom();
        dom?.incomingMessageElement.classList.remove("message--loading");
        streamStarted = true;
      }
      if (state.model === activeModel) {
        renderActiveTrack();
      }
      await sleep(STREAM_CONTENT_CHUNK_DELAY_MS);
    }
  };

  if (tabsElement) {
    tabsElement.addEventListener("click", (event) => {
      const tab = event.target.closest("[data-model-tab]");
      if (!tab) return;
      const nextModel = String(tab.dataset.modelTab || "").trim().toLowerCase();
      if (!nextModel || nextModel === activeModel) return;
      activeModel = nextModel;
      renderTrackTabs();
      renderActiveTrack();
    });
  }

  renderTrackTabs();
  renderActiveTrack();

  const reader = response.body.getReader();
  const decoder = new TextDecoder("utf-8");
  let buffered = "";

  while (true) {
    const { value, done } = await reader.read();
    if (done) break;

    buffered += decoder.decode(value, { stream: true });
    let boundaryIndex = buffered.indexOf("\n\n");
    while (boundaryIndex !== -1) {
      const rawEvent = buffered.slice(0, boundaryIndex);
      buffered = buffered.slice(boundaryIndex + 2);

      const dataPayload = rawEvent
        .split("\n")
        .map((line) => line.trim())
        .filter((line) => line.startsWith("data:"))
        .map((line) => line.slice(5).trim())
        .join("\n");
      if (!dataPayload) {
        boundaryIndex = buffered.indexOf("\n\n");
        continue;
      }

      let eventData;
      try {
        eventData = JSON.parse(dataPayload);
      } catch {
        boundaryIndex = buffered.indexOf("\n\n");
        continue;
      }

      if (eventData.type === "error") {
        throw new Error(eventData.error || "stream failed");
      }

      if (eventData.type === "session") {
        syncSessionFromServer(eventData?.session || null, requestSessionId, streamSessionRef);
        boundaryIndex = buffered.indexOf("\n\n");
        continue;
      }

      if (eventData.type === "delta") {
        const deltaModel = String(eventData.model || modelOrder[0] || activeModel || DEFAULT_MODEL_KEY)
          .trim()
          .toLowerCase();
        const state = ensureTrackState(deltaModel);
        if (!state) {
          boundaryIndex = buffered.indexOf("\n\n");
          continue;
        }
        if (typeof eventData.content === "string" && eventData.content) {
          await appendTrackContentSmoothly(state, eventData.content);
        }
        if (typeof eventData.reasoning_content === "string" && eventData.reasoning_content) {
          hasReasoningSignal = true;
          syncThinkingAvatar();
          await appendTrackReasoningSmoothly(state, eventData.reasoning_content);
        }
        if (!streamStarted) {
          const dom = getStreamDom();
          dom?.incomingMessageElement.classList.remove("message--loading");
          streamStarted = true;
        }
        const knownModelBefore = modelOrder.includes(deltaModel);
        if (!knownModelBefore) {
          renderTrackTabs();
        }
        const activeState = trackStates.get(activeModel);
        const activeBlockedByError =
          !!activeState &&
          !!activeState.error &&
          !activeState.content &&
          !activeState.reasoning_content;
        if (activeBlockedByError && deltaModel !== activeModel) {
          activeModel = deltaModel;
          renderTrackTabs();
        }
        if (state.model === activeModel) {
          renderActiveTrack();
        }
      }

      if (eventData.type === "model_error") {
        const failedModel = String(eventData.model || "").trim().toLowerCase();
        const state = ensureTrackState(failedModel);
        if (!state) {
          boundaryIndex = buffered.indexOf("\n\n");
          continue;
        }
        state.error = String(eventData.error || "请求失败");
        state.done = true;
        if (!streamStarted) {
          const dom = getStreamDom();
          dom?.incomingMessageElement.classList.remove("message--loading");
          streamStarted = true;
        }
        if (failedModel === activeModel) {
          const fallback = [...trackStates.values()].find(
            (item) =>
              item.model !== failedModel &&
              (String(item.content || "").trim() || String(item.reasoning_content || "").trim())
          );
          if (fallback?.model) {
            activeModel = fallback.model;
          }
        }
        renderTrackTabs();
        if (failedModel === activeModel) {
          renderActiveTrack();
        } else {
          renderActiveTrack();
        }
      }

      if (eventData.type === "done") {
        streamCompleted = true;
        resolvedSession = eventData?.session || null;
        const activeBeforeDone = activeModel;
        const finalResponses = normalizeModelResponses(
          eventData?.model_responses || [],
          modelOrder
        );
        if (finalResponses.length > 0) {
          finalResponses.forEach((item) => {
            const existingState = trackStates.get(item.model) || {
              model: item.model,
              content: "",
              reasoning_content: "",
              error: "",
              done: false,
            };
            trackStates.set(item.model, {
              model: item.model,
              content: item.content,
              reasoning_content: item.reasoning_content,
              error: existingState.error || item.error || "",
              done: true,
            });
          });
          const selectedFromDone = String(eventData?.selected_model || "").trim().toLowerCase();
          if (trackStates.has(activeBeforeDone)) {
            activeModel = activeBeforeDone;
          } else if (trackStates.has(selectedFromDone)) {
            activeModel = selectedFromDone;
          } else {
            activeModel = pickModelResponse(finalResponses, activeBeforeDone).model;
          }
        }
        const doneDom = getStreamDom();
        doneDom?.incomingMessageElement.classList.remove("message--loading");
        doneDom?.incomingMessageElement.classList.remove("message--thinking");
        renderTrackTabs();
        renderActiveTrack();
      }

      boundaryIndex = buffered.indexOf("\n\n");
    }
  }

  const finalDom = getStreamDom();
  if (finalDom?.messageTextElement) {
    await enhanceMessageBody(finalDom.messageTextElement);
    finalDom.incomingMessageElement.querySelector(".message__actions")?.classList.remove("hide");
    finalDom.incomingMessageElement.classList.remove("message--thinking");
  }
  if (streamSessionRef?.id || requestSessionId) {
    setSessionGenerating(streamSessionRef?.id || requestSessionId, false);
  }
  scrollChatsToBottom("auto");

  const modelResponses = [...trackStates.values()]
    .map((item) => ({
      model: item.model,
      content: item.content || "",
      reasoning_content: item.reasoning_content || "",
      error: item.error || "",
    }))
    .filter((item) => item.model && (item.content || item.reasoning_content || item.error));

  return {
    modelResponses,
    selectedModel: activeModel,
    session: resolvedSession,
  };
};

// Fetch API response based on user input
const requestApiResponse = async (incomingMessageElement, requestedModels = [], requestSessionId = null) => {
  const messageTextElement =
    incomingMessageElement.querySelector(".message__text");
  const normalizedRequestedModels = normalizeModelSelection(requestedModels);
  const streamSessionRef = { id: requestSessionId || activeSessionId };

  if (!authToken) {
    setSessionGenerating(streamSessionRef.id, false);
    incomingMessageElement.classList.remove("message--loading");
    incomingMessageElement.classList.remove("message--thinking");
    messageTextElement.innerText = "请先登录后再发送消息。";
    messageTextElement.closest(".message").classList.add("message--error");
    openModal(authModal);
    return;
  }

  try {
    const sessionId = getPersistedSessionId(streamSessionRef.id);
    const response = await authFetch(API_REQUEST_URL, {
      method: "POST",
      headers: {
        Accept: "text/event-stream, application/json",
      },
      body: JSON.stringify({
        message: currentUserMessage,
        models: normalizedRequestedModels,
        deep_search: deepSearchEnabled,
        ...(sessionId ? { session_id: sessionId } : {}),
      }),
    });

    if (response.status === 401) {
      logout();
      throw new Error("登录态已过期，请重新登录。");
    }

    const responseContentType = response.headers.get("content-type") || "";
    if (!response.ok) {
      let errorMessage = "请求失败";
      try {
        if (responseContentType.includes("application/json")) {
          const errorData = await response.json();
          errorMessage = errorData?.error?.message || errorMessage;
        } else {
          errorMessage = (await response.text()) || errorMessage;
        }
      } catch {
        // keep default
      }
      throw new Error(errorMessage);
    }

    if (responseContentType.includes("text/event-stream")) {
      if (normalizedRequestedModels.length > 1) {
        const multiStreamResult = await consumeAssistantEventStreamMulti(
          response,
          messageTextElement,
          incomingMessageElement,
          normalizedRequestedModels,
          streamSessionRef.id,
          streamSessionRef
        );
        const multiResponses = normalizeModelResponses(
          multiStreamResult?.modelResponses || [],
          normalizedRequestedModels
        );
        const boundSessionId = streamSessionRef.id;
        if (multiResponses.length > 1) {
          const selected = multiStreamResult?.selectedModel || multiResponses[0].model;
          const picked = pickModelResponse(multiResponses, selected);
          finalizeStreamingAssistantInSession(boundSessionId, {
            model: picked.model,
            content: picked.content || "",
            reasoning_content: picked.reasoning_content || "",
            selected_model: picked.model,
            model_responses: multiResponses,
          });
        } else if (multiResponses.length === 1) {
          const single = multiResponses[0];
          finalizeStreamingAssistantInSession(boundSessionId, {
            model: single.model,
            content: single.content || "",
            reasoning_content: single.reasoning_content || "",
          });
        } else {
          finalizeStreamingAssistantInSession(boundSessionId, {
            model: normalizedRequestedModels[0] || DEFAULT_MODEL_KEY,
            content: "",
            reasoning_content: "",
          });
        }
        syncSessionFromServer(multiStreamResult?.session || null, boundSessionId, streamSessionRef);
      } else {
        const streamResult = await consumeAssistantEventStream(
          response,
          messageTextElement,
          incomingMessageElement,
          streamSessionRef.id,
          streamSessionRef
        );
        const streamModelResponses = normalizeModelResponses(
          extractAssistantModelResponsesFromSession(
            streamResult?.session || null,
            normalizedRequestedModels
          ),
          normalizedRequestedModels
        );
        const single = streamModelResponses[0] || {
          model: normalizedRequestedModels[0] || DEFAULT_MODEL_KEY,
          content: streamResult?.content || "",
          reasoning_content: streamResult?.reasoning || "",
        };
        const boundSessionId = streamSessionRef.id;
        if (activeSessionId === boundSessionId) {
          setupAssistantModelTrack(incomingMessageElement, [single], single.model);
        }
        finalizeStreamingAssistantInSession(boundSessionId, {
          model: single.model,
          content: single.content || "",
          reasoning_content: single.reasoning_content || "",
        });
        syncSessionFromServer(streamResult?.session || null, boundSessionId, streamSessionRef);
      }
      return;
    }

    const responseData = await response.json();
    const responseChoices = Array.isArray(responseData?.choices) ? responseData.choices : [];
    const modelResponses = responseChoices
      .map((choice, index) => {
        const responseMessage = choice?.message || {};
        const parsed = extractReasoningAndContentFromMessage(responseMessage);
        return {
          model: String(responseMessage?.model || normalizedRequestedModels[index] || normalizedRequestedModels[0] || DEFAULT_MODEL_KEY)
            .trim()
            .toLowerCase(),
          content: parsed.content || "",
          reasoning_content: parsed.reasoning || "",
        };
      })
      .filter((item) => item.model && (item.content || item.reasoning_content));
    if (modelResponses.length === 0) throw new Error("Invalid API response.");
    incomingMessageElement.classList.remove("message--loading");
    incomingMessageElement.classList.remove("message--thinking");
    const boundSessionId = streamSessionRef.id;
    if (modelResponses.length > 1) {
      const picked = pickModelResponse(
        modelResponses,
        normalizedRequestedModels[0] || modelResponses[0].model
      );
      setupAssistantModelTrack(
        incomingMessageElement,
        modelResponses,
        picked.model
      );
      showTypingEffect(
        picked.content || "",
        marked.parse(picked.content || ""),
        messageTextElement,
        incomingMessageElement,
        false,
        picked.reasoning_content || "",
        boundSessionId
      );
      finalizeStreamingAssistantInSession(boundSessionId, {
        model: picked.model,
        content: picked.content || "",
        reasoning_content: picked.reasoning_content || "",
        selected_model: picked.model,
        model_responses: modelResponses,
      });
    } else {
      const single = modelResponses[0];
      showTypingEffect(
        single.content || "",
        marked.parse(single.content || ""),
        messageTextElement,
        incomingMessageElement,
        false,
        single.reasoning_content || "",
        boundSessionId
      );
      finalizeStreamingAssistantInSession(boundSessionId, {
        model: single.model,
        content: single.content || "",
        reasoning_content: single.reasoning_content || "",
      });
    }
    syncSessionFromServer(responseData?.session || null, boundSessionId, streamSessionRef);
  } catch (error) {
    setSessionGenerating(streamSessionRef.id, false);
    incomingMessageElement.classList.remove("message--loading");
    incomingMessageElement.classList.remove("message--thinking");
    messageTextElement.innerText = error.message;
    messageTextElement.closest(".message").classList.add("message--error");
    updateStreamingAssistantInSession(streamSessionRef.id, {
      streaming: false,
      content: error.message,
      reasoning_content: "",
    });
  }
};

// Add copy button to code blocks
const addCopyButtonToCodeBlocks = (scopeElement = document) => {
  const codeBlocks = scopeElement.querySelectorAll("pre");
  codeBlocks.forEach((block) => {
    if (block.querySelector(".code__copy-btn")) return;
    const codeElement = block.querySelector("code");
    if (!codeElement) return;
    let language =
      block.dataset.language ||
      [...codeElement.classList]
        .find((cls) => cls.startsWith("language-"))
        ?.replace("language-", "") ||
      "text";

    const languageLabel = document.createElement("div");
    languageLabel.innerText =
      language.charAt(0).toUpperCase() + language.slice(1).replaceAll("-", " ");
    languageLabel.classList.add("code__language-label");
    block.appendChild(languageLabel);

    const copyButton = document.createElement("button");
    copyButton.innerHTML = `<i class='bx bx-copy'></i>`;
    copyButton.classList.add("code__copy-btn");
    block.appendChild(copyButton);

    copyButton.addEventListener("click", () => {
      navigator.clipboard
        .writeText(codeElement.innerText)
        .then(() => {
          copyButton.innerHTML = `<i class='bx bx-check'></i>`;
          setTimeout(
            () => (copyButton.innerHTML = `<i class='bx bx-copy'></i>`),
            2000
          );
        })
        .catch((err) => {
          console.error("Copy failed:", err);
          alert("Unable to copy text!");
        });
    });
  });
};

// Show loading animation during API request
const displayLoadingAnimation = (requestModels = [], options = {}) => {
  const { deepSearch = false, requestSessionId = activeSessionId } = options;
  const loadingHtml = `

        <div class="message__content">
            <img class="message__avatar" src="assets/YoooFind.png" alt="Gemini avatar">
            <div class="message__body">
                <div class="message__model-track hide">
                    <div class="message__model-track-label">回答轨道</div>
                    <div class="message__model-tabs"></div>
                </div>
                <div class="message__reasoning hide">
                    <div class="message__reasoning-header">
                        <div class="message__reasoning-title">深度思考</div>
                    </div>
                    <div class="message__reasoning-text"></div>
                </div>
                <div class="message__text"></div>
                <div class="message__thinking-status">深度思考中...</div>
                <div class="message__loading-indicator">
                    <div class="message__loading-bar"></div>
                    <div class="message__loading-bar"></div>
                    <div class="message__loading-bar"></div>
                </div>
            </div>
        </div>
        <div class="message__actions hide">
            <button type="button" class="message__action-btn" data-action="copy" title="复制"><i class='bx bx-copy-alt'></i></button>
            <button type="button" class="message__action-btn" data-action="like" title="点赞"><i class='bx bx-like'></i></button>
            <button type="button" class="message__action-btn" data-action="dislike" title="点踩"><i class='bx bx-dislike'></i></button>
            <button type="button" class="message__action-btn" data-action="retry" title="重试"><i class='bx bx-refresh'></i></button>
        </div>

    `;

  const loadingMessageElement = createChatMessageElement(
    loadingHtml,
    "message--incoming",
    "message--loading"
  );
  chatHistoryContainer.appendChild(loadingMessageElement);
  const thinkingStatusElement = loadingMessageElement.querySelector(".message__thinking-status");
  if (thinkingStatusElement) {
    const labels = normalizeModelSelection(requestModels).map((key) => getModelLabel(key));
    const deepSearchSuffix = deepSearch ? "（已开启联网搜索）" : "";
    thinkingStatusElement.textContent = `正在向 ${labels.join(" / ")} 请求回答...${deepSearchSuffix}`;
  }
  const requestModelKeys = normalizeModelSelection(requestModels);
  if (requestModelKeys.length > 1) {
    const trackElement = loadingMessageElement.querySelector(".message__model-track");
    const tabsElement = loadingMessageElement.querySelector(".message__model-tabs");
    if (trackElement && tabsElement) {
      trackElement.classList.remove("hide");
      tabsElement.innerHTML = requestModelKeys
        .map(
          (modelKey, index) => `
        <button
          type="button"
          class="message__model-tab ${index === 0 ? "is-active" : ""}"
          data-model-tab="${escapeHtml(modelKey)}"
        >
          ${escapeHtml(getModelLabel(modelKey))}
        </button>
      `
        )
        .join("");
    }
  }
  showReasoningLoading(loadingMessageElement);
  scrollChatsToBottom("smooth");
  const messageTextElement = loadingMessageElement.querySelector(".message__text");
  registerSessionStreamDom(requestSessionId, loadingMessageElement, messageTextElement);

  requestApiResponse(loadingMessageElement, requestModels, requestSessionId);
};

// Copy message to clipboard
const copyMessageToClipboard = (copyButton) => {
  const messageRoot = copyButton.closest(".message");
  const messageContent = messageRoot?.querySelector(".message__text")?.innerText || "";
  if (!messageContent) return;

  navigator.clipboard.writeText(messageContent);
  copyButton.innerHTML = `<i class='bx bx-check'></i>`; // Confirmation icon
  setTimeout(
    () => (copyButton.innerHTML = `<i class='bx bx-copy-alt'></i>`),
    1000
  ); // Revert icon after 1 second
};

// Handle sending chat messages
const handleOutgoingMessage = async () => {
  const inputText = messageForm.querySelector(".prompt__form-input").value.trim();
  if ((inputText === "" && pendingAttachments.length === 0) || isActiveSessionGenerating()) return;

  shouldAutoScroll = true;
  closeSidebarDrawer();

  const activeAttachments = [...pendingAttachments];
  const outgoingText =
    inputText ||
    `已添加附件：${activeAttachments.map((file) => file.name).join("，")}`;
  if (!activeSessionId) {
    const session = createSession();
    chatSessions = [session, ...chatSessions];
    setActiveSession(session.id);
  }
  const requestSessionId = activeSessionId;
  setSessionGenerating(requestSessionId, true);
  updateActiveSessionTitle(getSessionTitleFromText(outgoingText));
  currentUserMessage = await buildMessageWithAttachments(inputText, activeAttachments);
  appendMessageToSession(requestSessionId, { role: "user", content: outgoingText });
  appendMessageToSession(requestSessionId, {
    role: "assistant",
    content: "",
    reasoning_content: "",
    streaming: true,
  });
  renderOutgoingMessage(outgoingText);
  scrollChatsToBottom("smooth");

  messageForm.reset(); // Clear input field
  pendingAttachments = [];
  renderPendingAttachments();
  if (fileInput) fileInput.value = "";
  adjustPromptInputHeight();
  themeRoot.style.setProperty("--prompt-expand-shift", "0px");
  document.body.classList.add("hide-header");
  updateChatBottomSafeSpace();
  displayLoadingAnimation(selectedModelKeys, {
    deepSearch: deepSearchEnabled,
    requestSessionId,
  });
};

const handleFeedbackAction = (buttonElement, actionType) => {
  const messageElement = buttonElement.closest(".message");
  if (!messageElement) return;

  const likeButton = messageElement.querySelector('[data-action="like"]');
  const dislikeButton = messageElement.querySelector('[data-action="dislike"]');
  if (!likeButton || !dislikeButton) return;

  const isLike = actionType === "like";
  const activeButton = isLike ? likeButton : dislikeButton;
  const inactiveButton = isLike ? dislikeButton : likeButton;
  const isActivating = !activeButton.classList.contains("is-active");

  likeButton.classList.remove("is-active");
  dislikeButton.classList.remove("is-active");
  if (isActivating) {
    activeButton.classList.add("is-active");
  }
  inactiveButton.classList.remove("is-active");
};

const retryIncomingMessage = (buttonElement) => {
  if (isActiveSessionGenerating()) return;

  const incomingMessage = buttonElement.closest(".message--incoming");
  if (!incomingMessage) return;

  let previousMessage = incomingMessage.previousElementSibling;
  while (previousMessage && !previousMessage.classList.contains("message--outgoing")) {
    previousMessage = previousMessage.previousElementSibling;
  }

  const previousUserText = previousMessage
    ?.querySelector(".message__text")
    ?.innerText?.trim();
  if (!previousUserText) return;

  promptInput.value = previousUserText;
  promptInput.dispatchEvent(new Event("input", { bubbles: true }));
  currentUserMessage = previousUserText;
  void handleOutgoingMessage();
};

// Toggle between light and dark themes
themeToggleButton.addEventListener("click", () => {
  const isLightTheme = themeRoot.classList.toggle("light_mode");
  localStorage.setItem("themeColor", isLightTheme ? "light_mode" : "dark_mode");

  // Update icon based on theme
  const newIconClass = isLightTheme ? "bx bx-moon" : "bx bx-sun";
  themeToggleButton.querySelector("i").className = newIconClass;
  document
    .querySelectorAll(".chats .message--incoming .message__text")
    .forEach((messageBody) => {
      void enhanceMessageBody(messageBody);
    });
});

// Voice input (Web Speech API)
voiceInputButton.addEventListener("click", () => {
  const SpeechRecognition =
    window.SpeechRecognition || window.webkitSpeechRecognition;

  if (!SpeechRecognition) {
    alert("当前浏览器不支持语音输入，请使用最新版 Chrome 或 Edge。");
    return;
  }

  const recognition = new SpeechRecognition();
  recognition.lang = "zh-CN";
  recognition.interimResults = false;
  recognition.maxAlternatives = 1;

  const icon = voiceInputButton.querySelector("i");
  icon.className = "bx bx-loader-alt bx-spin";

  recognition.onresult = (event) => {
    const transcript = event.results?.[0]?.[0]?.transcript?.trim();
    if (!transcript) return;
    promptInput.value = transcript;
    promptInput.dispatchEvent(new Event("input", { bubbles: true }));
    promptInput.focus();
  };

  recognition.onerror = () => {
    alert("语音识别失败，请再试一次。");
  };

  recognition.onend = () => {
    icon.className = "bx bx-microphone";
  };

  recognition.start();
});

attachButton?.addEventListener("click", () => {
  if (isActiveSessionGenerating()) return;
  attachMenu?.classList.toggle("hide");
});

attachMenu?.addEventListener("click", (event) => {
  const option = event.target.closest("[data-attach-mode]");
  if (!option || !fileInput) return;
  const mode = option.dataset.attachMode;
  fileInput.accept = mode === "image" ? IMAGE_ACCEPT : FILE_ACCEPT;
  attachMenu.classList.add("hide");
  fileInput.click();
});

fileInput?.addEventListener("change", (event) => {
  const files = Array.from(event.target?.files || []);
  if (files.length === 0) return;

  const nextAttachments = [...pendingAttachments];
  for (const file of files) {
    if (nextAttachments.length >= MAX_ATTACHMENT_COUNT) {
      alert(`最多可上传 ${MAX_ATTACHMENT_COUNT} 个附件`);
      break;
    }
    if (file.size > MAX_ATTACHMENT_SIZE) {
      alert(`"${file.name}" 超过 ${formatBytes(MAX_ATTACHMENT_SIZE)}，已跳过`);
      continue;
    }
    const duplicated = nextAttachments.some(
      (item) =>
        item.name === file.name &&
        item.size === file.size &&
        item.lastModified === file.lastModified
    );
    if (duplicated) continue;
    nextAttachments.push(file);
  }

  pendingAttachments = nextAttachments;
  renderPendingAttachments();
  fileInput.value = "";
});

document.addEventListener("click", (event) => {
  if (attachMenu && attachButton) {
    const clickedInsideMenu = attachMenu.contains(event.target);
    const clickedAttachButton = attachButton.contains(event.target);
    if (!clickedInsideMenu && !clickedAttachButton) {
      attachMenu.classList.add("hide");
    }
  }
  if (modelPickerPanel && modelPicker && !modelPicker.contains(event.target)) {
    closeModelPickerPanel();
  }
});

attachmentList?.addEventListener("click", (event) => {
  const removeButton = event.target.closest("[data-attachment-index]");
  if (!removeButton) return;
  const index = Number(removeButton.dataset.attachmentIndex);
  if (Number.isNaN(index) || index < 0 || index >= pendingAttachments.length) return;
  pendingAttachments.splice(index, 1);
  renderPendingAttachments();
});

promptInput.addEventListener("focus", () => setHeaderCursorPaused(true));
promptInput.addEventListener("blur", () => setHeaderCursorPaused(false));
promptInput.addEventListener("input", adjustPromptInputHeight);
promptInput.addEventListener("keydown", (e) => {
  if (e.key !== "Enter") return;
  if (e.shiftKey) return;
  e.preventDefault();
  void handleOutgoingMessage();
});
chatHistoryContainer.addEventListener(
  "scroll",
  () => {
    shouldAutoScroll = isActiveScrollNearBottom();
  },
  { passive: true }
);
window.addEventListener(
  "scroll",
  () => {
    shouldAutoScroll = isActiveScrollNearBottom();
  },
  { passive: true }
);
chatHistoryContainer.addEventListener("click", (event) => {
  const actionButton = event.target.closest(".message__action-btn");
  if (!actionButton) return;

  const actionType = actionButton.dataset.action;
  if (actionType === "copy") {
    copyMessageToClipboard(actionButton);
    return;
  }
  if (actionType === "like" || actionType === "dislike") {
    handleFeedbackAction(actionButton, actionType);
    return;
  }
  if (actionType === "retry") {
    retryIncomingMessage(actionButton);
  }
});

// Prevent default from submission and handle outgoing message
messageForm.addEventListener("submit", (e) => {
  e.preventDefault();
  void handleOutgoingMessage();
});

// Load saved chat history on page load
playHeaderTypingAnimation();
bindSidebarEvents();
bindModalEvents();
renderModelPickerOptions();
bindModelPickerEvents();
loadSavedChatHistory();
adjustPromptInputHeight();
void recordVisit();
if (authToken) {
  void (async () => {
    try {
      await fetchCurrentUser();
      await loadRemoteChatSessions();
    } catch {
      // Keep default local blank session when remote history fails.
    }
  })();
}
