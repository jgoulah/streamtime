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
  const [uploading, setUploading] = useState(false);
  const [uploadResult, setUploadResult] = useState(null);

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

  const handleFileUpload = async (event) => {
    const file = event.target.files?.[0];
    if (!file) return;

    try {
      setUploading(true);
      setUploadResult(null);
      const result = await api.uploadNetflixCSV(file);
      setUploadResult(result);

      // Refresh the history after successful upload
      if (result.success && result.imported > 0) {
        setTimeout(() => {
          fetchServiceHistory();
        }, 1000);
      }
    } catch (err) {
      setUploadResult({
        success: false,
        error: err.message,
      });
    } finally {
      setUploading(false);
      // Reset file input
      event.target.value = '';
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
          <div className="flex gap-4">
            <button
              onClick={() => handleTriggerScrape(serviceName)}
              disabled={scraping}
              className="px-4 py-2 bg-blue-600 text-white rounded hover:bg-blue-700 transition-colors disabled:bg-gray-600 disabled:cursor-not-allowed"
            >
              {scraping ? 'Scraping...' : 'Trigger Scrape'}
            </button>
            {serviceName === 'Netflix' && (
              <label className="px-4 py-2 bg-green-600 text-white rounded hover:bg-green-700 transition-colors cursor-pointer disabled:bg-gray-600">
                {uploading ? 'Uploading...' : 'Upload CSV'}
                <input
                  type="file"
                  accept=".csv"
                  onChange={handleFileUpload}
                  disabled={uploading}
                  className="hidden"
                />
              </label>
            )}
          </div>
        </div>

        {/* Upload Result Message */}
        {uploadResult && (
          <div className={`mb-4 p-4 rounded-lg ${uploadResult.success ? 'bg-green-900 border border-green-700' : 'bg-red-900 border border-red-700'}`}>
            {uploadResult.success ? (
              <div className="text-white">
                <p className="font-semibold mb-2">✓ CSV Import Successful</p>
                <p className="text-sm">
                  Total rows: {uploadResult.total_rows} |
                  Imported: {uploadResult.imported} |
                  Skipped: {uploadResult.skipped} |
                  Errors: {uploadResult.errors}
                </p>
                {uploadResult.error_details && uploadResult.error_details.length > 0 && (
                  <details className="mt-2 text-sm">
                    <summary className="cursor-pointer">Show error details</summary>
                    <ul className="mt-2 ml-4 list-disc">
                      {uploadResult.error_details.map((err, idx) => (
                        <li key={idx}>{err}</li>
                      ))}
                    </ul>
                  </details>
                )}
              </div>
            ) : (
              <p className="text-white">✗ Upload failed: {uploadResult.error}</p>
            )}
          </div>
        )}

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
