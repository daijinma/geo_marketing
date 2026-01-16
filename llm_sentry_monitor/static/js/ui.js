/**
 * ui.js - UI组件模块
 * 处理界面渲染和交互
 */

import { openDrawer } from './drawer.js';
import { getTaskStatus, exportTaskData } from './api.js';
import { pollTaskStatus } from './task.js';

// 存储当前显示的 detail_logs 数据，用于 Excel 导出
let currentDetailLogs = [];

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
 * 渲染 domain 次数统计
 * @param {object} domainStats - domain 统计对象，格式：{ "domain.com": 3, ... }
 * @returns {HTMLElement} domain 统计元素
 */
function renderDomainStats(domainStats) {
    const statsContainer = document.createElement('div');
    statsContainer.className = 'domain-stats';
    
    if (!domainStats || Object.keys(domainStats).length === 0) {
        statsContainer.innerHTML = '<span class="domain-stats-empty">暂无域名统计</span>';
        return statsContainer;
    }
    
    // 按次数降序排序
    const sortedDomains = Object.entries(domainStats)
        .sort((a, b) => b[1] - a[1]);
    
    const statsList = document.createElement('div');
    statsList.className = 'domain-stats-list';
    
    sortedDomains.forEach(([domain, count]) => {
        const domainItem = document.createElement('span');
        domainItem.className = 'domain-stat-item';
        domainItem.innerHTML = `<span class="domain-name">${escapeHtml(domain)}</span> <span class="domain-count">(${count}次)</span>`;
        statsList.appendChild(domainItem);
    });
    
    statsContainer.appendChild(statsList);
    return statsContainer;
}

/**
 * 将 API 数据转换为树状结构
 * @param {object} data - 任务数据
 * @returns {Array} 树状结构数据
 */
function transformDataToTree(data) {
    const tree = [];
    const queryMap = new Map();
    
    // 如果没有 sub_query_logs，返回空数组（向后兼容）
    if (!data.sub_query_logs || !Array.isArray(data.sub_query_logs) || data.sub_query_logs.length === 0) {
        return tree;
    }
    
    // 如果没有 task_queries，也返回空数组
    if (!data.task_queries || !Array.isArray(data.task_queries) || data.task_queries.length === 0) {
        return tree;
    }
    
    // 创建 task_query_id 到 query 的映射
    const taskQueryMap = new Map();
    data.task_queries.forEach(tq => {
        taskQueryMap.set(tq.id, tq.query);
    });
    
    // 遍历 sub_query_logs，构建树状结构
    data.sub_query_logs.forEach(log => {
        // 过滤：只处理有 url 的记录（过滤掉 A 类型，取消单独保存的 sub_query）
        if (!log.url) {
            return; // 跳过没有 url 的记录
        }
        
        const taskQueryId = log.task_query_id;
        const query = taskQueryMap.get(taskQueryId);
        
        if (!query) {
            return; // 跳过没有对应 query 的 log
        }
        
        // 获取或创建 1 级节点（查询词）
        if (!queryMap.has(query)) {
            queryMap.set(query, {
                query: query,
                count: 0,
                sub_queries: new Map()
            });
        }
        
        const queryNode = queryMap.get(query);
        queryNode.count += 1;
        
        // 获取 sub_query（可能为空）
        const subQuery = log.sub_query || '';
        
        // 获取或创建 2 级节点（sub_query）
        if (!queryNode.sub_queries.has(subQuery)) {
            queryNode.sub_queries.set(subQuery, {
                sub_query: subQuery,
                count: 0,
                domain_stats: {},
                citations: []
            });
        }
        
        const subQueryNode = queryNode.sub_queries.get(subQuery);
        subQueryNode.count += 1;
        
        // 统计 domain
        if (log.domain) {
            if (!subQueryNode.domain_stats[log.domain]) {
                subQueryNode.domain_stats[log.domain] = 0;
            }
            subQueryNode.domain_stats[log.domain] += 1;
        }
        
        // 收集 citation 信息（如果有 url）
        if (log.url) {
            subQueryNode.citations.push({
                url: log.url,
                domain: log.domain,
                title: log.title,
                snippet: log.snippet,
                site_name: log.site_name,
                cite_index: log.cite_index
            });
        }
    });
    
    // 转换为数组并排序
    return Array.from(queryMap.values()).map(node => ({
        ...node,
        sub_queries: Array.from(node.sub_queries.values())
            .sort((a, b) => b.count - a.count)  // 按次数降序
    }));
}

