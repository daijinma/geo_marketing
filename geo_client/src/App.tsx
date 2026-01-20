import { useEffect } from 'react';
import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom';
import Layout from './components/Layout';
import Login from './pages/Login';
import Dashboard from './pages/Dashboard';
import Search from './pages/Search';
import Tasks from './pages/Tasks';
import Logs from './pages/Logs';
import { useAuthStore } from './stores/authStore';

function App() {
  const { token, checkAndRefreshToken } = useAuthStore();

  // 应用启动时检查token是否有效
  useEffect(() => {
    const checkToken = async () => {
      if (token) {
        const isValid = await checkAndRefreshToken();
        if (!isValid) {
          // Token无效，但保留在路由中，由Login页面处理
        }
      }
    };
    checkToken();
  }, []);

  return (
    <BrowserRouter>
      <Routes>
        <Route path="/login" element={<Login />} />
        <Route
          path="/"
          element={token ? <Layout /> : <Navigate to="/login" replace />}
        >
          <Route index element={<Dashboard />} />
          <Route path="search" element={<Search />} />
          <Route path="tasks" element={<Tasks />} />
          <Route path="logs" element={<Logs />} />
        </Route>
      </Routes>
    </BrowserRouter>
  );
}

export default App;
