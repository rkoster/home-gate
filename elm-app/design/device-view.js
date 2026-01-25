import React, { useState, useMemo } from 'react';
import {
  Clock,
  Battery,
  Activity,
  AlertCircle,
  ChevronLeft,
  ChevronRight,
  Calendar,
  Zap,
  CheckCircle2,
  XCircle
} from 'lucide-react';

// --- Mock Data Generator ---
// Generates 96 intervals (24 hours * 4 intervals per hour)
const generateDailyData = () => {
  const intervals = [];
  // Random quota limit (e.g., 12 hours max)
  const quotaLimitMinutes = 12 * 60;
  let usedMinutes = 0;

  for (let i = 0; i < 96; i++) {
    // Simulating usage: distinct patterns (e.g., active during day, off at night)
    const hour = Math.floor(i / 4);
    let isActive = false;

    // Simulate "business hours" activity + random bursts
    if (hour >= 8 && hour <= 18) {
      isActive = Math.random() > 0.3;
    } else if (hour >= 19 && hour <= 23) {
      isActive = Math.random() > 0.6;
    }

    if (isActive) usedMinutes += 15;

    intervals.push({
      id: i,
      hour: hour,
      quarter: i % 4, // 0, 1, 2, 3
      isActive: isActive,
      timestamp: `${String(hour).padStart(2, '0')}:${String((i % 4) * 15).padStart(2, '0')}`
    });
  }

  return { intervals, quotaLimitMinutes, usedMinutes };
};

const ActivityBlock = ({ interval }) => {
  const [showTooltip, setShowTooltip] = useState(false);

  // Helper to format end time for tooltip
  const getEndTime = (startStr) => {
    const [h, m] = startStr.split(':').map(Number);
    const date = new Date();
    date.setHours(h, m + 15);
    return `${String(date.getHours()).padStart(2, '0')}:${String(date.getMinutes()).padStart(2, '0')}`;
  };

  const statusColor = interval.isActive
    ? 'bg-emerald-500 hover:bg-emerald-400 shadow-[0_0_8px_rgba(16,185,129,0.4)]'
    : 'bg-slate-200 dark:bg-slate-700 hover:bg-slate-300 dark:hover:bg-slate-600';

  return (
    <div
      className="relative group flex-1 h-full"
      onMouseEnter={() => setShowTooltip(true)}
      onMouseLeave={() => setShowTooltip(false)}
    >
      <div
        className={`w-full h-8 rounded-sm transition-all duration-200 cursor-pointer ${statusColor} ${interval.isActive ? 'scale-y-90 group-hover:scale-y-100' : 'scale-y-75 group-hover:scale-y-90'}`}
      ></div>

      {/* Tooltip */}
      {showTooltip && (
        <div className="absolute bottom-full left-1/2 -translate-x-1/2 mb-2 z-20 w-max">
          <div className="bg-slate-900 text-white text-xs py-1.5 px-3 rounded shadow-lg flex flex-col items-center">
            <span className="font-bold">{interval.timestamp} - {getEndTime(interval.timestamp)}</span>
            <span className={`text-[10px] uppercase tracking-wider font-semibold ${interval.isActive ? 'text-emerald-400' : 'text-slate-400'}`}>
              {interval.isActive ? 'Active' : 'Inactive'}
            </span>
            {/* Tiny Arrow */}
            <div className="absolute top-full left-1/2 -translate-x-1/2 border-4 border-transparent border-t-slate-900"></div>
          </div>
        </div>
      )}
    </div>
  );
};

