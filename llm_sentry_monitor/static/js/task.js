/**
 * task.js - 任务管理逻辑模块
 * 处理任务创建、状态轮询等逻辑
 */

import { createTask, getTaskStatus } from './api.js';
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
            
            const statusData = await getTaskStatus(taskId);
            const status = statusData.status;
            const data = statusData.data;

            // 更新状态显示
            updateStatusUI(status, data);

            if (status === 'done') {
                // 任务完成
                clearInterval(pollTimer);
                pollTimer = null;
                
                if (data && data.query_tokens) {
                    renderResults(data);
                    showSuccess('任务执行完成！');
                } else {
                    showError('任务完成但未获取到结果数据');
                }
            } else if (status === 'none') {
                // 任务不存在
                clearInterval(pollTimer);
                pollTimer = null;
                showError('任务不存在');
            } else if (pollCount >= MAX_POLL_COUNT) {
                // 超时
                clearInterval(pollTimer);
                pollTimer = null;
                showError('任务执行超时，请稍后手动查询');
            }
            // status === 'pending' 时继续轮询
        } catch (error) {
            clearInterval(pollTimer);
            pollTimer = null;
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
    const taskStatusSection = document.getElementById('task-status-section');

    if (!statusTextEl || !taskStatusSection) return;

    taskStatusSection.classList.remove('hidden');

    switch (status) {
        case 'pending':
            statusTextEl.textContent = '执行中...';
            statusTextEl.className = 'status-text status-pending';
            progressBarEl.classList.remove('hidden');
            break;
        case 'done':
            statusTextEl.textContent = '已完成';
            statusTextEl.className = 'status-text status-done';
            progressBarEl.classList.add('hidden');
            break;
        case 'none':
            statusTextEl.textContent = '任务不存在';
            statusTextEl.className = 'status-text status-error';
            progressBarEl.classList.add('hidden');
            break;
        default:
            statusTextEl.textContent = '未知状态';
            statusTextEl.className = 'status-text';
            progressBarEl.classList.add('hidden');
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

