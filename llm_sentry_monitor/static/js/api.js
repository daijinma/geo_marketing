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
 * 使用博查API搜索查询词
 * @param {string} query - 查询词
 * @returns {Promise<{success: boolean, data: object, error?: string}>}
 */
export async function searchBocha(query) {
    try {
        const response = await fetch(`${API_BASE_URL}/bocha/search?query=${encodeURIComponent(query)}`, {
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

/**
 * 验证API响应格式
 * @param {object} responseData - API响应数据
 * @returns {{valid: boolean, errors: string[]}}
 */
export function validateApiResponse(responseData) {
    const errors = [];
    
    // 验证顶层结构
    if (!responseData) {
        errors.push('响应数据为空');
        return { valid: false, errors };
    }
    
    if (typeof responseData.status !== 'string') {
        errors.push('缺少必需字段: status (应为字符串)');
    }
    
    if (!responseData.data || typeof responseData.data !== 'object') {
        errors.push('缺少必需字段: data (应为对象)');
        return { valid: false, errors };
    }
    
    const data = responseData.data;
    
    // 验证 data 中的必需字段
    if (typeof data.task_id !== 'number') {
        errors.push('data 缺少必需字段: task_id (应为数字)');
    }
    
    if (!Array.isArray(data.keywords)) {
        errors.push('data 缺少必需字段: keywords (应为数组)');
    }
    
    if (!Array.isArray(data.platforms)) {
        errors.push('data 缺少必需字段: platforms (应为数组)');
    }
    
    // 如果任务完成，验证结果数据
    if (responseData.status === 'done') {
        if (!data.results_by_platform || typeof data.results_by_platform !== 'object') {
            errors.push('任务完成但缺少 results_by_platform 字段');
        } else {
            // 验证每个平台的数据结构
            for (const [platform, platformData] of Object.entries(data.results_by_platform)) {
                if (!platformData || typeof platformData !== 'object') {
                    errors.push(`平台 ${platform} 的数据格式错误`);
                    continue;
                }
                
                if (!Array.isArray(platformData.query_tokens)) {
                    errors.push(`平台 ${platform} 缺少 query_tokens 数组`);
                } else {
                    // 验证每个 query_tokens 项
                    platformData.query_tokens.forEach((item, index) => {
                        if (!item || typeof item !== 'object') {
                            errors.push(`平台 ${platform} 的 query_tokens[${index}] 格式错误`);
                            return;
                        }
                        
                        if (typeof item.query !== 'string') {
                            errors.push(`平台 ${platform} 的 query_tokens[${index}] 缺少 query 字段`);
                        }
                        
                        if (!Array.isArray(item.tokens)) {
                            errors.push(`平台 ${platform} 的 query_tokens[${index}] 缺少 tokens 数组`);
                        }
                        
                        if (!Array.isArray(item.citations)) {
                            errors.push(`平台 ${platform} 的 query_tokens[${index}] 缺少 citations 数组`);
                        }
                    });
                }
                
                if (typeof platformData.status !== 'string') {
                    errors.push(`平台 ${platform} 缺少 status 字段`);
                }
            }
            
            // 验证进度信息
            if (!data.platform_progress || typeof data.platform_progress !== 'object') {
                errors.push('缺少 platform_progress 字段');
            } else {
                const progress = data.platform_progress;
                if (typeof progress.completed !== 'number' ||
                    typeof progress.failed !== 'number' ||
                    typeof progress.pending !== 'number' ||
                    typeof progress.total !== 'number') {
                    errors.push('platform_progress 字段格式错误');
                }
            }
        }
    }
    
    return {
        valid: errors.length === 0,
        errors: errors
    };
}

