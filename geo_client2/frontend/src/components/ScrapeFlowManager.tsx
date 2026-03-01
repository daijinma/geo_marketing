import { useState, useEffect, useRef } from 'react';
import { wailsAPI, FlowBundle } from '@/utils/wails-api';
import { toast } from 'sonner';
import { Upload, Download, Trash2, CheckCircle, Circle, RefreshCw, FileCode } from 'lucide-react';
import { cn } from '@/lib/utils';
import { Button } from '@/components/ui/button';

export function ScrapeFlowManager() {
  const [bundles, setBundles] = useState<FlowBundle[]>([]);
  const [loading, setLoading] = useState(false);
  const [uploading, setUploading] = useState(false);
  const fileInputRef = useRef<HTMLInputElement>(null);

  useEffect(() => {
    loadBundles();
  }, []);

  const loadBundles = async () => {
    setLoading(true);
    try {
      const result = await wailsAPI.scrapeFlow.list();
      // Assume result.bundles is the list and result.activeVersion is the active one
      // If the API returns differently, adjust here. 
      // Based on my definition: GetFlowBundles(): Promise<{ success: boolean; bundles: FlowBundle[]; activeVersion: string }>;
      
      if (result && result.bundles) {
          // Sort by version desc (newest first)
          const sorted = [...result.bundles].sort((a, b) => b.version.localeCompare(a.version));
          
          // If activeVersion is returned separately, we might need to map it.
          // But my interface definition has active boolean in FlowBundle.
          // Let's assume the backend sets the active flag correctly in the list.
          // If the backend returns activeVersion string, we can manually set the flags.
          
          if (result.activeVersion) {
              sorted.forEach(b => b.active = b.version === result.activeVersion);
          }
          
          setBundles(sorted);
      }
    } catch (error) {
      console.error('Failed to load flow bundles:', error);
      toast.error('获取抓取脚本列表失败');
    } finally {
      setLoading(false);
    }
  };

  const handleFileUpload = async (event: React.ChangeEvent<HTMLInputElement>) => {
    const file = event.target.files?.[0];
    if (!file) return;

    if (!file.name.endsWith('.zip')) {
      toast.error('请上传 .zip 格式的文件');
      return;
    }

    setUploading(true);
    try {
      const reader = new FileReader();
      reader.onload = async (e) => {
        const base64 = (e.target?.result as string).split(',')[1];
        try {
          const res = await wailsAPI.scrapeFlow.import(base64);
          if (res.success) {
            toast.success(`脚本包 ${res.version} 导入成功`);
            loadBundles();
          } else {
            toast.error(res.error || '导入失败');
          }
        } catch (err) {
            console.error(err);
            toast.error('导入出错');
        }
      };
      reader.readAsDataURL(file);
    } catch (error) {
      console.error('File reading failed:', error);
      toast.error('文件读取失败');
    } finally {
      setUploading(false);
      if (fileInputRef.current) {
        fileInputRef.current.value = '';
      }
    }
  };

  const handleSwitch = async (version: string) => {
    try {
      const res = await wailsAPI.scrapeFlow.switch(version);
      if (res && res.success) { // wailsAPI.switch returns Promise<{success: boolean; error?: string}>
        toast.success(`已切换至版本 ${version}`);
        loadBundles();
      } else {
        toast.error(res?.error || '切换失败');
      }
    } catch (error) {
      console.error('Switch failed:', error);
      toast.error('切换版本失败');
    }
  };

  const handleDelete = async (version: string) => {
    if (!confirm(`确定要删除版本 ${version} 吗？`)) return;

    try {
      const res = await wailsAPI.scrapeFlow.delete(version);
      if (res && res.success) {
        toast.success('删除成功');
        loadBundles();
      } else {
        toast.error(res?.error || '删除失败');
      }
    } catch (error) {
      console.error('Delete failed:', error);
      toast.error('删除版本失败');
    }
  };

  const handleExport = async (version: string) => {
    try {
      const res = await wailsAPI.scrapeFlow.export(version);
      if (res && res.success && res.content) {
        // res.content is base64 zip
        const link = document.createElement('a');
        link.href = `data:application/zip;base64,${res.content}`;
        link.download = `flow_bundle_${version}.zip`;
        document.body.appendChild(link);
        link.click();
        document.body.removeChild(link);
        toast.success('导出成功');
      } else {
        toast.error(res?.error || '导出失败');
      }
    } catch (error) {
      console.error('Export failed:', error);
      toast.error('导出版本失败');
    }
  };

  const handleExportActive = async () => {
    try {
      const res = await wailsAPI.scrapeFlow.exportActive();
      if (res && res.success && res.content && res.version) {
        const link = document.createElement('a');
        link.href = `data:application/zip;base64,${res.content}`;
        link.download = `flow_bundle_${res.version}.zip`;
        document.body.appendChild(link);
        link.click();
        document.body.removeChild(link);
        toast.success('当前版本已下载');
      } else {
        toast.error(res?.error || '下载失败');
      }
    } catch (error) {
      console.error('Export active failed:', error);
      toast.error('下载失败');
    }
  };

  const formatSize = (bytes: number) => {
    if (bytes === 0) return '0 B';
    const k = 1024;
    const sizes = ['B', 'KB', 'MB', 'GB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
  };

  const formatVersion = (v: string) => {
      // Expecting format: YYYYMMDDHHmmss
      if (v.length === 14) {
          return `${v.slice(0, 4)}-${v.slice(4, 6)}-${v.slice(6, 8)} ${v.slice(8, 10)}:${v.slice(10, 12)}:${v.slice(12, 14)}`;
      }
      return v;
  }

  return (
    <div className="space-y-6">
       <div className="flex flex-col space-y-1.5">
        <h3 className="text-2xl font-semibold leading-none tracking-tight flex items-center gap-2">
            <FileCode className="w-6 h-6" />
            抓取脚本管理
        </h3>
        <p className="text-sm text-muted-foreground">
            上传、切换和管理不同版本的抓取流程脚本 (Zip包)。
        </p>
      </div>

      <div className="flex justify-between items-center">
        <div className="flex items-center gap-2">
            <input
                type="file"
                ref={fileInputRef}
                className="hidden"
                accept=".zip"
                onChange={handleFileUpload}
            />
            <Button 
                onClick={() => fileInputRef.current?.click()} 
                disabled={uploading}
                variant="outline"
                className="gap-2"
            >
                <Upload className="w-4 h-4" />
                {uploading ? '上传中...' : '上传新版本'}
            </Button>
            <Button
                onClick={handleExportActive}
                variant="outline"
                className="gap-2"
            >
                <Download className="w-4 h-4" />
                下载当前版本
            </Button>
        </div>
        <Button variant="ghost" size="sm" onClick={loadBundles} disabled={loading}>
            <RefreshCw className={cn("w-4 h-4", loading && "animate-spin")} />
        </Button>
      </div>

      <div className="rounded-md border">
        <div className="relative w-full overflow-auto">
            <table className="w-full caption-bottom text-sm">
                <thead className="[&_tr]:border-b">
                    <tr className="border-b transition-colors hover:bg-muted/50 data-[state=selected]:bg-muted">
                        <th className="h-12 px-4 text-left align-middle font-medium text-muted-foreground w-[50px]">状态</th>
                         <th className="h-12 px-4 text-left align-middle font-medium text-muted-foreground">版本 (时间戳)</th>
                        <th className="h-12 px-4 text-left align-middle font-medium text-muted-foreground">大小</th>
                        <th className="h-12 px-4 text-right align-middle font-medium text-muted-foreground">操作</th>
                    </tr>
                </thead>
                <tbody className="[&_tr:last-child]:border-0">
                    {bundles.length === 0 ? (
                        <tr>
                            <td colSpan={4} className="p-4 text-center text-muted-foreground">
                                暂无版本记录
                            </td>
                        </tr>
                    ) : (
                        bundles.map((bundle) => (
                            <tr key={bundle.version} className={cn("border-b transition-colors hover:bg-muted/50", bundle.active && "bg-muted/30")}>
                                <td className="p-4 align-middle">
                                    {bundle.active ? (
                                        <CheckCircle className="w-5 h-5 text-green-500" />
                                    ) : (
                                        <Circle className="w-5 h-5 text-muted-foreground/30" />
                                    )}
                                </td>
                                <td className="p-4 align-middle font-medium">
                                    <div className="flex flex-col">
                                        <span>{bundle.version}</span>
                                        <span className="text-xs text-muted-foreground font-normal">{formatVersion(bundle.version)}</span>
                                    </div>
                                </td>
                                <td className="p-4 align-middle text-muted-foreground">
                                    {formatSize(bundle.size)}
                                </td>
                                <td className="p-4 align-middle text-right">
                                    <div className="flex justify-end items-center gap-2">
                                        {!bundle.active && (
                                            <Button 
                                                variant="ghost" 
                                                size="sm" 
                                                onClick={() => handleSwitch(bundle.version)}
                                                className="h-8 px-2 text-blue-600 hover:text-blue-700 hover:bg-blue-50"
                                            >
                                                启用
                                            </Button>
                                        )}
                                        <Button 
                                            variant="ghost" 
                                            size="icon" 
                                            onClick={() => handleExport(bundle.version)}
                                            className="h-8 w-8"
                                            title="导出"
                                        >
                                            <Download className="w-4 h-4" />
                                        </Button>
                                        <Button 
                                            variant="ghost" 
                                            size="icon" 
                                            onClick={() => handleDelete(bundle.version)}
                                            disabled={bundle.active}
                                            className={cn("h-8 w-8", bundle.active ? "opacity-30 cursor-not-allowed" : "text-red-500 hover:text-red-600 hover:bg-red-50")}
                                            title={bundle.active ? "当前版本不可删除" : "删除"}
                                        >
                                            <Trash2 className="w-4 h-4" />
                                        </Button>
                                    </div>
                                </td>
                            </tr>
                        ))
                    )}
                </tbody>
            </table>
        </div>
      </div>
    </div>
  );
}
