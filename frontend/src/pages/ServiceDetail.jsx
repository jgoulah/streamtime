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
    <div className="min-h-screen bg-slate-900 py-8 px-4 sm:px-6 lg:px-8">
      <div className="max-w-7xl mx-auto">
        {/* Header */}
        <div className="mb-8 flex items-center justify-between">
          <div>
            <Link to="/" className="text-blue-400 hover:text-blue-300 mb-2 inline-block">
              ← Back to Dashboard
            </Link>
            <h1 className="text-4xl font-bold text-white">{serviceName}</h1>
            <p className="text-gray-400">
              {formatDate(start_date)} - {formatDate(end_date)}
            </p>
          </div>
          <button
            onClick={() => handleTriggerScrape(serviceName)}
            disabled={scraping}
            className="px-4 py-2 bg-blue-600 text-white rounded hover:bg-blue-700 transition-colors disabled:bg-gray-600 disabled:cursor-not-allowed"
          >
            {scraping ? 'Scraping...' : 'Trigger Scrape'}
          </button>
        </div>

        {/* Chart */}
        {chartData.length > 0 && (
          <div className="bg-slate-800 p-6 rounded-lg shadow-lg mb-8">
            <h2 className="text-xl font-bold text-white mb-4">Daily Watch Time</h2>
            <ResponsiveContainer width="100%" height={300}>
              <LineChart data={chartData}>
                <CartesianGrid strokeDasharray="3 3" stroke="#374151" />
                <XAxis dataKey="date" stroke="#9ca3af" />
                <YAxis stroke="#9ca3af" />
                <Tooltip
                  contentStyle={{ backgroundColor: '#1e293b', border: '1px solid #374151' }}
                  labelStyle={{ color: '#e5e7eb' }}
                  formatter={(value) => [formatMinutes(value), 'Watch Time']}
                />
                <Line type="monotone" dataKey="minutes" stroke="#3b82f6" strokeWidth={2} />
              </LineChart>
            </ResponsiveContainer>
          </div>
        )}

        {/* Watch History */}
        <div className="bg-slate-800 p-6 rounded-lg shadow-lg">
          <h2 className="text-xl font-bold text-white mb-4">Watch History</h2>
          {!history || history.length === 0 ? (
            <p className="text-gray-400">No watch history found for this period</p>
          ) : (
            <div className="space-y-4">
              {history.map((item) => (
                <div
                  key={item.id}
                  className="bg-slate-700 p-4 rounded-lg hover:bg-slate-600 transition-colors"
                >
                  <div className="flex justify-between items-start mb-2">
                    <div>
                      <h3 className="text-lg font-semibold text-white">{item.title}</h3>
                      {item.episode_info && (
                        <p className="text-gray-400 text-sm">{item.episode_info}</p>
                      )}
                    </div>
                    <span className="text-blue-400 font-medium">
                      {formatMinutes(item.duration_minutes)}
                    </span>
                  </div>
                  <div className="flex justify-between items-center text-sm text-gray-400">
                    <span>{formatDateTime(item.watched_at)}</span>
                    {item.genre && <span className="text-gray-500">{item.genre}</span>}
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