export default function DeviceActivityPage() {
  const [currentDate, setCurrentDate] = useState(new Date());
  // Memoize data so it doesn't regenerate on every render, but would change with "date" in a real app
  const data = useMemo(() => generateDailyData(), [currentDate]);

  // Calculations
  const usagePercentage = Math.round((data.usedMinutes / data.quotaLimitMinutes) * 100);
  const isOverQuota = data.usedMinutes > data.quotaLimitMinutes;

  // Group intervals by hour for the grid view
  const hours = Array.from({ length: 24 }, (_, i) => {
    return {
      hourLabel: i,
      intervals: data.intervals.slice(i * 4, (i + 1) * 4)
    };
  });

  const formatTime = (mins) => {
    const h = Math.floor(mins / 60);
    const m = mins % 60;
    return `${h}h ${m}m`;
  };

  return (
    <div className="min-h-screen bg-slate-50 text-slate-900 font-sans p-4 md:p-8">
      <div className="max-w-5xl mx-auto space-y-6">

        {/* Header Section */}
        <div className="flex flex-col md:flex-row md:items-center justify-between gap-4">
          <div>
            <h1 className="text-2xl font-bold flex items-center gap-2 text-slate-800">
              <Activity className="w-6 h-6 text-indigo-600" />
              Device Monitor
            </h1>
            <p className="text-slate-500 text-sm mt-1">Activity logs and quota management for <span className="font-semibold text-slate-700">Sensor-X14</span></p>
          </div>

          <div className="flex items-center bg-white p-1.5 rounded-lg border border-slate-200 shadow-sm">
            <button
              onClick={() => setCurrentDate(new Date(currentDate.setDate(currentDate.getDate() - 1)))}
              className="p-1 hover:bg-slate-100 rounded-md text-slate-500 transition-colors"
            >
              <ChevronLeft size={20} />
            </button>
            <div className="flex items-center gap-2 px-4 py-1 border-x border-slate-100 mx-1">
              <Calendar size={16} className="text-slate-400" />
              <span className="text-sm font-medium text-slate-700 whitespace-nowrap">
                {currentDate.toLocaleDateString('en-US', { weekday: 'short', month: 'short', day: 'numeric' })}
              </span>
            </div>
            <button
              onClick={() => setCurrentDate(new Date(currentDate.setDate(currentDate.getDate() + 1)))}
              className="p-1 hover:bg-slate-100 rounded-md text-slate-500 transition-colors"
            >
              <ChevronRight size={20} />
            </button>
          </div>
        </div>

        {/* Quota & Stats Cards */}
        <div className="grid grid-cols-1 md:grid-cols-3 gap-6">

          {/* Main Quota Card */}
          <div className="md:col-span-2 bg-white rounded-2xl p-6 shadow-sm border border-slate-100 relative overflow-hidden">
            <div className="flex justify-between items-start mb-6">
              <div>
                <h2 className="text-lg font-semibold text-slate-800">Daily Quota Usage</h2>
                <p className="text-slate-500 text-xs">Resets at 00:00 UTC</p>
              </div>
              <div className={`px-3 py-1 rounded-full text-xs font-bold border ${isOverQuota ? 'bg-red-50 text-red-600 border-red-100' : 'bg-emerald-50 text-emerald-600 border-emerald-100'}`}>
                {isOverQuota ? 'QUOTA EXCEEDED' : 'WITHIN LIMITS'}
              </div>
            </div>

            <div className="relative pt-2 pb-6">
              <div className="flex justify-between text-sm font-medium mb-2">
                <span className="text-slate-600">{formatTime(data.usedMinutes)} used</span>
                <span className="text-slate-400">{formatTime(data.quotaLimitMinutes)} limit</span>
              </div>

              {/* Progress Bar Container */}
              <div className="h-4 w-full bg-slate-100 rounded-full overflow-hidden">
                <div
                  className={`h-full transition-all duration-1000 ease-out rounded-full ${isOverQuota ? 'bg-gradient-to-r from-red-500 to-red-600' : 'bg-gradient-to-r from-indigo-500 to-purple-500'}`}
                  style={{ width: `${Math.min(usagePercentage, 100)}%` }}
                ></div>
              </div>

              {/* Markers for 50% and 75% */}
              <div className="absolute top-8 left-1/2 w-px h-2 bg-slate-300 transform -translate-x-1/2"></div>
              <div className="absolute top-8 left-3/4 w-px h-2 bg-slate-300 transform -translate-x-1/2"></div>
            </div>

            {/* Background Decoration */}
            <div className="absolute -right-6 -bottom-6 opacity-5 pointer-events-none">
              <Zap size={120} />
            </div>
          </div>

          {/* Quick Stats */}
          <div className="bg-white rounded-2xl p-6 shadow-sm border border-slate-100 flex flex-col justify-center space-y-4">
            <div className="flex items-center gap-4">
              <div className="p-3 bg-emerald-50 rounded-xl">
                <CheckCircle2 className="w-6 h-6 text-emerald-600" />
              </div>
              <div>
                <p className="text-xs text-slate-500 font-medium uppercase tracking-wide">Active Time</p>
                <p className="text-xl font-bold text-slate-800">{formatTime(data.usedMinutes)}</p>
              </div>
            </div>
            <div className="w-full h-px bg-slate-100"></div>
            <div className="flex items-center gap-4">
              <div className="p-3 bg-slate-50 rounded-xl">
                <XCircle className="w-6 h-6 text-slate-400" />
              </div>
              <div>
                <p className="text-xs text-slate-500 font-medium uppercase tracking-wide">Idle Time</p>
                <p className="text-xl font-bold text-slate-800">{formatTime((24 * 60) - data.usedMinutes)}</p>
              </div>
            </div>
          </div>
        </div>

        {/* The 15-Minute Interval Grid */}
        <div className="bg-white rounded-2xl shadow-sm border border-slate-100 overflow-hidden">
          <div className="p-6 border-b border-slate-100 flex justify-between items-center">
             <h3 className="font-semibold text-slate-800 flex items-center gap-2">
               <Clock className="w-4 h-4 text-slate-400" />
               Activity Timeline (15m Intervals)
             </h3>
             <div className="flex gap-4 text-xs font-medium text-slate-500">
               <div className="flex items-center gap-2">
                 <div className="w-3 h-3 bg-emerald-500 rounded-sm"></div> Active
               </div>
               <div className="flex items-center gap-2">
                 <div className="w-3 h-3 bg-slate-200 rounded-sm"></div> Idle
               </div>
             </div>
          </div>

          <div className="p-6">
            {/* Grid Layout: 24 Hour blocks */}
            <div className="grid grid-cols-2 xs:grid-cols-3 sm:grid-cols-4 md:grid-cols-6 lg:grid-cols-8 gap-4">
              {hours.map((hourBlock) => (
                <div key={hourBlock.hourLabel} className="bg-slate-50/50 rounded-lg p-3 border border-slate-100/50 hover:border-slate-200 transition-colors">
                  {/* Hour Label */}
                  <div className="text-xs font-medium text-slate-400 mb-2 flex justify-between">
                    <span>{String(hourBlock.hourLabel).padStart(2, '0')}:00</span>
                  </div>

                  {/* The 4 quarters */}
                  <div className="flex items-end h-8 gap-1">
                    {hourBlock.intervals.map((interval) => (
                      <ActivityBlock key={interval.id} interval={interval} />
                    ))}
                  </div>

                  {/* Quarter labels (optional/subtle) */}
                  <div className="flex justify-between text-[8px] text-slate-300 mt-1 px-[1px]">
                    <span>00</span>
                    <span>30</span>
                  </div>
                </div>
              ))}
            </div>
          </div>
        </div>

        {/* Footer info */}
        <div className="flex items-center justify-center gap-2 text-xs text-slate-400 py-4">
          <AlertCircle size={12} />
          <span>Data is synced every 15 minutes. Last sync: Just now.</span>
        </div>

      </div>
    </div>
  );
}
