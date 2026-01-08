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
 * 渲染查询结果
 * @param {object} data - 任务数据
 */
export function renderResults(data) {
    const resultsSection = document.getElementById('results-section');
    const resultsContainer = document.getElementById('results-container');
    
    if (!resultsSection || !resultsContainer) return;

    resultsSection.classList.remove('hidden');
    resultsContainer.innerHTML = '';

    if (!data.query_tokens || data.query_tokens.length === 0) {
        resultsContainer.innerHTML = '<p class="no-results">暂无查询结果</p>';
        return;
    }

    // 遍历每个查询
    data.query_tokens.forEach((queryToken, index) => {
        const queryBlock = document.createElement('div');
        queryBlock.className = 'query-block';

        // 查询文本
        const queryHeader = document.createElement('div');
        queryHeader.className = 'query-header';
        queryHeader.innerHTML = `<span class="query-label">查询 ${index + 1}:</span> <span class="query-text">${escapeHtml(queryToken.query)}</span>`;
        queryBlock.appendChild(queryHeader);

        // 分词标签
        if (queryToken.tokens && queryToken.tokens.length > 0) {
            const tokensContainer = document.createElement('div');
            tokensContainer.className = 'tokens-container';
            
            queryToken.tokens.forEach(token => {
                const tokenTag = document.createElement('span');
                tokenTag.className = 'token-tag';
                tokenTag.textContent = token;
                tokenTag.title = `点击查看"${token}"的博查搜索结果`;
                tokenTag.addEventListener('click', () => {
                    openDrawer(token);
                });
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

        resultsContainer.appendChild(queryBlock);
    });
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

