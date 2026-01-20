/**
 * main.js - 应用入口模块
 * 初始化应用并绑定事件监听器
 */

import { startTask, stopPolling, getLastTaskId, pollTaskStatus } from './task.js';
import { getKeywords, getSelectedPlatforms, getQueryCount, showError } from './ui.js';
import { loadTaskByIds, downloadTaskData } from './ui.js';

/**
 * 初始化应用
 */
function init() {
    const submitBtn = document.getElementById('submit-btn');
    const keywordInput = document.getElementById('keyword-input');
    const loadTaskBtn = document.getElementById('load-task-btn');
    const downloadBtn = document.getElementById('download-btn');

    // 绑定提交按钮事件
    if (submitBtn) {
        submitBtn.addEventListener('click', handleSubmit);
    }

    // 绑定加载任务按钮事件
    if (loadTaskBtn) {
        loadTaskBtn.addEventListener('click', handleLoadTasks);
    }

    // 绑定下载按钮事件
    if (downloadBtn) {
        downloadBtn.addEventListener('click', handleDownload);
    }

    // 绑定回车键提交（Ctrl+Enter 或 Cmd+Enter）
    if (keywordInput) {
        keywordInput.addEventListener('keydown', (e) => {
            if ((e.ctrlKey || e.metaKey) && e.key === 'Enter') {
                handleSubmit();
            }
        });
    }

    // 页面加载时恢复最近一次的 task_id
    restoreLastTaskId();
}

/**
 * 恢复最近一次的 task_id
 */
async function restoreLastTaskId() {
    const lastTaskId = getLastTaskId();
    if (!lastTaskId) return;

    const taskIdInput = document.getElementById('task-id-input');
    if (!taskIdInput) return;

    // task_id 区域已经常显示，无需操作

    // 设置 task_id 值（只回显，不自动查询）
    taskIdInput.value = lastTaskId;
    
    // 不再自动加载任务数据，需要用户点击查询按钮
}

/**
 * 处理表单提交
 */
async function handleSubmit() {
    const keywords = getKeywords();
    const platforms = getSelectedPlatforms();
    const queryCount = getQueryCount();

    // 验证输入
    if (!keywords || keywords.length === 0) {
        showError('请输入至少一个查询条件');
        return;
    }

    if (platforms.length === 0) {
        showError('请至少选择一个平台');
        return;
    }

    if (queryCount < 1) {
        showError('查询次数必须大于0');
        return;
    }

    // 禁用提交按钮
    const submitBtn = document.getElementById('submit-btn');
    if (submitBtn) {
        submitBtn.disabled = true;
        submitBtn.textContent = '创建中...';
    }

    try {
        // 停止之前的轮询
        stopPolling();

        // 隐藏之前的结果
        const resultsSection = document.getElementById('results-section');
        if (resultsSection) {
            resultsSection.classList.add('hidden');
        }

        // 启动新任务
        await startTask(keywords, platforms, queryCount);
    } catch (error) {
        showError(`启动任务失败: ${error.message}`);
    } finally {
        // 恢复提交按钮
        if (submitBtn) {
            submitBtn.disabled = false;
            submitBtn.textContent = '创建查询任务';
        }
    }
}

/**
 * 处理加载任务
 */
async function handleLoadTasks() {
    const taskIdInput = document.getElementById('task-id-input');
    if (!taskIdInput) return;
    
    const taskIdsStr = taskIdInput.value.trim();
    if (!taskIdsStr) {
        showError('请输入 Task ID');
        return;
    }
    
    const taskIds = taskIdsStr.split(',').map(id => id.trim()).filter(id => id);
    if (taskIds.length === 0) {
        showError('Task ID 格式错误');
        return;
    }
    
    try {
        await loadTaskByIds(taskIds);
    } catch (error) {
        showError(`加载任务失败: ${error.message}`);
    }
}

/**
 * 处理下载数据
 */
async function handleDownload() {
    const taskIdInput = document.getElementById('task-id-input');
    if (!taskIdInput) return;
    
    const taskIdsStr = taskIdInput.value.trim();
    if (!taskIdsStr) {
        showError('请输入 Task ID');
        return;
    }
    
    const taskIds = taskIdsStr.split(',').map(id => id.trim()).filter(id => id);
    if (taskIds.length === 0) {
        showError('Task ID 格式错误');
        return;
    }
    
    try {
        await downloadTaskData(taskIds);
    } catch (error) {
        showError(`下载数据失败: ${error.message}`);
    }
}

// 页面加载完成后初始化
if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', init);
} else {
    init();
}


