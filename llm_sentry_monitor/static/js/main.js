/**
 * main.js - 应用入口模块
 * 初始化应用并绑定事件监听器
 */

import { startTask, stopPolling } from './task.js';
import { getKeyword, getSelectedPlatforms, showError } from './ui.js';

/**
 * 初始化应用
 */
function init() {
    const submitBtn = document.getElementById('submit-btn');
    const keywordInput = document.getElementById('keyword-input');

    // 绑定提交按钮事件
    if (submitBtn) {
        submitBtn.addEventListener('click', handleSubmit);
    }

    // 绑定回车键提交
    if (keywordInput) {
        keywordInput.addEventListener('keypress', (e) => {
            if (e.key === 'Enter') {
                handleSubmit();
            }
        });
    }
}

/**
 * 处理表单提交
 */
async function handleSubmit() {
    const keyword = getKeyword();
    const platforms = getSelectedPlatforms();

    // 验证输入
    if (!keyword) {
        showError('请输入关键词');
        return;
    }

    if (platforms.length === 0) {
        showError('请至少选择一个平台');
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
        await startTask([keyword], platforms);
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

// 页面加载完成后初始化
if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', init);
} else {
    init();
}

