import { BrowserRouter as Router, Routes, Route } from 'react-router-dom';
import Dashboard from './pages/Dashboard';
import ServiceDetail from './pages/ServiceDetail';

function App() {
  return (
    <Router>
      <Routes>
        <Route path="/" element={<Dashboard />} />
        <Route path="/service/:id" element={<ServiceDetail />} />
      </Routes>
    </Router>
  );
}

export default App;
