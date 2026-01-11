/**
 * task.js - 任务管理逻辑模块
 * 处理任务创建、状态轮询等逻辑
 */

import { createTask, getTaskStatus, validateApiResponse } from './api.js';
import { showError, showSuccess } from './ui.js';
import { renderResults } from './ui.js';

const POLL_INTERVAL = 2000; // 轮询间隔：2秒
const MAX_POLL_COUNT = 300; // 最大轮询次数：10分钟

let currentTaskId = null;
let pollTimer = null;

/**
 * 启动查询任务
 * @param {string[]} keywords - 关键词列表
 * @param {string[]} platforms - 平台列表
 */
export async function startTask(keywords, platforms) {
    try {
        // 重置状态
        currentTaskId = null;
        if (pollTimer) {
            clearInterval(pollTimer);
            pollTimer = null;
        }

        // 创建任务
        const result = await createTask(keywords, platforms);
        currentTaskId = result.task_id;

        showSuccess(`任务创建成功，任务ID: ${currentTaskId}`);
        
        // 开始轮询
        pollTaskStatus(currentTaskId);
    } catch (error) {
        showError(`创建任务失败: ${error.message}`);
        throw error;
    }
}

/**
 * 轮询任务状态
 * @param {number} taskId - 任务ID
 */
export function pollTaskStatus(taskId) {
    let pollCount = 0;
    
    const updateStatus = async () => {
        try {
            pollCount++;
            console.log(`[轮询 ${pollCount}] 查询任务 ${taskId} 状态...`);
            
            const statusData = await getTaskStatus(taskId);
            const status = statusData.status;
            const data = statusData.data;
            
            console.log(`[轮询 ${pollCount}] 任务状态: ${status}`, data?.platform_progress);

            // 更新状态显示
            updateStatusUI(status, data);
            
            // 检查所有平台是否已完成（即使状态还是 pending）
            let allPlatformsCompleted = false;
            if (data && data.platform_progress) {
                const progress = data.platform_progress;
                // 如果所有平台都已完成（completed + failed === total），则认为任务完成
                allPlatformsCompleted = (progress.completed + progress.failed) >= progress.total && progress.total > 0;
            }
            
            // 如果任务执行中但已有部分结果，也显示出来
            if (status === 'pending' && data && data.results_by_platform) {
                const hasResults = Object.values(data.results_by_platform).some(
                    platformData => platformData.query_tokens && platformData.query_tokens.length > 0
                );
                if (hasResults) {
                    renderResults(data);
                }
            }

            // 如果所有平台都已完成，即使状态还是 pending，也停止轮询
            if (allPlatformsCompleted && status === 'pending') {
                console.log('[轮询停止] 所有平台已完成，停止轮询');
                if (pollTimer) {
                    clearInterval(pollTimer);
                    pollTimer = null;
                    console.log('[轮询停止] pollTimer 已清除');
                }
                // 更新UI显示为完成状态
                if (data) {
                    renderResults(data);
                    if (data.platform_progress) {
                        const progress = data.platform_progress;
                        const summary = `任务完成！已完成: ${progress.completed}/${progress.total}, 失败: ${progress.failed}`;
                        showSuccess(summary);
                        // 更新状态文本
                        const statusTextEl = document.getElementById('status-text');
                        if (statusTextEl) {
                            statusTextEl.textContent = `已完成 (成功: ${progress.completed}, 失败: ${progress.failed}/${progress.total})`;
                            statusTextEl.className = 'status-text status-done';
                        }
                        // 设置进度条为 100% 并隐藏
                        const progressBarEl = document.getElementById('progress-bar');
                        const progressFillEl = progressBarEl ? progressBarEl.querySelector('.progress-fill') : null;
                        if (progressBarEl && progressFillEl) {
                            progressFillEl.classList.remove('animated');
                            progressFillEl.style.width = '100%';
                            progressFillEl.style.transition = 'width 0.3s ease-out';
                            setTimeout(() => {
                                progressBarEl.classList.add('hidden');
                            }, 500);
                        }
                    }
                }
                return;
            }

            if (status === 'completed' || status === 'done') {
                // 任务完成，立即停止轮询
                console.log(`[轮询停止] 任务状态为 ${status}，停止轮询`);
                if (pollTimer) {
                    clearInterval(pollTimer);
                    pollTimer = null;
                    console.log('[轮询停止] pollTimer 已清除');
                }
                
                // 验证API响应格式
                const validation = validateApiResponse(statusData);
                if (!validation.valid) {
                    console.warn('API响应格式验证失败:', validation.errors);
                    showError(`数据格式验证失败: ${validation.errors.join(', ')}`);
                }
                
                if (data) {
                    renderResults(data);
                    // 显示平台进度汇总
                    if (data.platform_progress) {
                        const progress = data.platform_progress;
                        const summary = `任务完成！已完成: ${progress.completed}/${progress.total}, 失败: ${progress.failed}`;
                        showSuccess(summary);
                    } else {
                        showSuccess('任务执行完成！');
                    }
                } else {
                    showError('任务完成但未获取到结果数据');
                }
                return; // 确保不再继续执行
            } else if (status === 'none') {
                // 任务不存在，停止轮询
                if (pollTimer) {
                    clearInterval(pollTimer);
                    pollTimer = null;
                }
                showError('任务不存在');
                return; // 确保不再继续执行
            } else if (pollCount >= MAX_POLL_COUNT) {
                // 超时，停止轮询
                if (pollTimer) {
                    clearInterval(pollTimer);
                    pollTimer = null;
                }
                showError('任务执行超时，请稍后手动查询');
                return; // 确保不再继续执行
            }
            // status === 'pending' 时继续轮询
        } catch (error) {
            // 错误情况下也停止轮询
            if (pollTimer) {
                clearInterval(pollTimer);
                pollTimer = null;
            }
            showError(`查询任务状态失败: ${error.message}`);
        }
    };

    // 立即执行一次
    updateStatus();
    
    // 设置定时轮询
    pollTimer = setInterval(updateStatus, POLL_INTERVAL);
}

