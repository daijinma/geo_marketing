/**
 * ui.js - UI组件模块
 * 处理界面渲染和交互
 */

import { openDrawer } from './drawer.js';

/**
 * 显示错误提示
 * @param {string} message - 错误消息
 */
export function showError(message) {
    const toast = document.getElementById('error-toast');
    const messageEl = document.getElementById('error-message');
    
    if (toast && messageEl) {
        messageEl.textContent = message;
        toast.classList.remove('hidden');
        
        // 3秒后自动隐藏
        setTimeout(() => {
            toast.classList.add('hidden');
        }, 3000);
    } else {
        alert(`错误: ${message}`);
    }
}

/**
 * 显示成功提示
 * @param {string} message - 成功消息
 */
export function showSuccess(message) {
    const toast = document.getElementById('success-toast');
    const messageEl = document.getElementById('success-message');
    
    if (toast && messageEl) {
        messageEl.textContent = message;
        toast.classList.remove('hidden');
        
        // 3秒后自动隐藏
        setTimeout(() => {
            toast.classList.add('hidden');
        }, 3000);
    }
}

/**
 * 获取平台显示名称
 * @param {string} platform - 平台标识
 * @returns {string} 平台显示名称
 */
function getPlatformDisplayName(platform) {
    const nameMap = {
        'deepseek': 'DeepSeek',
        'doubao': '豆包',
        'bocha': '博查API'
    };
    return nameMap[platform.toLowerCase()] || platform;
}

/**
 * 获取平台状态图标
 * @param {string} status - 平台状态
 * @returns {string} 状态图标HTML
 */
function getPlatformStatusIcon(status) {
    const iconMap = {
        'completed': '<span class="platform-status status-completed">✅</span>',
        'failed': '<span class="platform-status status-failed">❌</span>',
        'pending': '<span class="platform-status status-pending">⏳</span>'
    };
    return iconMap[status] || '<span class="platform-status status-pending">⏸️</span>';
}

/**
 * 渲染查询结果（Tab形式）
 * @param {object} data - 任务数据
 */
export function renderResults(data) {
    const resultsSection = document.getElementById('results-section');
    const resultsContainer = document.getElementById('results-container');
    
    if (!resultsSection || !resultsContainer) return;

    resultsSection.classList.remove('hidden');
    resultsContainer.innerHTML = '';

    // 检查是否有按平台分组的数据
    const resultsByPlatform = data.results_by_platform;
    const platforms = data.platforms || [];

    if (!resultsByPlatform || Object.keys(resultsByPlatform).length === 0) {
        // 如果没有按平台分组的数据，使用旧的展示方式
        if (!data.query_tokens || data.query_tokens.length === 0) {
            resultsContainer.innerHTML = '<p class="no-results">暂无查询结果</p>';
            return;
        }

        // 使用旧的展示方式（向后兼容）
        data.query_tokens.forEach((queryToken, index) => {
            const queryBlock = createQueryBlock(queryToken, index);
            resultsContainer.appendChild(queryBlock);
        });
        return;
    }

    // 创建Tab容器
    const tabsContainer = document.createElement('div');
    tabsContainer.className = 'tabs-container';

    // 创建Tab导航
    const tabsNav = document.createElement('div');
    tabsNav.className = 'tabs-nav';

    // 创建Tab内容容器
    const tabsContent = document.createElement('div');
    tabsContent.className = 'tabs-content';

    // 为每个平台创建Tab
    let activeTabIndex = 0;
    platforms.forEach((platform, index) => {
        const platformLower = platform.toLowerCase();
        const platformData = resultsByPlatform[platformLower];
        
        if (!platformData) {
            // 如果平台数据不存在，创建一个空的状态
            platformData = {
                query_tokens: [],
                status: 'pending'
            };
        }

        // 创建Tab按钮
        const tabButton = document.createElement('button');
        tabButton.className = `tab-button ${index === 0 ? 'active' : ''}`;
        tabButton.setAttribute('data-platform', platformLower);
        tabButton.innerHTML = `
            ${getPlatformStatusIcon(platformData.status)}
            <span>${getPlatformDisplayName(platform)}</span>
        `;
        tabButton.addEventListener('click', () => {
            // 切换Tab
            document.querySelectorAll('.tab-button').forEach(btn => btn.classList.remove('active'));
            document.querySelectorAll('.tab-content').forEach(content => content.classList.remove('active'));
            tabButton.classList.add('active');
            const content = document.querySelector(`.tab-content[data-platform="${platformLower}"]`);
            if (content) {
                content.classList.add('active');
            }
        });
        tabsNav.appendChild(tabButton);

        // 创建Tab内容
        const tabContent = document.createElement('div');
        tabContent.className = `tab-content ${index === 0 ? 'active' : ''}`;
        tabContent.setAttribute('data-platform', platformLower);

        // 显示平台状态信息
        const platformInfo = document.createElement('div');
        platformInfo.className = 'platform-info';
        
        let statusText = '';
        if (platformData.status === 'completed') {
            statusText = `已完成 - 引用数: ${platformData.citations_count || 0}`;
            if (platformData.response_time_ms) {
                statusText += `, 耗时: ${(platformData.response_time_ms / 1000).toFixed(2)}秒`;
            }
        } else if (platformData.status === 'failed') {
            statusText = `执行失败`;
            if (platformData.error_message) {
                statusText += `: ${escapeHtml(platformData.error_message)}`;
            }
        } else {
            statusText = '执行中...';
        }
        
        platformInfo.innerHTML = `<span class="platform-status-text">${statusText}</span>`;
        tabContent.appendChild(platformInfo);

        // 显示查询结果
        if (platformData.query_tokens && platformData.query_tokens.length > 0) {
            platformData.query_tokens.forEach((queryToken, queryIndex) => {
                const queryBlock = createQueryBlock(queryToken, queryIndex);
                tabContent.appendChild(queryBlock);
            });
        } else {
            const noResults = document.createElement('p');
            noResults.className = 'no-results';
            if (platformData.status === 'pending') {
                noResults.textContent = '正在执行中，请稍候...';
            } else if (platformData.status === 'failed') {
                noResults.textContent = '执行失败，暂无结果';
            } else {
                noResults.textContent = '暂无查询结果';
            }
            tabContent.appendChild(noResults);
        }

        tabsContent.appendChild(tabContent);
    });

    tabsContainer.appendChild(tabsNav);
    tabsContainer.appendChild(tabsContent);
    resultsContainer.appendChild(tabsContainer);
}

