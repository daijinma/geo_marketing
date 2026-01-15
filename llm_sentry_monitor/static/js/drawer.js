/**
 * drawer.js - 抽屉组件模块
 * 处理博查搜索结果的抽屉展示
 */

import { searchBocha } from './api.js';

/**
 * 打开抽屉并加载博查搜索结果
 * @param {string} query - 查询词
 */
export async function openDrawer(query) {
    const drawer = document.getElementById('drawer');
    const drawerOverlay = document.getElementById('drawer-overlay');
    const drawerTitle = document.getElementById('drawer-title');
    const drawerLoading = document.getElementById('drawer-loading');
    const drawerError = document.getElementById('drawer-error');
    const drawerResults = document.getElementById('drawer-results');

    if (!drawer || !drawerOverlay) return;

    // 显示抽屉
    drawer.classList.add('open');
    drawerOverlay.classList.remove('hidden');

    // 设置标题
    if (drawerTitle) {
        drawerTitle.textContent = `博查搜索结果: "${query}"`;
    }

    // 显示加载状态
    if (drawerLoading) {
        drawerLoading.classList.remove('hidden');
    }
    if (drawerError) {
        drawerError.classList.add('hidden');
        drawerError.textContent = '';
    }
    if (drawerResults) {
        drawerResults.innerHTML = '';
    }

    try {
        // 调用博查API
        const result = await searchBocha(query);

        // 隐藏加载状态
        if (drawerLoading) {
            drawerLoading.classList.add('hidden');
        }

        if (!result.success) {
            // 显示错误
            if (drawerError) {
                drawerError.textContent = result.error || '搜索失败';
                drawerError.classList.remove('hidden');
            }
            return;
        }

        // 渲染结果
        if (drawerResults && result.data) {
            renderBochaResults(drawerResults, result.data);
        }
    } catch (error) {
        // 隐藏加载状态
        if (drawerLoading) {
            drawerLoading.classList.add('hidden');
        }

        // 显示错误
        if (drawerError) {
            drawerError.textContent = `搜索失败: ${error.message}`;
            drawerError.classList.remove('hidden');
        }
    }
}

/**
 * 关闭抽屉
 */
export function closeDrawer() {
    const drawer = document.getElementById('drawer');
    const drawerOverlay = document.getElementById('drawer-overlay');

    if (drawer) {
        drawer.classList.remove('open');
    }
    if (drawerOverlay) {
        drawerOverlay.classList.add('hidden');
    }
}

/**
 * 渲染博查搜索结果
 * @param {HTMLElement} container - 容器元素
 * @param {object} data - 搜索结果数据
 */
function renderBochaResults(container, data) {
    container.innerHTML = '';

    // 显示完整文本（如果有）
    if (data.full_text) {
        const fullTextSection = document.createElement('div');
        fullTextSection.className = 'bocha-fulltext';
        const fullTextTitle = document.createElement('h4');
        fullTextTitle.textContent = '摘要';
        fullTextSection.appendChild(fullTextTitle);
        const fullTextContent = document.createElement('div');
        fullTextContent.className = 'fulltext-content';
        fullTextContent.textContent = data.full_text;
        fullTextSection.appendChild(fullTextContent);
        container.appendChild(fullTextSection);
    }

    // 显示拓展词（如果有）
    if (data.queries && data.queries.length > 0) {
        const queriesSection = document.createElement('div');
        queriesSection.className = 'bocha-queries';
        const queriesTitle = document.createElement('h4');
        queriesTitle.textContent = '拓展查询词';
        queriesSection.appendChild(queriesTitle);
        const queriesList = document.createElement('div');
        queriesList.className = 'queries-list';
        data.queries.forEach(query => {
            const queryTag = document.createElement('span');
            queryTag.className = 'query-tag';
            queryTag.textContent = query;
            queriesList.appendChild(queryTag);
        });
        queriesSection.appendChild(queriesList);
        container.appendChild(queriesSection);
    }

    // 显示搜索结果
    if (data.citations && data.citations.length > 0) {
        const resultsTitle = document.createElement('h4');
        resultsTitle.textContent = `搜索结果 (${data.citations.length} 个)`;
        container.appendChild(resultsTitle);

        const resultsList = document.createElement('div');
        resultsList.className = 'bocha-results-list';

        data.citations.forEach((citation, index) => {
            const resultCard = document.createElement('div');
            resultCard.className = 'bocha-result-card';

            const cardTitle = document.createElement('div');
            cardTitle.className = 'bocha-card-title';
            const titleLink = document.createElement('a');
            titleLink.href = citation.url;
            titleLink.target = '_blank';
            titleLink.rel = 'noopener noreferrer';
            titleLink.textContent = citation.title || citation.url;
            cardTitle.appendChild(titleLink);
            resultCard.appendChild(cardTitle);

            const cardMeta = document.createElement('div');
            cardMeta.className = 'bocha-card-meta';
            cardMeta.innerHTML = `
                <span class="bocha-domain">${escapeHtml(citation.domain || citation.site_name || '未知域名')}</span>
                <span class="bocha-index">[${citation.cite_index || index + 1}]</span>
            `;
            resultCard.appendChild(cardMeta);

            if (citation.snippet) {
                const cardSnippet = document.createElement('div');
                cardSnippet.className = 'bocha-card-snippet';
                cardSnippet.textContent = citation.snippet;
                resultCard.appendChild(cardSnippet);
            }

            const cardUrl = document.createElement('div');
            cardUrl.className = 'bocha-card-url';
            const urlLink = document.createElement('a');
            urlLink.href = citation.url;
            urlLink.target = '_blank';
            urlLink.rel = 'noopener noreferrer';
            urlLink.textContent = citation.url;
            urlLink.title = citation.url;
            cardUrl.appendChild(urlLink);
            resultCard.appendChild(cardUrl);

            resultsList.appendChild(resultCard);
        });

        container.appendChild(resultsList);
    } else {
        const noResults = document.createElement('p');
        noResults.className = 'no-results';
        noResults.textContent = '未找到搜索结果';
        container.appendChild(noResults);
    }
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

// 绑定关闭事件
document.addEventListener('DOMContentLoaded', () => {
    const drawerClose = document.getElementById('drawer-close');
    const drawerOverlay = document.getElementById('drawer-overlay');

    if (drawerClose) {
        drawerClose.addEventListener('click', closeDrawer);
    }

    if (drawerOverlay) {
        drawerOverlay.addEventListener('click', closeDrawer);
    }

    // ESC键关闭
    document.addEventListener('keydown', (e) => {
        if (e.key === 'Escape') {
            closeDrawer();
        }
    });
});


