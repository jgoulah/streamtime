import { useState, useEffect } from 'react';
import ServiceCard from '../components/ServiceCard';
import api from '../services/api';
import { formatMinutes } from '../utils/format';

const Dashboard = () => {
  const [services, setServices] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);
  const [filterType, setFilterType] = useState('all'); // 'all', 'year', 'month'
  const [selectedYear, setSelectedYear] = useState(new Date().getFullYear());
  const [selectedMonth, setSelectedMonth] = useState(new Date().getMonth() + 1);

  // Generate year options (from 2009 to current year)
  const currentYear = new Date().getFullYear();
  const years = Array.from({ length: currentYear - 2009 + 1 }, (_, i) => 2009 + i).reverse();

  const months = [
    { value: 1, label: 'January' },
    { value: 2, label: 'February' },
    { value: 3, label: 'March' },
    { value: 4, label: 'April' },
    { value: 5, label: 'May' },
    { value: 6, label: 'June' },
    { value: 7, label: 'July' },
    { value: 8, label: 'August' },
    { value: 9, label: 'September' },
    { value: 10, label: 'October' },
    { value: 11, label: 'November' },
    { value: 12, label: 'December' },
  ];

  useEffect(() => {
    fetchServices();
  }, [filterType, selectedYear, selectedMonth]);

  const fetchServices = async () => {
    try {
      setLoading(true);
      const params = {};

      if (filterType === 'year') {
        params.year = selectedYear;
      } else if (filterType === 'month') {
        params.year = selectedYear;
        params.month = selectedMonth;
      }

      const data = await api.getServices(params);
      setServices(data || []);
      setError(null);
    } catch (err) {
      setError('Failed to load services. Please try again.');
      console.error('Error fetching services:', err);
    } finally {
      setLoading(false);
    }
  };

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

  const getFilterLabel = () => {
    if (filterType === 'all') return 'All Time';
    if (filterType === 'year') return `Year: ${selectedYear}`;
    const monthLabel = months.find(m => m.value === selectedMonth)?.label;
    return `${monthLabel} ${selectedYear}`;
  };

  return (
    <div className="min-h-screen bg-slate-900 py-8 px-4 sm:px-6 lg:px-8">
      <div className="max-w-7xl mx-auto">
        {/* Header */}
        <div className="mb-8">
          <h1 className="text-5xl font-bold text-white mb-3">
            StreamTime
          </h1>
          <p className="text-slate-400 text-lg">Track your streaming service watch time across all platforms</p>
        </div>

        {/* Filters */}
        <div className="bg-slate-800 p-6 rounded-lg border border-slate-700 mb-8">
          <div className="flex flex-wrap items-center gap-4">
            <div className="flex items-center gap-2">
              <label className="text-slate-300 text-sm font-medium">View:</label>
              <div className="flex gap-2">
                <button
                  onClick={() => setFilterType('all')}
                  className={`px-4 py-2 rounded-md font-medium transition-colors ${
                    filterType === 'all'
                      ? 'bg-blue-600 text-white'
                      : 'bg-slate-700 text-slate-300 hover:bg-slate-600'
                  }`}
                >
                  All Time
                </button>
                <button
                  onClick={() => setFilterType('year')}
                  className={`px-4 py-2 rounded-md font-medium transition-colors ${
                    filterType === 'year'
                      ? 'bg-blue-600 text-white'
                      : 'bg-slate-700 text-slate-300 hover:bg-slate-600'
                  }`}
                >
                  Year
                </button>
                <button
                  onClick={() => setFilterType('month')}
                  className={`px-4 py-2 rounded-md font-medium transition-colors ${
                    filterType === 'month'
                      ? 'bg-blue-600 text-white'
                      : 'bg-slate-700 text-slate-300 hover:bg-slate-600'
                  }`}
                >
                  Month
                </button>
              </div>
            </div>

            {(filterType === 'year' || filterType === 'month') && (
              <div className="flex items-center gap-2">
                <label className="text-slate-300 text-sm font-medium">Year:</label>
                <select
                  value={selectedYear}
                  onChange={(e) => setSelectedYear(parseInt(e.target.value))}
                  className="bg-slate-700 text-white px-4 py-2 rounded-md border border-slate-600 focus:border-blue-500 focus:outline-none"
                >
                  {years.map((year) => (
                    <option key={year} value={year}>
                      {year}
                    </option>
                  ))}
                </select>
              </div>
            )}

            {filterType === 'month' && (
              <div className="flex items-center gap-2">
                <label className="text-slate-300 text-sm font-medium">Month:</label>
                <select
                  value={selectedMonth}
                  onChange={(e) => setSelectedMonth(parseInt(e.target.value))}
                  className="bg-slate-700 text-white px-4 py-2 rounded-md border border-slate-600 focus:border-blue-500 focus:outline-none"
                >
                  {months.map((month) => (
                    <option key={month.value} value={month.value}>
                      {month.label}
                    </option>
                  ))}
                </select>
              </div>
            )}

            <div className="ml-auto text-slate-400 text-sm">
              Showing: <span className="text-white font-medium">{getFilterLabel()}</span>
            </div>
          </div>
        </div>

        {/* Summary Stats */}
        <div className="grid grid-cols-1 md:grid-cols-3 gap-6 mb-8">
          <div className="bg-slate-800 p-6 rounded-lg border border-slate-700">
            <h3 className="text-slate-300 text-xs uppercase tracking-wider font-semibold mb-2">Total Watch Time</h3>
            <p className="text-4xl font-bold text-blue-400">
              {formatMinutes(totalMinutes)}
            </p>
          </div>

          <div className="bg-slate-800 p-6 rounded-lg border border-slate-700">
            <h3 className="text-slate-300 text-xs uppercase tracking-wider font-semibold mb-2">Total Shows/Movies</h3>
            <p className="text-4xl font-bold text-purple-400">
              {totalShows.toLocaleString()}
            </p>
          </div>

          <div className="bg-slate-800 p-6 rounded-lg border border-slate-700">
            <h3 className="text-slate-300 text-xs uppercase tracking-wider font-semibold mb-2">Active Services</h3>
            <p className="text-4xl font-bold text-pink-400">
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
