/**
 * api.js - API调用封装模块
 * 统一处理所有API请求和错误处理
 */

const API_BASE_URL = '';

/**
 * 创建查询任务
 * @param {string[]} keywords - 关键词列表
 * @param {string[]} platforms - 平台列表
 * @returns {Promise<{task_id: number}>}
 */
export async function createTask(keywords, platforms) {
    try {
        const response = await fetch(`${API_BASE_URL}/mock`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
            },
            body: JSON.stringify({
                keywords: keywords,
                platforms: platforms,
                settings: {
                    headless: false,
                    timeout: 60000,
                    delay_between_tasks: 5
                }
            })
        });

        if (!response.ok) {
            const errorData = await response.json().catch(() => ({ detail: '请求失败' }));
            throw new Error(errorData.detail || `HTTP ${response.status}`);
        }

        return await response.json();
    } catch (error) {
        if (error instanceof TypeError && error.message.includes('fetch')) {
            throw new Error('网络连接失败，请检查服务器是否运行');
        }
        throw error;
    }
}

/**
 * 查询任务状态
 * @param {number} taskId - 任务ID
 * @returns {Promise<{status: string, data: object}>}
 */
export async function getTaskStatus(taskId) {
    try {
        const response = await fetch(`${API_BASE_URL}/status?id=${taskId}`);

        if (!response.ok) {
            const errorData = await response.json().catch(() => ({ detail: '请求失败' }));
            throw new Error(errorData.detail || `HTTP ${response.status}`);
        }

        return await response.json();
    } catch (error) {
        if (error instanceof TypeError && error.message.includes('fetch')) {
            throw new Error('网络连接失败，请检查服务器是否运行');
        }
        throw error;
    }
}

/**
 * 使用博查API搜索词条
 * @param {string} token - 搜索词条
 * @returns {Promise<{success: boolean, data: object, error?: string}>}
 */
export async function searchBocha(token) {
    try {
        const response = await fetch(`${API_BASE_URL}/bocha/search?token=${encodeURIComponent(token)}`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
            }
        });

        if (!response.ok) {
            const errorData = await response.json().catch(() => ({ error: '请求失败' }));
            throw new Error(errorData.error || `HTTP ${response.status}`);
        }

        return await response.json();
    } catch (error) {
        if (error instanceof TypeError && error.message.includes('fetch')) {
            throw new Error('网络连接失败，请检查服务器是否运行');
        }
        throw error;
    }
}