/**
 * 切换树节点的展开/收起状态
 * @param {HTMLElement} header - 节点头部元素
 */
function toggleTreeNode(header) {
    const node = header.closest('.tree-node');
    if (!node) return;
    
    const content = node.querySelector('.tree-node-content');
    const toggle = header.querySelector('.tree-toggle');
    
    if (!content || !toggle) return;
    
    const isExpanded = !content.classList.contains('collapsed');
    
    if (isExpanded) {
        content.classList.add('collapsed');
        toggle.textContent = '▶';
    } else {
        content.classList.remove('collapsed');
        toggle.textContent = '▼';
    }
}

/**
 * 渲染树状结构
 * @param {Array} treeData - 树状结构数据
 * @param {HTMLElement} container - 容器元素
 */
function renderTreeStructure(treeData, container) {
    if (!treeData || treeData.length === 0) {
        container.innerHTML = '<p class="no-results">暂无查询结果</p>';
        return;
    }
    
    const treeContainer = document.createElement('div');
    treeContainer.className = 'tree-container';
    
    treeData.forEach(queryNode => {
        // 1级节点：查询词
        const level1Node = document.createElement('div');
        level1Node.className = 'tree-node tree-node-level-1';
        
        const level1Header = document.createElement('div');
        level1Header.className = 'tree-node-header';
        
        const level1Label = document.createElement('span');
        level1Label.className = 'tree-label';
        level1Label.innerHTML = `<span class="tree-query-text" title="点击查看博查搜索结果">${escapeHtml(queryNode.query)}</span> <span class="tree-count">(${queryNode.count}次)</span>`;
        
        // 使查询词可点击
        const queryTextEl = level1Label.querySelector('.tree-query-text');
        if (queryTextEl) {
            queryTextEl.style.cursor = 'pointer';
            queryTextEl.style.textDecoration = 'underline';
            queryTextEl.style.textDecorationColor = 'transparent';
            queryTextEl.addEventListener('click', (e) => {
                e.stopPropagation(); // 阻止触发展开/收起
                openDrawer(queryNode.query);
            });
            queryTextEl.addEventListener('mouseenter', () => {
                queryTextEl.style.textDecorationColor = 'var(--primary-color)';
                queryTextEl.style.color = 'var(--primary-color)';
            });
            queryTextEl.addEventListener('mouseleave', () => {
                queryTextEl.style.textDecorationColor = 'transparent';
                queryTextEl.style.color = '';
            });
        }
        
        const toggle = document.createElement('span');
        toggle.className = 'tree-toggle';
        toggle.textContent = '▼';
        level1Header.appendChild(toggle);
        level1Header.appendChild(level1Label);
        level1Header.addEventListener('click', (e) => {
            // 如果点击的是查询词文本，不触发展开/收起
            if (e.target.closest('.tree-query-text')) {
                return;
            }
            toggleTreeNode(level1Header);
        });
        
        const level1Content = document.createElement('div');
        level1Content.className = 'tree-node-content';
        
        // 2级节点：sub_query
        queryNode.sub_queries.forEach(subQueryNode => {
            const level2Node = document.createElement('div');
            level2Node.className = 'tree-node tree-node-level-2';
            
            const level2Header = document.createElement('div');
            level2Header.className = 'tree-node-header';
            const subQueryText = subQueryNode.sub_query || '(空)';
            level2Header.innerHTML = `
                <span class="tree-toggle">▼</span>
                <span class="tree-label">${escapeHtml(subQueryText)} <span class="tree-count">(${subQueryNode.count}次)</span></span>
            `;
            level2Header.addEventListener('click', () => toggleTreeNode(level2Header));
            
            const level2Content = document.createElement('div');
            level2Content.className = 'tree-node-content';
            
            // 3级：domain 统计
            const domainStatsSection = document.createElement('div');
            domainStatsSection.className = 'tree-node-detail-section';
            domainStatsSection.innerHTML = '<div class="tree-detail-label">域名统计:</div>';
            const domainStatsEl = renderDomainStats(subQueryNode.domain_stats);
            domainStatsSection.appendChild(domainStatsEl);
            level2Content.appendChild(domainStatsSection);
            
            // 3级：citations
            if (subQueryNode.citations && subQueryNode.citations.length > 0) {
                const citationsSection = document.createElement('div');
                citationsSection.className = 'tree-node-detail-section';
                citationsSection.innerHTML = '<div class="tree-detail-label">引用链接 (citations):</div>';
                
                const citationsContainer = document.createElement('div');
                citationsContainer.className = 'citations-container';
                subQueryNode.citations.forEach(citation => {
                    const card = renderLinkCard(citation);
                    citationsContainer.appendChild(card);
                });
                citationsSection.appendChild(citationsContainer);
                level2Content.appendChild(citationsSection);
            } else {
                const noCitations = document.createElement('p');
                noCitations.className = 'no-citations';
                noCitations.textContent = '暂无引用链接';
                level2Content.appendChild(noCitations);
            }
            
            level2Node.appendChild(level2Header);
            level2Node.appendChild(level2Content);
            level1Content.appendChild(level2Node);
        });
        
        level1Node.appendChild(level1Header);
        level1Node.appendChild(level1Content);
        treeContainer.appendChild(level1Node);
    });
    
    container.appendChild(treeContainer);
}