/**
 * 创建查询块
 * @param {object} queryToken - 查询数据
 * @param {number} index - 索引
 * @returns {HTMLElement} 查询块元素
 */
function createQueryBlock(queryToken, index) {
    const queryBlock = document.createElement('div');
    queryBlock.className = 'query-block';

    // 查询文本
    const queryHeader = document.createElement('div');
    queryHeader.className = 'query-header';
    const queryText = document.createElement('span');
    queryText.className = 'query-text';
    queryText.textContent = queryToken.query;
    queryText.title = `点击查看"${queryToken.query}"的博查搜索结果`;
    queryText.style.cursor = 'pointer';
    queryText.addEventListener('click', () => {
        openDrawer(queryToken.query);
    });
    queryHeader.innerHTML = `<span class="query-label">查询 ${index + 1}:</span> `;
    queryHeader.appendChild(queryText);
    queryBlock.appendChild(queryHeader);

    // 分词标签（不再可点击）
    if (queryToken.tokens && queryToken.tokens.length > 0) {
        const tokensContainer = document.createElement('div');
        tokensContainer.className = 'tokens-container';
        
        queryToken.tokens.forEach(token => {
            const tokenTag = document.createElement('span');
            tokenTag.className = 'token-tag';
            tokenTag.textContent = token;
            tokensContainer.appendChild(tokenTag);
        });
        
        queryBlock.appendChild(tokensContainer);
    }

    // 关联链接卡片
    if (queryToken.citations && queryToken.citations.length > 0) {
        const citationsContainer = document.createElement('div');
        citationsContainer.className = 'citations-container';
        
        queryToken.citations.forEach(citation => {
            const card = renderLinkCard(citation);
            citationsContainer.appendChild(card);
        });
        
        queryBlock.appendChild(citationsContainer);
    } else {
        const noCitations = document.createElement('p');
        noCitations.className = 'no-citations';
        noCitations.textContent = '暂无关联链接';
        queryBlock.appendChild(noCitations);
    }

    return queryBlock;
}

/**
 * 渲染单个链接卡片
 * @param {object} citation - 链接信息
 * @returns {HTMLElement} 卡片元素
 */
function renderLinkCard(citation) {
    const card = document.createElement('div');
    card.className = 'link-card';

    const title = document.createElement('div');
    title.className = 'link-title';
    const titleLink = document.createElement('a');
    titleLink.href = citation.url;
    titleLink.target = '_blank';
    titleLink.rel = 'noopener noreferrer';
    titleLink.textContent = citation.title || citation.url;
    title.appendChild(titleLink);
    card.appendChild(title);

    const meta = document.createElement('div');
    meta.className = 'link-meta';
    meta.innerHTML = `
        <span class="link-domain">${escapeHtml(citation.domain || citation.site_name || '未知域名')}</span>
        <span class="link-index">[${citation.cite_index || '?'}]</span>
    `;
    card.appendChild(meta);

    if (citation.snippet) {
        const snippet = document.createElement('div');
        snippet.className = 'link-snippet';
        snippet.textContent = citation.snippet;
        card.appendChild(snippet);
    }

    const url = document.createElement('div');
    url.className = 'link-url';
    const urlLink = document.createElement('a');
    urlLink.href = citation.url;
    urlLink.target = '_blank';
    urlLink.rel = 'noopener noreferrer';
    urlLink.textContent = citation.url;
    urlLink.title = citation.url;
    url.appendChild(urlLink);
    card.appendChild(url);

    return card;
}

/**
 * HTML转义
 * @param {string} text - 原始文本
 * @returns {string} 转义后的文本
 */
function escapeHtml(text) {
    const div = document.createElement('div');
    div.textContent = text;
    return div.innerHTML;
}

/**
 * 获取选中的平台
 * @returns {string[]} 平台列表
 */
export function getSelectedPlatforms() {
    const platforms = [];
    const checkboxes = document.querySelectorAll('.platform-checkboxes input[type="checkbox"]:checked');
    checkboxes.forEach(checkbox => {
        platforms.push(checkbox.value);
    });
    return platforms;
}

/**
 * 获取输入的关键词
 * @returns {string} 关键词
 */
export function getKeyword() {
    const input = document.getElementById('keyword-input');
    return input ? input.value.trim() : '';
}

