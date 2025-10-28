import { Link } from 'react-router-dom';
import { formatMinutes } from '../utils/format';

const ServiceCard = ({ service }) => {
  const { service_id, service_name, color, total_minutes, total_shows, last_watched } = service;

  return (
    <Link
      to={`/service/${service_id}`}
      className="block bg-slate-800 rounded-lg border-2 border-slate-700 hover:border-blue-500 transition-colors p-6"
    >
      {/* Color accent bar */}
      <div
        className="h-1 w-16 rounded-full mb-4"
        style={{ backgroundColor: color }}
      />

      <h2 className="text-2xl font-bold text-white mb-6">
        {service_name}
      </h2>

      <div className="space-y-4">
        <div>
          <div className="text-xs uppercase tracking-wider text-slate-400 mb-1">Total Watch Time</div>
          <div className="text-2xl font-bold text-white">
            {formatMinutes(total_minutes)}
          </div>
        </div>

        <div className="grid grid-cols-2 gap-4">
          <div>
            <div className="text-xs uppercase tracking-wider text-slate-400 mb-1">Shows/Movies</div>
            <div className="text-xl font-bold text-white">{total_shows.toLocaleString()}</div>
          </div>

          {last_watched && (
            <div>
              <div className="text-xs uppercase tracking-wider text-slate-400 mb-1">Last Watched</div>
              <div className="text-sm font-medium text-slate-300">
                {new Date(last_watched).toLocaleDateString('en-US', { month: 'short', day: 'numeric' })}
              </div>
            </div>
          )}
        </div>
      </div>

      <div className="mt-6 text-sm font-medium text-blue-400">
        View details →
      </div>
    </Link>
  );
};

export default ServiceCard;