/**
 * 渲染查询结果（优先使用表格结构，否则使用树状结构）
 * @param {object} data - 任务数据
 */
export function renderResults(data) {
    const resultsSection = document.getElementById('results-section');
    const resultsContainer = document.getElementById('results-container');
    
    if (!resultsSection || !resultsContainer) return;

    resultsSection.classList.remove('hidden');
    resultsContainer.innerHTML = '';

    // 优先使用表格结构（如果有 summary_table 和 detail_logs）
    if (data.summary_table && data.detail_logs) {
        // 保存 detail_logs 数据用于 Excel 导出
        currentDetailLogs = data.detail_logs || [];
        
        // 渲染汇总表格标题
        const tableTitle = document.createElement('h3');
        tableTitle.textContent = '汇总表格';
        tableTitle.className = 'results-section-title';
        resultsContainer.appendChild(tableTitle);
        
        // 渲染汇总表格
        const tableContainer = document.createElement('div');
        tableContainer.className = 'table-container';
        renderTableResults(data.summary_table, tableContainer);
        resultsContainer.appendChild(tableContainer);
        
        // 渲染 domain 统计表格标题
        const domainTableTitle = document.createElement('h3');
        domainTableTitle.textContent = 'Domain 统计';
        domainTableTitle.className = 'results-section-title';
        resultsContainer.appendChild(domainTableTitle);
        
        // 构建并渲染 domain 统计表格
        const domainStats = buildDomainStats(data);
        const domainTableContainer = document.createElement('div');
        domainTableContainer.className = 'table-container';
        renderDomainTable(domainStats, domainTableContainer);
        resultsContainer.appendChild(domainTableContainer);
        
        // 渲染详细日志标题
        const logsTitle = document.createElement('h3');
        const logCount = data.detail_logs ? data.detail_logs.length : 0;
        logsTitle.textContent = `详细记录 (共${logCount}条)`;
        logsTitle.className = 'results-section-title';
        resultsContainer.appendChild(logsTitle);
        
        // 渲染详细日志列表
        const logsContainer = document.createElement('div');
        logsContainer.className = 'detail-logs-wrapper';
        renderDetailLogs(data.detail_logs, logsContainer);
        resultsContainer.appendChild(logsContainer);
        return;
    }

    // 如果没有表格数据，回退到树状结构（基于 sub_query_logs）
    const treeData = transformDataToTree(data);
    
    if (treeData.length > 0) {
        // 使用树状结构展示
        renderTreeStructure(treeData, resultsContainer);
        return;
    }
    
    // 如果没有 sub_query_logs，回退到旧的展示方式（向后兼容）
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

    // 如果有平台数据但没有 sub_query_logs，显示平台状态信息
        const platformInfo = document.createElement('div');
        platformInfo.className = 'platform-info';
        
    let statusText = '查询结果';
    if (platforms.length > 0) {
        const platformData = resultsByPlatform[platforms[0].toLowerCase()];
        if (platformData) {
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
        }
        }
        
        platformInfo.innerHTML = `<span class="platform-status-text">${statusText}</span>`;
    resultsContainer.appendChild(platformInfo);

    // 显示查询结果（使用旧的展示方式）
    if (data.query_tokens && data.query_tokens.length > 0) {
        data.query_tokens.forEach((queryToken, queryIndex) => {
                const queryBlock = createQueryBlock(queryToken, queryIndex);
            resultsContainer.appendChild(queryBlock);
            });
        } else {
            const noResults = document.createElement('p');
            noResults.className = 'no-results';
                noResults.textContent = '暂无查询结果';
        resultsContainer.appendChild(noResults);
    }
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
 * 根据 task_id 列表加载历史数据
 * @param {number[]} taskIds - 任务ID列表
 */
export async function loadTaskByIds(taskIds) {
    try {
        const statusData = await getTaskStatus(taskIds);
        
        if (statusData.status === 'none') {
            showError('未找到指定的任务');
            return;
        }
        
        // 更新 task_id 输入框
        const taskIdInput = document.getElementById('task-id-input');
        if (taskIdInput) {
            const currentIds = taskIdInput.value.trim();
            const existingIds = currentIds ? currentIds.split(',').map(id => id.trim()) : [];
            const allIds = [...new Set([...existingIds, ...taskIds.map(String)])];
            taskIdInput.value = allIds.join(',');
        }
        
        // 保存最近一次的 task_id 到 localStorage（取第一个）
        if (taskIds.length > 0) {
            saveLastTaskId(taskIds[0]);
        }
        
        // 显示 task_id 区域
        const taskIdSection = document.getElementById('task-id-section');
        if (taskIdSection) {
            taskIdSection.classList.remove('hidden');
        }
        
        // 渲染结果
        if (statusData.data) {
            if (statusData.status === 'multiple') {
                // 多个任务
                renderMultipleTasks(statusData.data.tasks || []);
            } else {
                // 单个任务
                renderResults(statusData.data);
                
                // 如果任务还在进行中（pending），继续轮询任务状态
                if (statusData.status === 'pending' && taskIds.length === 1) {
                    const taskId = parseInt(taskIds[0], 10);
                    if (!isNaN(taskId)) {
                        pollTaskStatus(taskId);
                    }
                }
            }
        }
        
        if (statusData.status === 'pending') {
            showSuccess(`任务 ${taskIds[0]} 正在执行中，已恢复轮询...`);
        } else {
            showSuccess(`成功加载 ${taskIds.length} 个任务的数据`);
        }
    } catch (error) {
        showError(`加载任务失败: ${error.message}`);
        throw error;
    }
}

/**
 * 保存最近一次的 task_id 到 localStorage
 * @param {number|string} taskId - 任务ID
 */
function saveLastTaskId(taskId) {
    try {
        localStorage.setItem('lastTaskId', String(taskId));
    } catch (error) {
        console.warn('保存 task_id 到 localStorage 失败:', error);
    }
}

/**
 * 导出 Excel 文件（基于 detail_logs 数据）
 * @param {Array} detailLogs - 详细日志数据
 */
function exportToExcel(detailLogs) {
    if (!detailLogs || detailLogs.length === 0) {
        showError('没有可导出的数据');
        return;
    }
    
    // 检查 SheetJS 是否已加载
    if (typeof XLSX === 'undefined') {
        showError('Excel 导出库未加载，请刷新页面重试');
        return;
    }
    
    try {
        // 准备数据
        const worksheetData = [];
        
        // 表头
        worksheetData.push([
            '任务ID',
            '查询词',
            '轮次',
            '平台',
            'Sub Query',
            '时间',
            '域名',
            '标题',
            '摘要',
            '网址'
        ]);
        
        // 数据行
        detailLogs.forEach(log => {
            // 过滤：只导出有 url 的记录
            if (!log.url) {
                return;
            }
            
            worksheetData.push([
                log.task_id || '',
                log.query || '',
                log.round !== null && log.round !== undefined ? log.round : 'N/A',
                log.platform || '',
                log.sub_query || '',
                log.time || 'N/A',
                log.domain || '',
                log.title || '',
                log.snippet || '',
                log.url || ''
            ]);
        });
        
        // 创建工作簿
        const wb = XLSX.utils.book_new();
        const ws = XLSX.utils.aoa_to_sheet(worksheetData);
        
        // 设置列宽
        ws['!cols'] = [
            { wch: 10 },  // 任务ID
            { wch: 30 },  // 查询词
            { wch: 8 },   // 轮次
            { wch: 12 },  // 平台
            { wch: 40 },  // Sub Query
            { wch: 20 },  // 时间
            { wch: 25 },  // 域名
            { wch: 50 },  // 标题
            { wch: 60 },  // 摘要
            { wch: 50 }   // 网址
        ];
        
        // 添加工作表
        XLSX.utils.book_append_sheet(wb, ws, '详细记录');
        
        // 生成文件名
        const timestamp = new Date().toISOString().slice(0, 19).replace(/:/g, '-');
        const filename = `任务数据_${timestamp}.xlsx`;
        
        // 导出文件
        XLSX.writeFile(wb, filename);
        
        showSuccess(`Excel 文件已导出: ${filename}`);
    } catch (error) {
        showError(`导出 Excel 失败: ${error.message}`);
        console.error('Excel 导出错误:', error);
    }
}

/**
 * 下载任务数据（Excel 格式）
 * @param {number[]} taskIds - 任务ID列表
 */
export async function downloadTaskData(taskIds) {
    try {
        // 如果当前有显示的 detail_logs 数据，直接使用
        if (currentDetailLogs && currentDetailLogs.length > 0) {
            exportToExcel(currentDetailLogs);
            return;
        }
        
        // 如果没有当前数据，从 API 获取
        const statusData = await getTaskStatus(taskIds);
        
        if (statusData.status === 'none') {
            showError('未找到指定的任务');
            return;
        }
        
        let detailLogs = [];
        
        if (statusData.status === 'multiple') {
            // 多个任务
            const tasks = statusData.data.tasks || [];
            tasks.forEach(task => {
                if (task.detail_logs && Array.isArray(task.detail_logs)) {
                    detailLogs.push(...task.detail_logs);
                }
            });
        } else {
            // 单个任务
            if (statusData.data && statusData.data.detail_logs) {
                detailLogs = statusData.data.detail_logs;
            }
        }
        
        if (detailLogs.length === 0) {
            showError('没有可导出的数据');
            return;
        }
        
        exportToExcel(detailLogs);
    } catch (error) {
        showError(`下载数据失败: ${error.message}`);
        throw error;
    }
}

/**
 * 构建 domain 统计数据结构
 * @param {object} data - 任务数据（包含 detail_logs 或 sub_query_logs）
 * @returns {object} 统计对象，格式：{ domains: [...], subQueries: [...], stats: { domain: { total: count, sub_query1: count, ... } } }
 */
function buildDomainStats(data) {
    const stats = {};
    const subQueriesSet = new Set();
    
    // 优先使用 detail_logs，如果没有则使用 sub_query_logs
    const logs = data.detail_logs || data.sub_query_logs || [];
    
    // 遍历日志数据，统计 domain 信息
    logs.forEach(log => {
        // 过滤：只处理有 url 的记录（过滤掉 A 类型）
        if (!log.url) {
            return;
        }
        
        const domain = log.domain || '';
        const subQuery = log.sub_query || '';
        
        // 如果 domain 为空，跳过
        if (!domain) {
            return;
        }
        
        // 初始化 domain 统计
        if (!stats[domain]) {
            stats[domain] = {
                total: 0,
                subQueries: {}
            };
        }
        
        // 统计总次数
        stats[domain].total += 1;
        
        // 统计按 sub_query 的次数
        const subQueryKey = subQuery || '(空)';
        if (!stats[domain].subQueries[subQueryKey]) {
            stats[domain].subQueries[subQueryKey] = 0;
        }
        stats[domain].subQueries[subQueryKey] += 1;
        
        // 收集所有唯一的 sub_query（包括空值）
        if (subQuery) {
            subQueriesSet.add(subQuery);
        } else {
            // 标记存在空 sub_query
            subQueriesSet.add('(空)');
        }
    });
    
    // 转换为数组并排序（按总次数降序）
    const domains = Object.keys(stats).sort((a, b) => stats[b].total - stats[a].total);
    
    // 将 subQueriesSet 转换为排序后的数组
    const subQueries = Array.from(subQueriesSet).sort();
    // 如果存在 '(空)'，将其移到数组开头
    const emptyIndex = subQueries.indexOf('(空)');
    if (emptyIndex > 0) {
        subQueries.splice(emptyIndex, 1);
        subQueries.unshift('(空)');
    }
    
    return {
        domains,
        subQueries,
        stats
    };
}

/**
 * 渲染 domain 统计表格
 * @param {object} domainStats - domain 统计对象（buildDomainStats 的返回值）
 * @param {HTMLElement} container - 容器元素
 */
function renderDomainTable(domainStats, container) {
    if (!domainStats || !domainStats.domains || domainStats.domains.length === 0) {
        container.innerHTML = '<p class="no-results">暂无域名统计</p>';
        return;
    }
    
    const tableWrapper = document.createElement('div');
    tableWrapper.className = 'table-wrapper';
    
    const table = document.createElement('table');
    table.className = 'results-table domain-table table-mini';
    
    // 表头
    const thead = document.createElement('thead');
    const headerRow = document.createElement('tr');
    
    // 第一列：Domain
    const domainHeader = document.createElement('th');
    domainHeader.className = 'domain-column';
    domainHeader.textContent = 'Domain';
    headerRow.appendChild(domainHeader);
    
    // 第二列：总计
    const totalHeader = document.createElement('th');
    totalHeader.textContent = '总计';
    headerRow.appendChild(totalHeader);
    
    // 后续列：各个 sub_query
    domainStats.subQueries.forEach(subQuery => {
        const subQueryHeader = document.createElement('th');
        subQueryHeader.textContent = escapeHtml(subQuery);
        headerRow.appendChild(subQueryHeader);
    });
    
    thead.appendChild(headerRow);
    table.appendChild(thead);
    
    // 表体
    const tbody = document.createElement('tbody');
    domainStats.domains.forEach(domain => {
        const tr = document.createElement('tr');
        
        // Domain 列
        const domainCell = document.createElement('td');
        domainCell.className = 'domain-column';
        domainCell.textContent = escapeHtml(domain);
        domainCell.title = domain; // 添加 title 以便鼠标悬停时显示完整内容
        tr.appendChild(domainCell);
        
        // 总计列
        const totalCell = document.createElement('td');
        totalCell.textContent = domainStats.stats[domain].total;
        tr.appendChild(totalCell);
        
        // 各 sub_query 列
        domainStats.subQueries.forEach(subQuery => {
            const subQueryCell = document.createElement('td');
            const count = domainStats.stats[domain].subQueries[subQuery] || 0;
            subQueryCell.textContent = count;
            tr.appendChild(subQueryCell);
        });
        
        tbody.appendChild(tr);
    });
    table.appendChild(tbody);
    
    tableWrapper.appendChild(table);
    container.appendChild(tableWrapper);
}

/**
 * 渲染表格结果（查询词、平台、sub_query、sub_query次数）
 * @param {Array} summaryTableData - 汇总表格数据
 * @param {HTMLElement} container - 容器元素
 */
function renderTableResults(summaryTableData, container) {
    if (!summaryTableData || summaryTableData.length === 0) {
        container.innerHTML = '<p class="no-results">暂无表格数据</p>';
        return;
    }
    
    const tableWrapper = document.createElement('div');
    tableWrapper.className = 'table-wrapper';
    
    const table = document.createElement('table');
    table.className = 'results-table';
    
    // 表头
    const thead = document.createElement('thead');
    thead.innerHTML = `
        <tr>
            <th>查询词</th>
            <th>平台</th>
            <th>Sub Query</th>
            <th>Sub Query次数</th>
        </tr>
    `;
    table.appendChild(thead);
    
    // 表体
    const tbody = document.createElement('tbody');
    summaryTableData.forEach(row => {
        const tr = document.createElement('tr');
        tr.innerHTML = `
            <td>${escapeHtml(row.query || '')}</td>
            <td>${escapeHtml(row.platform || '')}</td>
            <td>${escapeHtml(row.sub_query || '')}</td>
            <td>${row.count || 0}</td>
        `;
        tbody.appendChild(tr);
    });
    table.appendChild(tbody);
    
    tableWrapper.appendChild(table);
    container.appendChild(tableWrapper);
}

/**
 * 渲染详细日志列表（单行横向滚动，超链样式）
 * @param {Array} detailLogs - 详细日志数据
 * @param {HTMLElement} container - 容器元素
 */
function renderDetailLogs(detailLogs, container) {
    if (!detailLogs || detailLogs.length === 0) {
        container.innerHTML = '<p class="no-results">暂无详细日志</p>';
        return;
    }
    
    const logsContainer = document.createElement('div');
    logsContainer.className = 'detail-logs-container';
    
    detailLogs.forEach(log => {
        // 过滤：只显示有 url 的记录（过滤掉 A 类型，虽然后端已过滤，但作为保险）
        if (!log.url) {
            return; // 跳过没有 url 的记录
        }
        
        const logItem = document.createElement('div');
        logItem.className = 'detail-log-item';
        
        const parts = [
            `任务ID: ${log.task_id || ''}`,
            `查询词: ${escapeHtml(log.query || '')}`,
            `轮次: ${log.round !== null && log.round !== undefined ? log.round : 'N/A'}`,
            `平台: ${escapeHtml(log.platform || '')}`,
            `Sub Query: ${escapeHtml(log.sub_query || '')}`,
            `时间: ${log.time || 'N/A'}`,
            `域名: ${escapeHtml(log.domain || '')}`,
            `标题: ${escapeHtml(log.title || 'N/A')}`,
            `摘要: ${escapeHtml(log.snippet || 'N/A')}`,
            log.url ? `<a href="${escapeHtml(log.url)}" target="_blank" rel="noopener noreferrer" class="detail-log-link">${escapeHtml(log.url)}</a>` : '网址: N/A'
        ];
        
        logItem.innerHTML = parts.join(' | ');
        logsContainer.appendChild(logItem);
    });
    
    container.appendChild(logsContainer);
}

/**
 * 渲染多个任务的数据
 * @param {object[]} tasks - 任务数据列表
 */
function renderMultipleTasks(tasks) {
    const resultsSection = document.getElementById('results-section');
    const resultsContainer = document.getElementById('results-container');
    
    if (!resultsSection || !resultsContainer) return;
    
    resultsSection.classList.remove('hidden');
    resultsContainer.innerHTML = '';
    
    if (tasks.length === 0) {
        resultsContainer.innerHTML = '<p class="no-results">暂无任务数据</p>';
        return;
    }
    
    // 合并所有任务的汇总表格数据
    const allSummaryTableData = [];
    const allDetailLogs = [];
    const allSubQueryLogs = [];
    
    tasks.forEach(task => {
        // 合并汇总表格数据
        if (task.summary_table && Array.isArray(task.summary_table)) {
            allSummaryTableData.push(...task.summary_table);
        }
        
        // 合并详细日志数据
        if (task.detail_logs && Array.isArray(task.detail_logs)) {
            allDetailLogs.push(...task.detail_logs);
        }
        
        // 合并 sub_query_logs 数据（用于 domain 统计，如果没有 detail_logs）
        if (task.sub_query_logs && Array.isArray(task.sub_query_logs)) {
            allSubQueryLogs.push(...task.sub_query_logs);
        }
    });
    
    // 保存 detail_logs 数据用于 Excel 导出
    currentDetailLogs = allDetailLogs;
    
    // 渲染汇总表格标题
    const tableTitle = document.createElement('h3');
    tableTitle.textContent = '汇总表格';
    tableTitle.className = 'results-section-title';
    resultsContainer.appendChild(tableTitle);
    
    // 渲染汇总表格
    const tableContainer = document.createElement('div');
    tableContainer.className = 'table-container';
    renderTableResults(allSummaryTableData, tableContainer);
    resultsContainer.appendChild(tableContainer);
    
    // 渲染 domain 统计表格标题
    const domainTableTitle = document.createElement('h3');
    domainTableTitle.textContent = 'Domain 统计';
    domainTableTitle.className = 'results-section-title';
    resultsContainer.appendChild(domainTableTitle);
    
    // 构建并渲染 domain 统计表格（合并所有任务的数据）
    const mergedData = {
        detail_logs: allDetailLogs.length > 0 ? allDetailLogs : undefined,
        sub_query_logs: allSubQueryLogs.length > 0 ? allSubQueryLogs : undefined
    };
    const domainStats = buildDomainStats(mergedData);
    const domainTableContainer = document.createElement('div');
    domainTableContainer.className = 'table-container';
    renderDomainTable(domainStats, domainTableContainer);
    resultsContainer.appendChild(domainTableContainer);
    
    // 渲染详细日志标题
    const logsTitle = document.createElement('h3');
    const logCount = allDetailLogs ? allDetailLogs.length : 0;
    logsTitle.textContent = `详细记录 (共${logCount}条)`;
    logsTitle.className = 'results-section-title';
    resultsContainer.appendChild(logsTitle);
    
    // 渲染详细日志列表
    const logsContainer = document.createElement('div');
    logsContainer.className = 'detail-logs-wrapper';
    renderDetailLogs(allDetailLogs, logsContainer);
    resultsContainer.appendChild(logsContainer);
}

/**
 * 更新 task_id 输入框，替换为新的 task_id（新请求时替换，不是追加）
 * @param {number} taskId - 任务ID
 */
export function addTaskId(taskId) {
    const taskIdInput = document.getElementById('task-id-input');
    if (!taskIdInput) return;
    
    // 直接替换为新的 task_id（新请求时替换）
    taskIdInput.value = String(taskId);
    
    // task_id 区域已经常显示，无需操作
}

/**
 * 获取输入的查询条件列表（多行）
 * @returns {string[]} 查询条件列表
 */
export function getKeywords() {
    const input = document.getElementById('keyword-input');
    if (!input) return [];
    
    const value = input.value.trim();
    if (!value) return [];
    
    // 按换行分割，过滤空行
    return value.split('\n')
        .map(line => line.trim())
        .filter(line => line.length > 0);
}

/**
 * 获取查询次数
 * @returns {number} 查询次数
 */
export function getQueryCount() {
    const input = document.getElementById('query-count-input');
    if (!input) return 1;
    
    const value = parseInt(input.value, 10);
    return isNaN(value) || value < 1 ? 1 : value;
}


