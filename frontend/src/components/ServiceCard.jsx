import { Link } from 'react-router-dom';
import { formatMinutes } from '../utils/format';

const ServiceCard = ({ service }) => {
  const { service_id, service_name, color, total_minutes, total_shows, last_watched } = service;

  return (
    <Link
      to={`/service/${service_id}`}
      className="block p-6 rounded-lg shadow-lg hover:shadow-xl transition-shadow duration-200 border-l-4"
      style={{
        backgroundColor: '#1e293b',
        borderLeftColor: color,
      }}
    >
      <div className="flex items-center justify-between mb-4">
        <h2 className="text-2xl font-bold text-white">{service_name}</h2>
        <div
          className="w-3 h-3 rounded-full"
          style={{ backgroundColor: color }}
        />
      </div>

      <div className="space-y-2">
        <div className="flex justify-between items-center">
          <span className="text-gray-400">Total Watch Time:</span>
          <span className="text-xl font-semibold text-white">
            {formatMinutes(total_minutes)}
          </span>
        </div>

        <div className="flex justify-between items-center">
          <span className="text-gray-400">Shows/Movies:</span>
          <span className="text-lg font-medium text-white">{total_shows}</span>
        </div>

        {last_watched && (
          <div className="flex justify-between items-center text-sm">
            <span className="text-gray-400">Last Watched:</span>
            <span className="text-gray-300">
              {new Date(last_watched).toLocaleDateString()}
            </span>
          </div>
        )}
      </div>

      <div className="mt-4 text-sm text-gray-400 hover:text-gray-300">
        View details →
      </div>
    </Link>
  );
};

export default ServiceCard;
