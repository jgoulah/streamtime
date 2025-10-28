import { useState, useEffect } from 'react';
import { useParams, Link } from 'react-router-dom';
import { LineChart, Line, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer } from 'recharts';
import api from '../services/api';
import { formatMinutes, formatDate, formatDateTime, getServiceSlug } from '../utils/format';

const ServiceDetail = () => {
  const { id } = useParams();
  const [data, setData] = useState(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);
  const [scraping, setScraping] = useState(false);

  useEffect(() => {
    fetchServiceHistory();
  }, [id]);

  const fetchServiceHistory = async () => {
    try {
      setLoading(true);
      const historyData = await api.getServiceHistory(id);
      setData(historyData);
      setError(null);
    } catch (err) {
      setError('Failed to load service history');
      console.error('Error fetching service history:', err);
    } finally {
      setLoading(false);
    }
  };

  const handleTriggerScrape = async (serviceName) => {
    try {
      setScraping(true);
      const slug = getServiceSlug(serviceName);
      await api.triggerScrape(slug);
      alert(`Scraper triggered for ${serviceName}. Check back in a few minutes.`);
    } catch (err) {
      alert(`Failed to trigger scraper: ${err.message}`);
    } finally {
      setScraping(false);
    }
  };

  if (loading) {
    return (
      <div className="min-h-screen bg-slate-900 flex items-center justify-center">
        <div className="text-center">
          <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-500 mx-auto"></div>
          <p className="mt-4 text-gray-400">Loading history...</p>
        </div>
      </div>
    );
  }

  if (error || !data) {
    return (
      <div className="min-h-screen bg-slate-900 flex items-center justify-center">
        <div className="text-center">
          <p className="text-red-400 mb-4">{error || 'No data found'}</p>
          <Link
            to="/"
            className="px-4 py-2 bg-blue-600 text-white rounded hover:bg-blue-700 transition-colors"
          >
            Back to Dashboard
          </Link>
        </div>
      </div>
    );
  }

  const { history, daily_stats, start_date, end_date } = data;

  // Prepare chart data
  const chartData = Object.entries(daily_stats || {}).map(([date, minutes]) => ({
    date: new Date(date).toLocaleDateString('en-US', { month: 'short', day: 'numeric' }),
    minutes,
  }));

  // Get service name from first history item
  const serviceName = history && history.length > 0 ? history[0].service_name || 'Service' : 'Service';

  return (
    <div className="min-h-screen bg-gradient-to-br from-slate-900 via-slate-800 to-slate-900 py-8 px-4 sm:px-6 lg:px-8">
      <div className="max-w-7xl mx-auto">
        {/* Header */}
        <div className="mb-8">
          <Link to="/" className="inline-flex items-center gap-2 text-blue-400 hover:text-blue-300 mb-4 transition-colors">
            <span className="text-lg">←</span> Back to Dashboard
          </Link>
          <div className="flex items-center justify-between">
            <div>
              <h1 className="text-5xl font-bold bg-gradient-to-r from-white to-slate-300 bg-clip-text text-transparent mb-2">
                {serviceName}
              </h1>
              <p className="text-slate-400 text-lg">
                📅 {formatDate(start_date)} - {formatDate(end_date)}
              </p>
            </div>
            <button
              onClick={() => handleTriggerScrape(serviceName)}
              disabled={scraping}
              className="px-6 py-3 bg-gradient-to-r from-blue-600 to-blue-500 text-white rounded-lg font-medium hover:shadow-lg hover:shadow-blue-500/50 transition-all disabled:from-slate-600 disabled:to-slate-600 disabled:cursor-not-allowed disabled:shadow-none"
            >
              {scraping ? '⏳ Scraping...' : '🔄 Trigger Scrape'}
            </button>
          </div>
        </div>

        {/* Chart */}
        {chartData.length > 0 && (
          <div className="bg-gradient-to-br from-slate-800 to-slate-900 p-6 rounded-lg border border-slate-700 shadow-xl mb-8">
            <h2 className="text-2xl font-bold text-white mb-6">📊 Daily Watch Time</h2>
            <ResponsiveContainer width="100%" height={300}>
              <LineChart data={chartData}>
                <CartesianGrid strokeDasharray="3 3" stroke="#475569" />
                <XAxis dataKey="date" stroke="#94a3b8" />
                <YAxis stroke="#94a3b8" />
                <Tooltip
                  contentStyle={{ backgroundColor: '#1e293b', border: '1px solid #475569' }}
                  labelStyle={{ color: '#e2e8f0' }}
                  formatter={(value) => [formatMinutes(value), 'Watch Time']}
                />
                <Line type="monotone" dataKey="minutes" stroke="#3b82f6" strokeWidth={2} />
              </LineChart>
            </ResponsiveContainer>
          </div>
        )}

        {/* Watch History */}
        <div className="bg-gradient-to-br from-slate-800 to-slate-900 p-6 rounded-lg border border-slate-700 shadow-xl">
          <h2 className="text-2xl font-bold text-white mb-6">🎥 Watch History</h2>
          {!history || history.length === 0 ? (
            <div className="text-center py-12">
              <p className="text-slate-400">No watch history found for this period</p>
            </div>
          ) : (
            <div className="space-y-3">
              {history.map((item) => (
                <div
                  key={item.id}
                  className="bg-slate-700/50 border border-slate-600 p-5 rounded-lg hover:border-blue-500 hover:bg-slate-700 transition-all"
                >
                  <div className="flex justify-between items-start mb-2">
                    <div className="flex-1">
                      <h3 className="text-lg font-semibold text-white">
                        {item.title}
                      </h3>
                      {item.episode_info && (
                        <p className="text-slate-400 text-sm mt-1">
                          📺 {item.episode_info}
                        </p>
                      )}
                    </div>
                    <span className="text-blue-400 font-medium ml-4 flex items-center gap-1">
                      ⏱️ {formatMinutes(item.duration_minutes)}
                    </span>
                  </div>
                  <div className="flex justify-between items-center text-sm text-slate-400">
                    <span>🕐 {formatDateTime(item.watched_at)}</span>
                    {item.genre && <span className="text-slate-500 px-2 py-1 bg-slate-600/30 rounded">{item.genre}</span>}
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>
      </div>
    </div>
  );
};

export default ServiceDetail;
