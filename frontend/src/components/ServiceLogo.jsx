const ServiceLogo = ({ serviceName, size = 'md', showFullTextFallback = false }) => {
  const sizeClasses = {
    sm: 'w-6 h-6',
    md: 'w-10 h-10',
    lg: 'w-16 h-16',
    xl: 'w-20 h-20',
  };

  // Map service names to local logo files
  const logoFiles = {
    'Netflix': '/logos/netflix.svg',
    'YouTube TV': '/logos/youtube-tv.svg',
    'YouTube': '/logos/youtube.svg',
    'Amazon Video': '/logos/amazon.svg',
    'Prime Video': '/logos/amazon.svg',
    'HBO Max': '/logos/hbo.svg',
    'Apple TV+': '/logos/apple-tv.svg',
    'Peacock': '/logos/peacock.svg',
    'Hulu': '/logos/hulu.svg',
    'Disney+': '/logos/disney.svg',
  };

  const logoPath = logoFiles[serviceName];

  // Fallback if logo not found
  if (!logoPath) {
    if (showFullTextFallback) {
      return (
        <div className="text-2xl font-bold text-white">
          {serviceName || 'Unknown Service'}
        </div>
      );
    }
    return (
      <div className={`${sizeClasses[size]} bg-slate-700 rounded-lg flex items-center justify-center text-white font-bold text-xl`}>
        {serviceName ? serviceName.charAt(0) : '?'}
      </div>
    );
  }

  const handleError = (e) => {
    console.error('Failed to load logo:', logoPath, 'for service:', serviceName);
    // Replace with fallback
    e.target.style.display = 'none';
    const fallback = document.createElement('div');
    if (showFullTextFallback) {
      fallback.className = 'text-2xl font-bold text-white';
      fallback.textContent = serviceName || 'Unknown Service';
    } else {
      fallback.className = `${sizeClasses[size]} bg-slate-700 rounded-lg flex items-center justify-center text-white font-bold text-xl`;
      fallback.textContent = serviceName ? serviceName.charAt(0) : '?';
    }
    e.target.parentElement.appendChild(fallback);
  };

  return (
    <div className="inline-block">
      <img
        src={logoPath}
        alt={`${serviceName} logo`}
        className={`${sizeClasses[size]} object-contain`}
        onError={handleError}
      />
    </div>
  );
};

export default ServiceLogo;
