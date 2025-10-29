import { useState, useEffect, useCallback } from 'react';
import ServiceCard from '../components/ServiceCard';
import DateFilter from '../components/DateFilter';
import api from '../services/api';
import { formatMinutes } from '../utils/format';

const Dashboard = () => {
  const [services, setServices] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);
  const [filterParams, setFilterParams] = useState({});
  const [isInitialLoad, setIsInitialLoad] = useState(true);

  const handleFilterChange = useCallback(({ params }) => {
    setFilterParams(params);
  }, []);

  const fetchServices = useCallback(async () => {
    try {
      if (isInitialLoad) {
        setLoading(true);
      }
      const data = await api.getServices(filterParams);
      setServices(data || []);
      setError(null);
    } catch (err) {
      setError('Failed to load services. Please try again.');
      console.error('Error fetching services:', err);
    } finally {
      setLoading(false);
      setIsInitialLoad(false);
    }
  }, [filterParams, isInitialLoad]);

  useEffect(() => {
    fetchServices();
  }, [fetchServices]);

  const totalMinutes = services.reduce((sum, service) => sum + service.total_minutes, 0);
  const totalShows = services.reduce((sum, service) => sum + service.total_shows, 0);

  if (loading) {
    return (
      <div className="min-h-screen bg-slate-900 flex items-center justify-center">
        <div className="text-center">
          <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-500 mx-auto"></div>
          <p className="mt-4 text-gray-400">Loading services...</p>
        </div>
      </div>
    );
  }

  if (error) {
    return (
      <div className="min-h-screen bg-slate-900 flex items-center justify-center">
        <div className="text-center">
          <p className="text-red-400 mb-4">{error}</p>
          <button
            onClick={fetchServices}
            className="px-4 py-2 bg-blue-600 text-white rounded hover:bg-blue-700 transition-colors"
          >
            Retry
          </button>
        </div>
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-gradient-to-br from-slate-900 via-slate-800 to-slate-900 py-8 px-4 sm:px-6 lg:px-8">
      <div className="max-w-7xl mx-auto">
        {/* Header */}
        <div className="mb-8">
          <h1 className="text-5xl font-bold text-white mb-3 bg-gradient-to-r from-white to-slate-300 bg-clip-text text-transparent">
            StreamTime
          </h1>
          <p className="text-slate-400 text-lg">Track your streaming service watch time across all platforms</p>
        </div>

        {/* Filters */}
        <div className="mb-8">
          <DateFilter onFilterChange={handleFilterChange} />
        </div>

        {/* Summary Stats */}
        <div className="grid grid-cols-1 md:grid-cols-3 gap-6 mb-8">
          <div className="bg-gradient-to-br from-slate-800 to-slate-900 p-6 rounded-lg border border-slate-700 shadow-lg hover:shadow-blue-500/10 transition-shadow">
            <h3 className="text-slate-300 text-xs uppercase tracking-wider font-semibold mb-2">📺 Total Watch Time</h3>
            <p className="text-4xl font-bold bg-gradient-to-r from-blue-400 to-cyan-400 bg-clip-text text-transparent">
              {formatMinutes(totalMinutes)}
            </p>
          </div>

          <div className="bg-gradient-to-br from-slate-800 to-slate-900 p-6 rounded-lg border border-slate-700 shadow-lg hover:shadow-purple-500/10 transition-shadow">
            <h3 className="text-slate-300 text-xs uppercase tracking-wider font-semibold mb-2">🎬 Total Shows/Movies</h3>
            <p className="text-4xl font-bold bg-gradient-to-r from-purple-400 to-pink-400 bg-clip-text text-transparent">
              {totalShows.toLocaleString()}
            </p>
          </div>

          <div className="bg-gradient-to-br from-slate-800 to-slate-900 p-6 rounded-lg border border-slate-700 shadow-lg hover:shadow-pink-500/10 transition-shadow">
            <h3 className="text-slate-300 text-xs uppercase tracking-wider font-semibold mb-2">⭐ Active Services</h3>
            <p className="text-4xl font-bold bg-gradient-to-r from-pink-400 to-rose-400 bg-clip-text text-transparent">
              {services.length}
            </p>
          </div>
        </div>

        {/* Service Cards */}
        {services.length === 0 ? (
          <div className="text-center py-12 bg-slate-800 rounded-lg">
            <p className="text-gray-400 text-lg">No services found</p>
            <p className="text-gray-500 mt-2">Enable services in your configuration to get started</p>
          </div>
        ) : (
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
            {services.map((service) => (
              <ServiceCard key={service.service_id} service={service} />
            ))}
          </div>
        )}
      </div>
    </div>
  );
};

export default Dashboard;
