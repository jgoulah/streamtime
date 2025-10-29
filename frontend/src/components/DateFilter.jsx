import { useState, useEffect } from 'react';

const DateFilter = ({ onFilterChange }) => {
  const [filterType, setFilterType] = useState('month'); // 'all', 'year', 'month', 'day'
  const [selectedYear, setSelectedYear] = useState(new Date().getFullYear());
  const [selectedMonth, setSelectedMonth] = useState(new Date().getMonth() + 1);
  const [selectedDay, setSelectedDay] = useState(new Date().getDate());

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

  // Notify parent component of filter changes
  useEffect(() => {
    const params = {};

    if (filterType === 'year') {
      params.year = selectedYear;
    } else if (filterType === 'month') {
      params.year = selectedYear;
      params.month = selectedMonth;
    } else if (filterType === 'day') {
      params.year = selectedYear;
      params.month = selectedMonth;
      params.day = selectedDay;
    }

    onFilterChange({ filterType, params });
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [filterType, selectedYear, selectedMonth, selectedDay]);

  const getFilterLabel = () => {
    if (filterType === 'all') return 'All Time';
    if (filterType === 'year') return `Year: ${selectedYear}`;
    const monthLabel = months.find(m => m.value === selectedMonth)?.label;
    if (filterType === 'month') return `${monthLabel} ${selectedYear}`;
    return `${monthLabel} ${selectedDay}, ${selectedYear}`;
  };

  return (
    <div className="bg-slate-800 p-6 rounded-lg border border-slate-700 shadow-xl">
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
            <button
              onClick={() => setFilterType('day')}
              className={`px-4 py-2 rounded-md font-medium transition-colors ${
                filterType === 'day'
                  ? 'bg-blue-600 text-white'
                  : 'bg-slate-700 text-slate-300 hover:bg-slate-600'
              }`}
            >
              Day
            </button>
          </div>
        </div>

        {(filterType === 'year' || filterType === 'month' || filterType === 'day') && (
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

        {(filterType === 'month' || filterType === 'day') && (
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

        {filterType === 'day' && (
          <div className="flex items-center gap-2">
            <label className="text-slate-300 text-sm font-medium">Day:</label>
            <select
              value={selectedDay}
              onChange={(e) => setSelectedDay(parseInt(e.target.value))}
              className="bg-slate-700 text-white px-4 py-2 rounded-md border border-slate-600 focus:border-blue-500 focus:outline-none"
            >
              {Array.from({ length: new Date(selectedYear, selectedMonth, 0).getDate() }, (_, i) => i + 1).map((day) => (
                <option key={day} value={day}>
                  {day}
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
  );
};

export default DateFilter;