/**
 * 更新状态UI
 * @param {string} status - 任务状态
 * @param {object} data - 任务数据
 */
function updateStatusUI(status, data) {
    const statusTextEl = document.getElementById('status-text');
    const progressBarEl = document.getElementById('progress-bar');
    const progressFillEl = progressBarEl ? progressBarEl.querySelector('.progress-fill') : null;
    const taskStatusSection = document.getElementById('task-status-section');

    if (!statusTextEl || !taskStatusSection) return;

    taskStatusSection.classList.remove('hidden');

    // 显示平台进度信息
    let statusText = '';
    let statusClass = '';
    
    switch (status) {
        case 'pending':
            statusClass = 'status-pending';
            if (data && data.platform_progress) {
                const progress = data.platform_progress;
                statusText = `执行中... (已完成: ${progress.completed}/${progress.total})`;
                
                // 显示进度条并更新实际进度
                if (progressBarEl && progressFillEl) {
                    progressBarEl.classList.remove('hidden');
                    // 计算进度百分比
                    const progressPercent = progress.total > 0 
                        ? Math.round((progress.completed / progress.total) * 100) 
                        : 0;
                    // 移除动画类，使用实际宽度
                    progressFillEl.classList.remove('animated');
                    progressFillEl.style.width = `${progressPercent}%`;
                    progressFillEl.style.transition = 'width 0.3s ease-out';
                }
            } else {
                statusText = '执行中...';
                // 没有进度数据时，显示动画效果
                if (progressBarEl && progressFillEl) {
                    progressBarEl.classList.remove('hidden');
                    progressFillEl.classList.add('animated');
                    progressFillEl.style.width = '';
                }
            }
            break;
        case 'completed':
        case 'done':
            statusClass = 'status-done';
            if (data && data.platform_progress) {
                const progress = data.platform_progress;
                statusText = `已完成 (成功: ${progress.completed}, 失败: ${progress.failed}/${progress.total})`;
            } else {
                statusText = '已完成';
            }
            // 任务完成时，先设置进度条为 100%，然后隐藏
            if (progressBarEl && progressFillEl) {
                progressFillEl.classList.remove('animated');
                progressFillEl.style.width = '100%';
                progressFillEl.style.transition = 'width 0.3s ease-out';
                // 延迟隐藏，让用户看到完成状态
                setTimeout(() => {
                    progressBarEl.classList.add('hidden');
                }, 500);
            } else {
                progressBarEl.classList.add('hidden');
            }
            break;
        case 'none':
            statusClass = 'status-error';
            statusText = '任务不存在';
            progressBarEl.classList.add('hidden');
            break;
        default:
            statusText = '未知状态';
            progressBarEl.classList.add('hidden');
    }
    
    statusTextEl.textContent = statusText;
    statusTextEl.className = `status-text ${statusClass}`;
    
    // 如果任务执行中，显示各平台状态
    if (status === 'pending' && data && data.results_by_platform) {
        updatePlatformStatuses(data.results_by_platform, data.platforms);
    }
}

/**
 * 更新各平台状态显示
 * @param {object} resultsByPlatform - 按平台分组的结果
 * @param {string[]} platforms - 平台列表
 */
function updatePlatformStatuses(resultsByPlatform, platforms) {
    // 查找或创建平台状态容器
    let platformStatusContainer = document.getElementById('platform-status-container');
    if (!platformStatusContainer) {
        const taskStatusSection = document.getElementById('task-status-section');
        if (taskStatusSection) {
            platformStatusContainer = document.createElement('div');
            platformStatusContainer.id = 'platform-status-container';
            platformStatusContainer.className = 'platform-status-container';
            taskStatusSection.appendChild(platformStatusContainer);
        }
    }
    
    if (!platformStatusContainer || !platforms) return;
    
    platformStatusContainer.innerHTML = '';
    
    const statusList = document.createElement('div');
    statusList.className = 'platform-status-list';
    
    platforms.forEach(platform => {
        const platformLower = platform.toLowerCase();
        const platformData = resultsByPlatform[platformLower];
        const status = platformData ? platformData.status : 'pending';
        
        const statusItem = document.createElement('div');
        statusItem.className = 'platform-status-item';
        
        const icon = getPlatformStatusIcon(status);
        const name = getPlatformDisplayName(platform);
        
        statusItem.innerHTML = `
            ${icon}
            <span class="platform-name">${name}</span>
            <span class="platform-status-text">${getStatusText(status, platformData)}</span>
        `;
        
        statusList.appendChild(statusItem);
    });
    
    platformStatusContainer.appendChild(statusList);
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
 * 获取状态文本
 * @param {string} status - 状态
 * @param {object} platformData - 平台数据
 * @returns {string} 状态文本
 */
function getStatusText(status, platformData) {
    if (status === 'completed') {
        return '已完成';
    } else if (status === 'failed') {
        return '失败';
    } else {
        return '执行中...';
    }
}

/**
 * 停止当前任务轮询
 */
export function stopPolling() {
    if (pollTimer) {
        clearInterval(pollTimer);
        pollTimer = null;
    }
    currentTaskId = null;
}

